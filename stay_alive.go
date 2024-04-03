/*
MIT License - SlashGordon

Permission is granted to use, copy, modify, and/or distribute this software for any purpose with or without fee, subject to the inclusion of the above copyright notice and this permission notice in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY.
*/
package main

import (
	"math"
	"math/rand"
	"time"

	"github.com/go-vgo/robotgo"
)

type Position struct {
	X, Y int
}

var lastManualMove time.Time

func moveMouseRandomly() {
	for {

		if time.Since(lastManualMove) < time.Duration(4*time.Second) {
			// Wait for 1 minute of inactivity before moving the mouse again
			time.Sleep(5 * time.Second) // Check every 5 seconds to reduce CPU usage
			continue
		}

		screenWidth, screenHeight := robotgo.GetScreenSize()
		randX := rand.Intn(screenWidth)
		randY := rand.Intn(screenHeight)
		currX, currY := robotgo.Location()
		positions := calculateNewPositions(currX, currY, randX, randY)
		if !moveMouse(positions) {
			// If the mouse was moved manually, wait for 1 minute before moving it again
			time.Sleep(5 * time.Second) // Check every 5 seconds to reduce CPU usage
		}
	}
}

// calculateNewPositions calculates the positions for the mouse to move through.
func calculateNewPositions(currX, currY, targetX, targetY int) []Position {
	dist := math.Sqrt(float64((targetX-currX)*(targetX-currX) + (targetY-currY)*(targetY-currY)))
	steps := int(dist / 50)
	if steps < 10 {
		steps = 10
	}

	var positions []Position
	xInc := float64(targetX-currX) / float64(steps)
	yInc := float64(targetY-currY) / float64(steps)

	for i := 0; i <= steps; i++ {
		newX := currX + int(math.Round(float64(i)*xInc))
		newY := currY + int(math.Round(float64(i)*yInc))
		positions = append(positions, Position{X: newX, Y: newY})
	}

	return positions
}

// moveMouse moves the mouse to each position, checking for manual movement.
func moveMouse(positions []Position) bool {
	const threshold = 10 // Sensitivity threshold for manual movement detection

	for _, pos := range positions {
		robotgo.Move(pos.X, pos.Y)
		time.Sleep(time.Duration(rand.Intn(100)+50) * time.Millisecond)

		newX, newY := robotgo.Location()

		// Check if the mouse has moved significantly from the expected position
		if math.Abs(float64(newX-pos.X)) > threshold || math.Abs(float64(newY-pos.Y)) > threshold {
			lastManualMove = time.Now() // Update the time of the last manual movement
			return false
		}
	}

	return true
}

func main() {
	rand.Seed(time.Now().UnixNano())
	lastManualMove = time.Now() // Initialize with the current time
	// subtract 1 minute to start moving the mouse immediately
	lastManualMove = lastManualMove.Add(-2 * time.Minute)
	go moveMouseRandomly()

	for {
		// This loop keeps the main goroutine alive
		time.Sleep(1 * time.Minute)
	}
}
