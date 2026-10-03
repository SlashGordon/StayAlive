package main

/*
#cgo LDFLAGS: -framework ApplicationServices
#include <ApplicationServices/ApplicationServices.h>

static double secondsSinceLastInput(void) {
	return CGEventSourceSecondsSinceLastEventType(
		kCGEventSourceStateHIDSystemState, kCGAnyInputEventType);
}
*/
import "C"

import "time"

// lastSystemInput returns the time of the last keyboard, mouse or trackpad
// event macOS has seen. Synthetic events posted by this program count as well,
// so the caller has to filter its own moves.
func lastSystemInput(now time.Time) (time.Time, bool) {
	idle := time.Duration(float64(C.secondsSinceLastInput()) * float64(time.Second))
	return now.Add(-idle), true
}
