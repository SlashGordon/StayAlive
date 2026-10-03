//go:build !darwin

package main

import "time"

// lastSystemInput is only implemented on macOS. Elsewhere the jiggler detects
// activity by watching the cursor position, so keyboard input is not seen.
func lastSystemInput(time.Time) (time.Time, bool) {
	return time.Time{}, false
}
