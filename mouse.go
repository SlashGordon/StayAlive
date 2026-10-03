package main

import "github.com/go-vgo/robotgo"

// Mouse is the part of the OS cursor API the jiggler needs.
type Mouse interface {
	Location() Position
	Move(Position)
	// Display returns the bounds of the display that contains p.
	Display(p Position) (Rect, bool)
}

type robotMouse struct{}

func (robotMouse) Location() Position {
	x, y := robotgo.Location()
	return Position{X: x, Y: y}
}

func (robotMouse) Move(p Position) {
	robotgo.Move(p.X, p.Y)
}

func (robotMouse) Display(p Position) (Rect, bool) {
	for i := range robotgo.DisplaysNum() {
		x, y, w, h := robotgo.GetDisplayBounds(i)
		if r := (Rect{X: x, Y: y, W: w, H: h}); r.Contains(p) {
			return r, true
		}
	}
	return Rect{}, false
}
