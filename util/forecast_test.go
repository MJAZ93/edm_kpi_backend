package util

import (
	"testing"
	"time"
)

func TestForecastTask_WillReach(t *testing.T) {
	start := time.Now().AddDate(0, -3, 0)
	end := time.Now().AddDate(0, 9, 0)

	result := ForecastTask(1, "Test Task", 0, 50000, 18000, start, end)

	if !result.WillReachTarget {
		t.Errorf("expected will_reach_target=true, projected=%.0f", result.ProjectedFinalValue)
	}
	if result.Alert != nil {
		t.Errorf("expected no alert, got %v", *result.Alert)
	}
}

func TestForecastTask_WillNotReach(t *testing.T) {
	start := time.Now().AddDate(0, -10, 0)
	end := time.Now().AddDate(0, 1, 0)

	result := ForecastTask(1, "Slow Task", 0, 50000, 5000, start, end)

	if result.WillReachTarget {
		t.Errorf("expected will_reach_target=false, projected=%.0f", result.ProjectedFinalValue)
	}
	if result.Alert == nil {
		t.Errorf("expected FORECAST_RISK alert")
	} else if *result.Alert != "FORECAST_RISK" {
		t.Errorf("expected FORECAST_RISK, got %s", *result.Alert)
	}
}

func TestForecastTask_VelocityCalculation(t *testing.T) {
	start := time.Now().AddDate(0, 0, -100) // 100 days ago
	end := time.Now().AddDate(0, 0, 100)    // 100 days from now

	result := ForecastTask(1, "Velocity Test", 0, 10000, 5000, start, end)

	// velocity ~= 50/day, projected ~= 5000 + 50*100 = 10000
	if result.VelocityPerDay < 45 || result.VelocityPerDay > 55 {
		t.Errorf("expected velocity ~50/day, got %.2f", result.VelocityPerDay)
	}
}
