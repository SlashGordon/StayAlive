package main

import (
	"math"
	"math/rand/v2"
	"time"
)

// Position is a point in global screen coordinates.
type Position struct {
	X, Y int
}

// Rect is a display area in global screen coordinates.
type Rect struct {
	X, Y, W, H int
}

// Contains reports whether p lies inside r.
func (r Rect) Contains(p Position) bool {
	return p.X >= r.X && p.X < r.X+r.W && p.Y >= r.Y && p.Y < r.Y+r.H
}

// Clamp moves p to the nearest point inside r.
func (r Rect) Clamp(p Position) Position {
	return Position{
		X: min(max(p.X, r.X), r.X+r.W-1),
		Y: min(max(p.Y, r.Y), r.Y+r.H-1),
	}
}

// Step is one cursor event: move to Pos, then wait before the next one.
type Step struct {
	Pos  Position
	Wait time.Duration
}

const (
	frameTime       = 12 * time.Millisecond // a mouse reports roughly every 8-16 ms
	overshootChance = 0.3
	homeJitter      = 3 // pixels a return trip may miss the start point by
)

// motion generates cursor paths that look like a hand on a mouse: curved
// strokes, slow start and stop, slight tremor and the occasional overshoot.
type motion struct {
	rng *rand.Rand
}

// gesture moves from start to target, rests briefly and drifts back to a
// point near start. The returned steps do not include start itself.
func (m motion) gesture(start, target Position) []Step {
	steps := m.move(start, target)
	steps[len(steps)-1].Wait += m.between(150*time.Millisecond, 900*time.Millisecond)

	home := Position{
		X: start.X + m.rng.IntN(2*homeJitter+1) - homeJitter,
		Y: start.Y + m.rng.IntN(2*homeJitter+1) - homeJitter,
	}
	return append(steps, m.move(steps[len(steps)-1].Pos, home)...)
}

// move is one aimed movement. Longer ones sometimes overshoot and correct.
func (m motion) move(from, to Position) []Step {
	d := distance(from, to)
	if d < 40 || m.rng.Float64() >= overshootChance {
		return m.stroke(from, to)
	}

	f := 0.04 + m.rng.Float64()*0.06
	past := Position{
		X: to.X + int(math.Round(float64(to.X-from.X)*f)),
		Y: to.Y + int(math.Round(float64(to.Y-from.Y)*f)),
	}
	steps := m.stroke(from, past)
	steps[len(steps)-1].Wait += m.between(60*time.Millisecond, 180*time.Millisecond)
	return append(steps, m.stroke(past, to)...)
}

// stroke follows a cubic Bézier curve with a minimum-jerk speed profile.
// The duration follows Fitts's law, so long moves take longer but not
// proportionally longer.
func (m motion) stroke(from, to Position) []Step {
	d := distance(from, to)
	dur := time.Duration((120 + 140*math.Log2(d/10+1)) * (0.8 + 0.4*m.rng.Float64()) * float64(time.Millisecond))
	n := max(int(dur/frameTime), 3)

	// Bend the curve sideways by up to a quarter of its length.
	ax, ay := float64(from.X), float64(from.Y)
	dx, dy := float64(to.X)-ax, float64(to.Y)-ay
	nx, ny := 0.0, 0.0
	if d > 0 {
		nx, ny = -dy/d, dx/d
	}
	bend1 := (m.rng.Float64() - 0.5) * 0.5 * d
	bend2 := (m.rng.Float64() - 0.5) * 0.5 * d
	c1x, c1y := ax+dx*0.3+nx*bend1, ay+dy*0.3+ny*bend1
	c2x, c2y := ax+dx*0.7+nx*bend2, ay+dy*0.7+ny*bend2

	steps := make([]Step, 0, n)
	for i := 1; i <= n; i++ {
		t := minimumJerk(float64(i) / float64(n))
		x := bezier(ax, c1x, c2x, float64(to.X), t)
		y := bezier(ay, c1y, c2y, float64(to.Y), t)
		if i < n {
			// Tremor fades out as the hand settles on the target.
			tremor := 0.8 * (1 - t)
			x += m.rng.NormFloat64() * tremor
			y += m.rng.NormFloat64() * tremor
		}
		p := Position{X: int(math.Round(x)), Y: int(math.Round(y))}
		wait := time.Duration(float64(frameTime) * (0.75 + 0.5*m.rng.Float64()))

		// A real mouse sends nothing while it does not move.
		if len(steps) > 0 && steps[len(steps)-1].Pos == p {
			steps[len(steps)-1].Wait += wait
			continue
		}
		steps = append(steps, Step{Pos: p, Wait: wait})
	}
	if len(steps) == 0 {
		steps = append(steps, Step{Pos: to, Wait: frameTime})
	}
	return steps
}

// between returns a random duration in [lo, hi).
func (m motion) between(lo, hi time.Duration) time.Duration {
	return lo + time.Duration(m.rng.Int64N(int64(hi-lo)))
}

// minimumJerk maps linear time to the position of a smooth reach: slow
// start, fast middle, slow end.
func minimumJerk(t float64) float64 {
	return t * t * t * (10 - 15*t + 6*t*t)
}

func bezier(p0, p1, p2, p3, t float64) float64 {
	u := 1 - t
	return u*u*u*p0 + 3*u*u*t*p1 + 3*u*t*t*p2 + t*t*t*p3
}

func distance(a, b Position) float64 {
	return math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y))
}
