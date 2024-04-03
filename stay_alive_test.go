package main

import (
	"reflect"
	"testing"
)

func TestCalculateNewPositions(t *testing.T) {
	tests := []struct {
		name    string
		currX   int
		currY   int
		targetX int
		targetY int
		want    []Position
	}{
		{
			name:    "simple straight line",
			currX:   0,
			currY:   0,
			targetX: 100,
			targetY: 0,
			want:    []Position{{0, 0}, {10, 0}, {20, 0}, {30, 0}, {40, 0}, {50, 0}, {60, 0}, {70, 0}, {80, 0}, {90, 0}, {100, 0}},
		},
		// Add more test cases as necessary
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateNewPositions(tt.currX, tt.currY, tt.targetX, tt.targetY)

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("calculateNewPositions() got = %v, want %v", got, tt.want)
			}
		})
	}
}
