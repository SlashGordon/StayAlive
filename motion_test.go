package main

import (
	"math/rand/v2"
	"testing"
	"time"
)

func newMotion(seed uint64) motion {
	return motion{rng: rand.New(rand.NewPCG(seed, seed))}
}

func TestStrokeEndsOnTarget(t *testing.T) {
	for seed := range uint64(50) {
		m := newMotion(seed)
		from, to := Position{500, 500}, Position{620, 430}
		steps := m.stroke(from, to)
		if got := steps[len(steps)-1].Pos; got != to {
			t.Fatalf("seed %d: stroke ended at %v, want %v", seed, got, to)
		}
	}
}

func TestStrokeSpeedProfile(t *testing.T) {
	m := newMotion(1)
	steps := m.stroke(Position{0, 0}, Position{400, 0})
	n := len(steps)
	if n < 10 {
		t.Fatalf("only %d steps for a 400px stroke", n)
	}
	// Slow start, fast middle, slow end.
	first := distance(Position{0, 0}, steps[0].Pos)
	mid := distance(steps[n/2-1].Pos, steps[n/2].Pos)
	last := distance(steps[n-2].Pos, steps[n-1].Pos)
	if mid <= first || mid <= last {
		t.Errorf("step lengths first=%.1f mid=%.1f last=%.1f, want the middle fastest", first, mid, last)
	}
}

func TestStrokeHasNoDuplicatePoints(t *testing.T) {
	m := newMotion(2)
	steps := m.stroke(Position{0, 0}, Position{5, 0})
	for i := 1; i < len(steps); i++ {
		if steps[i].Pos == steps[i-1].Pos {
			t.Fatalf("step %d repeats position %v", i, steps[i].Pos)
		}
	}
}

func TestStrokeDurationGrowsWithDistance(t *testing.T) {
	total := func(steps []Step) (d time.Duration) {
		for _, s := range steps {
			d += s.Wait
		}
		return d
	}
	var short, long time.Duration
	for seed := range uint64(20) {
		short += total(newMotion(seed).stroke(Position{0, 0}, Position{20, 0}))
		long += total(newMotion(seed).stroke(Position{0, 0}, Position{600, 0}))
	}
	if long <= short {
		t.Errorf("600px strokes took %s, 20px strokes %s", long, short)
	}
}

func TestGestureReturnsNearStart(t *testing.T) {
	for seed := range uint64(50) {
		m := newMotion(seed)
		start := Position{800, 400}
		steps := m.gesture(start, Position{880, 350})
		end := steps[len(steps)-1].Pos
		if abs(end.X-start.X) > homeJitter || abs(end.Y-start.Y) > homeJitter {
			t.Fatalf("seed %d: gesture ended at %v, too far from %v", seed, end, start)
		}
	}
}

func TestRectClamp(t *testing.T) {
	r := Rect{X: -100, Y: 0, W: 200, H: 100}
	if got := r.Clamp(Position{-500, 50}); got != (Position{-100, 50}) {
		t.Errorf("Clamp left = %v", got)
	}
	if got := r.Clamp(Position{500, 500}); got != (Position{99, 99}) {
		t.Errorf("Clamp bottom right = %v", got)
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
