package util

import "testing"

func TestComputeExecutionScore(t *testing.T) {
	tests := []struct {
		name     string
		planned  float64
		achieved float64
		expected float64
	}{
		{"100% execution", 1000, 1000, 100},
		{"50% execution", 1000, 500, 50},
		{"0% execution", 1000, 0, 0},
		{"over 100% capped", 1000, 1500, 100},
		{"zero planned", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeExecutionScore(tt.planned, tt.achieved)
			if result != tt.expected {
				t.Errorf("got %.2f, want %.2f", result, tt.expected)
			}
		})
	}
}

func TestComputeGoalScore(t *testing.T) {
	tests := []struct {
		name     string
		start    float64
		target   float64
		current  float64
		expected float64
	}{
		{"fully achieved", 0, 50000, 50000, 100},
		{"half achieved", 0, 50000, 25000, 50},
		{"not started", 0, 50000, 0, 0},
		{"over achieved capped", 0, 50000, 60000, 100},
		{"reduction goal", 78, 73, 75, 60},
		{"same start/target", 50, 50, 50, 100},
		// Negative progress: going opposite direction of target
		{"negative growth", 0, 100, -20, -20},
		{"negative reduction", 24.4, 21.3, 26, -51.61},
		{"extreme negative clamped", 0, 100, -500, -100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ComputeGoalScore(tt.start, tt.target, tt.current)
			// Use tolerance for float comparison
			diff := result - tt.expected
			if diff < -0.01 || diff > 0.01 {
				t.Errorf("got %.2f, want %.2f", result, tt.expected)
			}
		})
	}
}

func TestComputePerformanceScore(t *testing.T) {
	// With default weights: 0.6 exec + 0.4 goal
	result := ComputePerformanceScore(80, 70)
	expected := (80 * 0.6) + (70 * 0.4) // = 48 + 28 = 76
	if result != expected {
		t.Errorf("got %.2f, want %.2f", result, expected)
	}
}

func TestGetTrafficLight(t *testing.T) {
	tests := []struct {
		score    float64
		expected string
	}{
		{95, "GREEN"},
		{90, "GREEN"},
		{89.9, "YELLOW"},
		{60, "YELLOW"},
		{59.9, "RED"},
		{0, "RED"},
	}

	for _, tt := range tests {
		result := GetTrafficLight(tt.score)
		if result != tt.expected {
			t.Errorf("score %.1f: got %s, want %s", tt.score, result, tt.expected)
		}
	}
}
