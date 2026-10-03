package main

import (
	"context"
	"log"
	"math"
	"math/rand/v2"
	"time"
)

const (
	// moveTolerance is how far (in pixels) the cursor may differ from where we
	// left it before we count it as a move by the user.
	moveTolerance = 3
	// ownInputSlack covers the gap between posting our last event and taking
	// the timestamp, so our own moves never look like user input.
	ownInputSlack = 200 * time.Millisecond
)

// Config controls when and how far the jiggler moves the cursor.
type Config struct {
	IdleAfter time.Duration // user idle time before the first move
	Interval  time.Duration // longest pause between two moves while the user stays idle
	Radius    int           // maximum distance of a move in pixels
	Poll      time.Duration // how often to check for user activity
}

// Jiggler moves the mouse while the user is idle and stops as soon as the
// user touches the mouse, trackpad or keyboard.
type Jiggler struct {
	cfg       Config
	mouse     Mouse
	log       *log.Logger
	rng       *rand.Rand
	motion    motion
	now       func() time.Time
	lastInput func(now time.Time) (time.Time, bool)
	sleep     func(ctx context.Context, d time.Duration) bool

	lastUserInput time.Time     // last input we attribute to the user
	lastOwnInput  time.Time     // last cursor move we made ourselves
	lastPos       Position      // where the cursor was at the last check
	gap           time.Duration // pause before our next move
}

func NewJiggler(cfg Config, mouse Mouse, logger *log.Logger, rng *rand.Rand) *Jiggler {
	return &Jiggler{
		cfg:       cfg,
		mouse:     mouse,
		log:       logger,
		rng:       rng,
		motion:    motion{rng: rng},
		now:       time.Now,
		lastInput: lastSystemInput,
		sleep:     sleepCtx,
	}
}

// Run blocks until ctx is cancelled. The user counts as active at start, so
// the first move happens after cfg.IdleAfter without input.
func (j *Jiggler) Run(ctx context.Context) {
	j.lastUserInput = j.now()
	j.lastPos = j.mouse.Location()

	ticker := time.NewTicker(j.cfg.Poll)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.tick(ctx)
		}
	}
}

func (j *Jiggler) tick(ctx context.Context) {
	now := j.now()
	j.observe(now)
	if now.Sub(j.lastUserInput) < j.cfg.IdleAfter || now.Sub(j.lastOwnInput) < j.gap {
		return
	}
	j.jiggle(ctx)
	// A person does not move the mouse on a fixed beat. Stay below the
	// configured interval so the screen lock never wins.
	j.gap = time.Duration(float64(j.cfg.Interval) * (0.5 + 0.5*j.rng.Float64()))
}

// observe records user activity since the last check and reports whether
// there was any.
func (j *Jiggler) observe(now time.Time) bool {
	active := false

	if pos := j.mouse.Location(); distance(pos, j.lastPos) > moveTolerance {
		j.lastPos = pos
		j.lastUserInput = now
		active = true
	}

	// The system clock also sees keystrokes and clicks, but it sees our own
	// moves too. Input that happened after our last move is the user's.
	if t, ok := j.lastInput(now); ok && t.After(j.lastOwnInput.Add(ownInputSlack)) && t.After(j.lastUserInput) {
		j.lastUserInput = t
		active = true
	}

	return active
}

// jiggle moves the cursor to a random nearby point and back. It stops early
// when the user becomes active.
func (j *Jiggler) jiggle(ctx context.Context) {
	start := j.lastPos
	display, onDisplay := j.mouse.Display(start)
	steps := j.motion.gesture(start, j.randomTarget(start))

	j.log.Printf("idle for %s, moving cursor", j.now().Sub(j.lastUserInput).Round(time.Second))
	for _, s := range steps {
		if j.observe(j.now()) {
			j.log.Print("user activity detected, pausing")
			return
		}
		p := s.Pos
		if onDisplay {
			p = display.Clamp(p)
		}
		j.mouse.Move(p)
		j.lastPos = p
		j.lastOwnInput = j.now()

		if !j.sleep(ctx, s.Wait) {
			return
		}
	}
}

// randomTarget picks a point in a random direction. Most moves are short
// nudges, some go up to the full radius.
func (j *Jiggler) randomTarget(start Position) Position {
	angle := j.rng.Float64() * 2 * math.Pi
	f := j.rng.Float64()
	dist := float64(j.cfg.Radius) * (0.15 + 0.85*f*f)
	return Position{
		X: start.X + int(math.Round(dist*math.Cos(angle))),
		Y: start.Y + int(math.Round(dist*math.Sin(angle))),
	}
}

// sleepCtx waits for d and reports false when ctx ends first.
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
