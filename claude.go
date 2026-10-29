package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const (
	// claudeCheck is how often the transcripts are read.
	claudeCheck = 10 * time.Second
	// claudeGrace is how long no session may be working before we stop. It
	// bridges the moment between a background agent finishing and its result
	// starting a new turn in the main session.
	claudeGrace = time.Minute
)

// errClaudeDone is the context cause when all Claude Code sessions are done.
var errClaudeDone = errors.New("all Claude Code sessions are done")

// finishedWorkflow lists workflow states that end a run.
var finishedWorkflow = []string{"completed", "failed", "cancelled", "canceled", "killed", "stopped"}

// claudeProjectsDir returns where Claude Code keeps its session transcripts.
func claudeProjectsDir() (string, error) {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return filepath.Join(dir, "projects"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "projects"), nil
}

// untilClaudeDone returns a context that ends with errClaudeDone once no
// Claude Code session has been working for claudeGrace.
func untilClaudeDone(ctx context.Context, w *claudeWatcher, logger *log.Logger) context.Context {
	ctx, cancel := context.WithCancelCause(ctx)
	go func() {
		ticker := time.NewTicker(claudeCheck)
		defer ticker.Stop()
		var last []string
		idleSince := time.Now()
		for first := true; ; first = false {
			busy := w.busy()
			if first || !slices.Equal(busy, last) {
				if len(busy) == 0 {
					logger.Print("no Claude session is working")
				} else {
					logger.Printf("Claude sessions working: %s", strings.Join(busy, ", "))
				}
				last = busy
			}
			if len(busy) > 0 {
				idleSince = time.Now()
			} else if time.Since(idleSince) >= claudeGrace {
				cancel(errClaudeDone)
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return ctx
}

// claudeWatcher tells from the transcripts Claude Code writes whether any of
// its sessions is still working. A session works while its last turn is open,
// a background agent has not finished or a workflow is still running.
type claudeWatcher struct {
	root  string        // the projects directory with one folder per working directory
	quiet time.Duration // a session that wrote nothing for this long counts as done
	now   func() time.Time

	turns map[string]turnCache // last result per transcript, so unchanged files are not parsed again
}

type turnCache struct {
	size    int64
	modTime time.Time
	open    bool
}

func newClaudeWatcher(root string, quiet time.Duration) *claudeWatcher {
	return &claudeWatcher{root: root, quiet: quiet, now: time.Now, turns: map[string]turnCache{}}
}

// busy returns a label for every session that is working, sorted.
func (w *claudeWatcher) busy() []string {
	cutoff := w.now().Add(-w.quiet)
	projects, _ := os.ReadDir(w.root)
	var busy []string
	for _, p := range projects {
		if !p.IsDir() {
			continue
		}
		dir := filepath.Join(w.root, p.Name())
		files, _ := os.ReadDir(dir)
		for _, f := range files {
			id, ok := strings.CutSuffix(f.Name(), ".jsonl")
			if !ok || f.IsDir() {
				continue
			}
			if reason := w.session(filepath.Join(dir, f.Name()), filepath.Join(dir, id), cutoff); reason != "" {
				busy = append(busy, p.Name()+" "+id[:min(8, len(id))]+" ("+reason+")")
			}
		}
	}
	slices.Sort(busy)
	return busy
}

// session returns why the session with the given transcript and data
// directory is working, or "" when it is done.
func (w *claudeWatcher) session(transcript, dir string, cutoff time.Time) string {
	if backgroundAgentRunning(filepath.Join(dir, "subagents"), cutoff) {
		return "background agent"
	}
	if workflowRunning(dir, cutoff) {
		return "workflow"
	}

	info, err := os.Stat(transcript)
	if err != nil {
		return ""
	}
	// A foreground agent writes only to its own transcript while the main
	// one waits for its result.
	latest := info.ModTime()
	if t := newestModTime(filepath.Join(dir, "subagents")); t.After(latest) {
		latest = t
	}
	if latest.Before(cutoff) {
		return ""
	}
	if w.turnOpen(transcript, info) {
		return "working"
	}
	return ""
}

// backgroundAgentRunning reports whether an agent started in the background
// has not finished yet and wrote something after cutoff.
func backgroundAgentRunning(dir string, cutoff time.Time) bool {
	metas, _ := filepath.Glob(filepath.Join(dir, "agent-*.meta.json"))
	for _, path := range metas {
		var meta struct {
			RequestShape string `json:"requestShape"`
			Finished     bool   `json:"finished"`
		}
		data, err := os.ReadFile(path)
		if err != nil || json.Unmarshal(data, &meta) != nil {
			continue
		}
		if meta.RequestShape != "background" || meta.Finished {
			continue
		}
		transcript := strings.TrimSuffix(path, ".meta.json") + ".jsonl"
		if modTime(transcript).After(cutoff) || modTime(path).After(cutoff) {
			return true
		}
	}
	return false
}

// workflowRunning reports whether a workflow run has not reached a final
// state and it or one of its agents wrote something after cutoff. A run
// without a state file counts as running.
func workflowRunning(dir string, cutoff time.Time) bool {
	runs := map[string]bool{}
	states, _ := filepath.Glob(filepath.Join(dir, "workflows", "*.json"))
	for _, path := range states {
		runs[strings.TrimSuffix(filepath.Base(path), ".json")] = true
	}
	agentDirs, _ := os.ReadDir(filepath.Join(dir, "subagents", "workflows"))
	for _, d := range agentDirs {
		if d.IsDir() {
			runs[d.Name()] = true
		}
	}

	for run := range runs {
		state := filepath.Join(dir, "workflows", run+".json")
		var wf struct {
			Status string `json:"status"`
		}
		if data, err := os.ReadFile(state); err == nil && json.Unmarshal(data, &wf) == nil && slices.Contains(finishedWorkflow, wf.Status) {
			continue
		}
		if modTime(state).After(cutoff) || newestModTime(filepath.Join(dir, "subagents", "workflows", run)).After(cutoff) {
			return true
		}
	}
	return false
}

// turnOpen reports whether the last message in a transcript leaves the turn
// open: a tool call waiting for its result, or a prompt or tool result
// waiting for the model's answer.
func (w *claudeWatcher) turnOpen(path string, info fs.FileInfo) bool {
	if c, ok := w.turns[path]; ok && c.size == info.Size() && c.modTime.Equal(info.ModTime()) {
		return c.open
	}
	open := lastTurnOpen(path)
	w.turns[path] = turnCache{size: info.Size(), modTime: info.ModTime(), open: open}
	return open
}

type transcriptEntry struct {
	Type        string `json:"type"`
	IsSidechain bool   `json:"isSidechain"`
	Message     struct {
		StopReason string          `json:"stop_reason"`
		Content    json.RawMessage `json:"content"`
	} `json:"message"`
}

func lastTurnOpen(path string) bool {
	e, ok := lastMessage(path)
	if !ok {
		return false
	}
	if e.Type == "assistant" {
		switch e.Message.StopReason {
		case "", "tool_use", "pause_turn":
			return true
		}
		return false
	}
	return !bytes.Contains(e.Message.Content, []byte("[Request interrupted by user"))
}

// lastMessage returns the last user or assistant entry of the main
// conversation. It reads the file from the end and widens the window until
// it finds one, because single lines can be megabytes long.
func lastMessage(path string) (transcriptEntry, bool) {
	f, err := os.Open(path)
	if err != nil {
		return transcriptEntry{}, false
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return transcriptEntry{}, false
	}

	size := info.Size()
	for window := int64(256 << 10); ; window *= 4 {
		start := max(size-window, 0)
		buf := make([]byte, size-start)
		if _, err := f.ReadAt(buf, start); err != nil {
			return transcriptEntry{}, false
		}
		lines := bytes.Split(buf, []byte("\n"))
		if start > 0 {
			lines = lines[1:] // the first line is cut off
		}
		for i := len(lines) - 1; i >= 0; i-- {
			var e transcriptEntry
			if json.Unmarshal(lines[i], &e) != nil || e.IsSidechain {
				continue
			}
			if e.Type == "user" || e.Type == "assistant" {
				return e, true
			}
		}
		if start == 0 {
			return transcriptEntry{}, false
		}
	}
}

func modTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// newestModTime returns the latest modification time of any file below dir.
func newestModTime(dir string) time.Time {
	var newest time.Time
	filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil && info.ModTime().After(newest) {
			newest = info.ModTime()
		}
		return nil
	})
	return newest
}
