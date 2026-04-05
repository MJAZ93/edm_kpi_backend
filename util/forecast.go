package util

import (
	"fmt"
	"math"
	"time"
)

type ForecastResult struct {
	TaskID              uint    `json:"task_id"`
	Title               string  `json:"title"`
	StartValue          float64 `json:"start_value"`
	TargetValue         float64 `json:"target_value"`
	CurrentValue        float64 `json:"current_value"`
	StartDate           string  `json:"start_date"`
	EndDate             string  `json:"end_date"`
	DaysElapsed         int     `json:"days_elapsed"`
	DaysRemaining       int     `json:"days_remaining"`
	VelocityPerDay      float64 `json:"velocity_per_day"`
	ProjectedFinalValue float64 `json:"projected_final_value"`
	WillReachTarget     bool    `json:"will_reach_target"`
	Alert               *string `json:"alert"`
	AlertMessage        *string `json:"alert_message,omitempty"`
}

func ForecastTask(taskID uint, title string, startValue, targetValue, currentValue float64, startDate, endDate time.Time) ForecastResult {
	now := time.Now()
	daysElapsed := int(math.Max(1, now.Sub(startDate).Hours()/24))
	daysRemaining := int(math.Max(0, endDate.Sub(now).Hours()/24))

	velocity := (currentValue - startValue) / float64(daysElapsed)
	projectedFinal := currentValue + (velocity * float64(daysRemaining))

	willReach := projectedFinal >= targetValue

	result := ForecastResult{
		TaskID:              taskID,
		Title:               title,
		StartValue:          startValue,
		TargetValue:         targetValue,
		CurrentValue:        currentValue,
		StartDate:           startDate.Format("2006-01-02"),
		EndDate:             endDate.Format("2006-01-02"),
		DaysElapsed:         daysElapsed,
		DaysRemaining:       daysRemaining,
		VelocityPerDay:      math.Round(velocity*100) / 100,
		ProjectedFinalValue: math.Round(projectedFinal*100) / 100,
		WillReachTarget:     willReach,
	}

	if projectedFinal < targetValue*0.9 {
		alert := "FORECAST_RISK"
		pct := math.Round((projectedFinal / targetValue) * 100)
		msg := fmt.Sprintf("Ao ritmo actual, a tarefa irá atingir apenas %.0f%% do objectivo.", pct)
		result.Alert = &alert
		result.AlertMessage = &msg
	}

	return result
}
