package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

var claudeNow = time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)

const (
	promptLine    = `{"type":"user","message":{"role":"user","content":"fix the build"}}`
	toolUseLine   = `{"type":"assistant","message":{"role":"assistant","stop_reason":"tool_use","content":[{"type":"tool_use"}]}}`
	toolDoneLine  = `{"type":"user","message":{"role":"user","content":[{"type":"tool_result"}]}}`
	endTurnLine   = `{"type":"assistant","message":{"role":"assistant","stop_reason":"end_turn","content":[{"type":"text","text":"Done."}]}}`
	interruptLine = `{"type":"user","message":{"role":"user","content":[{"type":"text","text":"[Request interrupted by user for tool use]"}]}}`
	limitLine     = `{"type":"assistant","message":{"role":"assistant","stop_reason":"stop_sequence","content":[{"type":"text","text":"You've hit your limit"}]}}`
	metaLine      = `{"type":"last-prompt"}`
)

// writeFile creates path with content and sets its modification time to age
// before claudeNow.
func writeFile(t *testing.T, path, content string, age time.Duration) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	mtime := claudeNow.Add(-age)
	if err := os.Chtimes(path, mtime, mtime); err != nil {
		t.Fatal(err)
	}
}

func transcript(lines ...string) string {
	return strings.Join(lines, "\n") + "\n"
}

func newTestWatcher(t *testing.T) (*claudeWatcher, string) {
	t.Helper()
	root := t.TempDir()
	w := newClaudeWatcher(root, 15*time.Minute)
	w.now = func() time.Time { return claudeNow }
	return w, filepath.Join(root, "-Users-me-project")
}

func TestClaudeTurnState(t *testing.T) {
	for name, tc := range map[string]struct {
		lines []string
		age   time.Duration
		busy  bool
	}{
		"prompt sent":          {[]string{promptLine}, time.Minute, true},
		"tool running":         {[]string{promptLine, toolUseLine, metaLine}, time.Minute, true},
		"tool result returned": {[]string{promptLine, toolUseLine, toolDoneLine}, time.Minute, true},
		"turn ended":           {[]string{promptLine, toolUseLine, toolDoneLine, endTurnLine, metaLine}, time.Minute, false},
		"interrupted":          {[]string{promptLine, toolUseLine, interruptLine}, time.Minute, false},
		"usage limit":          {[]string{promptLine, limitLine}, time.Minute, false},
		"quiet too long":       {[]string{promptLine, toolUseLine}, 20 * time.Minute, false},
		"no messages":          {[]string{metaLine}, time.Minute, false},
	} {
		t.Run(name, func(t *testing.T) {
			w, project := newTestWatcher(t)
			writeFile(t, filepath.Join(project, "abcdef1234.jsonl"), transcript(tc.lines...), tc.age)
			busy := w.busy()
			if got := len(busy) > 0; got != tc.busy {
				t.Errorf("busy = %v (%v), want %v", got, busy, tc.busy)
			}
		})
	}
}

func TestClaudeIgnoresSidechainEntries(t *testing.T) {
	w, project := newTestWatcher(t)
	sidechain := `{"type":"user","isSidechain":true,"message":{"role":"user","content":"sub task"}}`
	writeFile(t, filepath.Join(project, "s.jsonl"), transcript(promptLine, endTurnLine, sidechain), time.Minute)
	if busy := w.busy(); len(busy) > 0 {
		t.Errorf("busy = %v, want none", busy)
	}
}

func TestClaudeForegroundAgentKeepsSessionFresh(t *testing.T) {
	w, project := newTestWatcher(t)
	writeFile(t, filepath.Join(project, "s.jsonl"), transcript(promptLine, toolUseLine), time.Hour)
	writeFile(t, filepath.Join(project, "s", "subagents", "agent-a1.jsonl"), transcript(promptLine), time.Minute)
	writeFile(t, filepath.Join(project, "s", "subagents", "agent-a1.meta.json"), `{"requestShape":"foreground"}`, time.Hour)
	if busy := w.busy(); len(busy) != 1 {
		t.Errorf("busy = %v, want the session waiting for its agent", busy)
	}
}

func TestClaudeBackgroundAgent(t *testing.T) {
	for name, tc := range map[string]struct {
		meta string
		age  time.Duration
		busy bool
	}{
		"running":        {`{"requestShape":"background"}`, time.Minute, true},
		"finished":       {`{"requestShape":"background","finished":true}`, time.Minute, false},
		"quiet too long": {`{"requestShape":"background"}`, time.Hour, false},
	} {
		t.Run(name, func(t *testing.T) {
			w, project := newTestWatcher(t)
			writeFile(t, filepath.Join(project, "s.jsonl"), transcript(promptLine, endTurnLine), time.Hour)
			writeFile(t, filepath.Join(project, "s", "subagents", "agent-a1.meta.json"), tc.meta, time.Hour)
			writeFile(t, filepath.Join(project, "s", "subagents", "agent-a1.jsonl"), transcript(toolDoneLine), tc.age)
			busy := w.busy()
			if got := len(busy) > 0; got != tc.busy {
				t.Errorf("busy = %v (%v), want %v", got, busy, tc.busy)
			}
		})
	}
}

func TestClaudeWorkflow(t *testing.T) {
	for name, tc := range map[string]struct {
		state string // "" means no state file yet
		age   time.Duration
		busy  bool
	}{
		"no state yet":   {"", time.Minute, true},
		"running":        {`{"status":"running"}`, time.Minute, true},
		"completed":      {`{"status":"completed"}`, time.Minute, false},
		"quiet too long": {"", time.Hour, false},
	} {
		t.Run(name, func(t *testing.T) {
			w, project := newTestWatcher(t)
			writeFile(t, filepath.Join(project, "s.jsonl"), transcript(promptLine, endTurnLine), time.Hour)
			writeFile(t, filepath.Join(project, "s", "subagents", "workflows", "wf_1", "agent-a1.jsonl"), transcript(toolDoneLine), tc.age)
			if tc.state != "" {
				writeFile(t, filepath.Join(project, "s", "workflows", "wf_1.json"), tc.state, tc.age)
			}
			busy := w.busy()
			if got := len(busy) > 0; got != tc.busy {
				t.Errorf("busy = %v (%v), want %v", got, busy, tc.busy)
			}
		})
	}
}

func TestClaudeReadsPastLongLines(t *testing.T) {
	w, project := newTestWatcher(t)
	huge := `{"type":"user","message":{"role":"user","content":[{"type":"tool_result","content":"` + strings.Repeat("x", 3<<20) + `"}]}}`
	writeFile(t, filepath.Join(project, "s.jsonl"), transcript(promptLine, toolUseLine, huge, metaLine), time.Minute)
	if busy := w.busy(); len(busy) != 1 {
		t.Errorf("busy = %v, want the session with the long tool result", busy)
	}
}

func TestClaudeRereadsChangedTranscript(t *testing.T) {
	w, project := newTestWatcher(t)
	path := filepath.Join(project, "s.jsonl")
	writeFile(t, path, transcript(promptLine, toolUseLine), 2*time.Minute)
	if len(w.busy()) != 1 {
		t.Fatal("want busy while the tool runs")
	}
	writeFile(t, path, transcript(promptLine, toolUseLine, toolDoneLine, endTurnLine), time.Minute)
	if busy := w.busy(); len(busy) > 0 {
		t.Errorf("busy = %v after the turn ended", busy)
	}
}
