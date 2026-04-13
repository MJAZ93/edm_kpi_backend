package util

import (
	"math"
	"os"
	"strconv"
)

func ComputeExecutionScore(totalPlanned, totalAchieved float64) float64 {
	if totalPlanned <= 0 {
		return 0
	}
	score := (totalAchieved / totalPlanned) * 100
	return math.Min(score, 100)
}

// ComputeExecutionScoreReduction inverts the formula for reduction goals:
// lower achieved vs planned = better score (e.g. losses, defects).
func ComputeExecutionScoreReduction(totalPlanned, totalAchieved float64) float64 {
	if totalPlanned <= 0 || totalAchieved <= 0 {
		return 0
	}
	score := (totalPlanned / totalAchieved) * 100
	return math.Min(score, 100)
}

func ComputeGoalScore(startValue, targetValue, currentValue float64) float64 {
	diff := targetValue - startValue
	if diff == 0 {
		return 100
	}
	score := ((currentValue - startValue) / diff) * 100
	// Clamp between -100 and 100
	return math.Max(-100, math.Min(score, 100))
}

func ComputePerformanceScore(executionScore, goalScore float64) float64 {
	execWeight := getFloatEnv("SCORE_EXECUTION_WEIGHT", 0.6)
	goalWeight := getFloatEnv("SCORE_GOAL_WEIGHT", 0.4)
	return (executionScore * execWeight) + (goalScore * goalWeight)
}

func GetTrafficLight(score float64) string {
	if score >= 90 {
		return "GREEN"
	}
	if score >= 60 {
		return "YELLOW"
	}
	return "RED"
}

func getFloatEnv(key string, defaultVal float64) float64 {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return defaultVal
	}
	return f
}
