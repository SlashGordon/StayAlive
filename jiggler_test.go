package main

import (
	"context"
	"io"
	"log"
	"math/rand/v2"
	"testing"
	"time"
)

type fakeMouse struct {
	pos    Position
	moves  []Position
	onMove func(step int)
}

func (m *fakeMouse) Location() Position { return m.pos }

func (m *fakeMouse) Move(p Position) {
	m.pos = p
	m.moves = append(m.moves, p)
	if m.onMove != nil {
		m.onMove(len(m.moves))
	}
}

func (m *fakeMouse) Display(Position) (Rect, bool) {
	return Rect{W: 1920, H: 1080}, true
}

type fixture struct {
	j         *Jiggler
	mouse     *fakeMouse
	now       time.Time
	lastInput time.Time // what the OS reports as the last input
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	f := &fixture{
		mouse: &fakeMouse{pos: Position{500, 500}},
		now:   time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC),
	}
	f.lastInput = f.now
	cfg := Config{IdleAfter: 30 * time.Second, Interval: 30 * time.Second, Radius: 100, Poll: time.Second}
	f.j = NewJiggler(cfg, f.mouse, log.New(io.Discard, "", 0), rand.New(rand.NewPCG(1, 2)))
	f.j.now = func() time.Time { return f.now }
	f.j.lastInput = func(time.Time) (time.Time, bool) { return f.lastInput, true }
	f.j.sleep = func(context.Context, time.Duration) bool { return true }

	// Same start state as Run.
	f.j.lastUserInput = f.now
	f.j.lastPos = f.mouse.Location()
	return f
}

func (f *fixture) advance(d time.Duration) {
	f.now = f.now.Add(d)
	f.j.tick(context.Background())
}

func TestWaitsForIdleTimeAfterStart(t *testing.T) {
	f := newFixture(t)
	f.advance(29 * time.Second)
	if len(f.mouse.moves) != 0 {
		t.Fatalf("moved after 29s, want no move before 30s")
	}
	f.advance(time.Second)
	if len(f.mouse.moves) == 0 {
		t.Fatalf("no move after 30s idle")
	}
}

func TestJiggleReturnsNearStart(t *testing.T) {
	f := newFixture(t)
	f.advance(30 * time.Second)
	if distance(f.mouse.pos, Position{500, 500}) > 2*homeJitter {
		t.Errorf("cursor ended at %v, want start position", f.mouse.pos)
	}
}

func TestCursorMoveResetsIdleTime(t *testing.T) {
	f := newFixture(t)
	f.advance(20 * time.Second)
	f.mouse.pos = Position{800, 300} // user moves the mouse
	f.advance(time.Second)           // noticed at the next poll
	f.advance(29 * time.Second)
	if len(f.mouse.moves) != 0 {
		t.Fatalf("moved 29s after user activity")
	}
	f.advance(time.Second)
	if len(f.mouse.moves) == 0 {
		t.Fatalf("no move 30s after user activity")
	}
}

func TestKeyboardInputResetsIdleTime(t *testing.T) {
	f := newFixture(t)
	f.advance(20 * time.Second)
	f.lastInput = f.now // user types, cursor stays
	f.advance(20 * time.Second)
	if len(f.mouse.moves) != 0 {
		t.Fatalf("moved 20s after keyboard input")
	}
}

func TestOwnMovesDoNotCountAsUserInput(t *testing.T) {
	f := newFixture(t)
	// The OS sees our moves as input, so the reported last input follows them.
	f.mouse.onMove = func(int) { f.lastInput = f.now }

	f.advance(30 * time.Second)
	first := len(f.mouse.moves)
	if first == 0 {
		t.Fatal("no first move")
	}
	f.advance(30 * time.Second)
	if len(f.mouse.moves) == first {
		t.Fatal("second move missing, own moves were taken for user input")
	}
}

func TestIntervalBetweenMoves(t *testing.T) {
	f := newFixture(t)
	f.advance(30 * time.Second)
	first := len(f.mouse.moves)
	f.advance(14 * time.Second)
	if len(f.mouse.moves) != first {
		t.Fatal("moved again before half the interval passed")
	}
	f.advance(16 * time.Second)
	if len(f.mouse.moves) == first {
		t.Fatal("no second move within the interval")
	}
}

func TestUserTakesOverDuringJiggle(t *testing.T) {
	f := newFixture(t)
	f.mouse.onMove = func(step int) {
		if step == 3 {
			f.mouse.pos = Position{100, 100} // user grabs the mouse
		}
	}
	f.advance(30 * time.Second)
	if n := len(f.mouse.moves); n != 3 {
		t.Fatalf("made %d moves, want stop after the user moved at step 3", n)
	}
	if f.j.lastUserInput != f.now {
		t.Errorf("user activity not recorded")
	}
}
