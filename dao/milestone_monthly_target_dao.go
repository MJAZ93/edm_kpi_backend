package dao

import (
	"time"

	"kpi-backend/model"
)

type MilestoneMonthlyTargetDao struct{}

// ListByMilestone returns all monthly target rows for a milestone, ordered by period.
func (d *MilestoneMonthlyTargetDao) ListByMilestone(milestoneID uint) ([]model.MilestoneMonthlyTarget, error) {
	var list []model.MilestoneMonthlyTarget
	err := Database.
		Where("milestone_id = ? AND deleted_at IS NULL", milestoneID).
		Order("period ASC").
		Find(&list).Error
	return list, err
}

// EnsureMonthsForMilestone creates empty monthly rows for each month in the
// [start, end] range if they don't already exist. Used when a milestone is
// first created so the user has one row per month to fill in.
func (d *MilestoneMonthlyTargetDao) EnsureMonthsForMilestone(milestoneID uint, start, end time.Time, createdBy uint) error {
	if start.IsZero() || end.IsZero() {
		return nil
	}
	cur := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	stop := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)

	// Collect existing periods
	existing := map[string]bool{}
	var rows []model.MilestoneMonthlyTarget
	Database.
		Select("period").
		Where("milestone_id = ? AND deleted_at IS NULL", milestoneID).
		Find(&rows)
	for _, r := range rows {
		existing[r.Period] = true
	}

	var toCreate []model.MilestoneMonthlyTarget
	for m := cur; !m.After(stop); m = m.AddDate(0, 1, 0) {
		key := m.Format("2006-01")
		if existing[key] {
			continue
		}
		toCreate = append(toCreate, model.MilestoneMonthlyTarget{
			MilestoneID:   milestoneID,
			Period:        key,
			PlannedValue:  0,
			AchievedValue: 0,
			CreatedBy:     createdBy,
		})
	}
	if len(toCreate) > 0 {
		return Database.Create(&toCreate).Error
	}
	return nil
}

// UpsertRow saves (create or update) a single monthly row for a milestone.
// If plannedValue or achievedValue is nil, that field is left untouched on update.
func (d *MilestoneMonthlyTargetDao) UpsertRow(milestoneID uint, period string, plannedValue, achievedValue *float64, notes *string, userID uint) (model.MilestoneMonthlyTarget, error) {
	var row model.MilestoneMonthlyTarget
	err := Database.
		Where("milestone_id = ? AND period = ? AND deleted_at IS NULL", milestoneID, period).
		First(&row).Error

	if err != nil {
		// create
		row = model.MilestoneMonthlyTarget{
			MilestoneID: milestoneID,
			Period:      period,
			CreatedBy:   userID,
		}
		if plannedValue != nil {
			row.PlannedValue = *plannedValue
		}
		if achievedValue != nil {
			row.AchievedValue = *achievedValue
		}
		if notes != nil {
			row.Notes = *notes
		}
		err = Database.Create(&row).Error
		return row, err
	}

	// update
	updates := map[string]interface{}{"updated_by": userID}
	if plannedValue != nil {
		updates["planned_value"] = *plannedValue
	}
	if achievedValue != nil {
		updates["achieved_value"] = *achievedValue
	}
	if notes != nil {
		updates["notes"] = *notes
	}
	err = Database.Model(&row).Updates(updates).Error
	return row, err
}

// SyncAchievedFromProgress syncs the achieved_value of a milestone's monthly
// target row for the given YYYY-MM period by summing all progress events whose
// period_reference resolves to that month.
//
// Conversion rules applied inside the SQL (same logic as periodRefToMonth in Go):
//   - YYYY-MM   → used directly
//   - YYYY-Www  → month of that ISO week's Monday
//   - YYYY-Qn   → first month of the quarter
//   - YYYY-Sn   → first month of the semester
//   - YYYY      → January of that year
//   - otherwise → created_at month (fallback)
//
// Creates the monthly row if it doesn't exist.
func (d *MilestoneMonthlyTargetDao) SyncAchievedFromProgress(milestoneID uint, period string, userID uint) error {
	if period == "" {
		return nil
	}
	var sum float64
	Database.Raw(`
		SELECT COALESCE(SUM(increment_value), 0)
		FROM milestone_progresses
		WHERE milestone_id = ?
		  AND (
		    CASE
		      WHEN period_reference ~ '^[0-9]{4}-[0-9]{2}$' THEN period_reference
		      WHEN period_reference ~ '^[0-9]{4}-W[0-9]{2}$' THEN
		        to_char(to_date(period_reference || '-1', 'IYYY-"W"IW-ID'), 'YYYY-MM')
		      WHEN period_reference ~ '^[0-9]{4}-Q[1-4]$' THEN
		        substring(period_reference, 1, 4) || '-' ||
		        lpad((((substring(period_reference, 7, 1)::int - 1) * 3 + 1)::text), 2, '0')
		      WHEN period_reference ~ '^[0-9]{4}-S[12]$' THEN
		        substring(period_reference, 1, 4) || '-' ||
		        CASE substring(period_reference, 7, 1) WHEN '1' THEN '01' ELSE '07' END
		      WHEN period_reference ~ '^[0-9]{4}$' THEN period_reference || '-01'
		      ELSE to_char(created_at AT TIME ZONE 'UTC', 'YYYY-MM')
		    END
		  ) = ?
		  AND deleted_at IS NULL
	`, milestoneID, period).Scan(&sum)

	_, err := d.UpsertRow(milestoneID, period, nil, &sum, nil, userID)
	return err
}

// MilestoneMonthlyRollup computes the milestone's aggregated planned and
// achieved values (across all its monthly rows) using the milestone's
// aggregation_type. Returns (plannedTotal, achievedTotal).
func (d *MilestoneMonthlyTargetDao) MilestoneMonthlyRollup(milestoneID uint) (float64, float64, error) {
	var ms model.Milestone
	if err := Database.Select("aggregation_type").Where("id = ?", milestoneID).First(&ms).Error; err != nil {
		return 0, 0, err
	}
	aggType := ms.AggregationType
	if aggType == "" {
		aggType = "SUM_UP"
	}

	rows, err := d.ListByMilestone(milestoneID)
	if err != nil {
		return 0, 0, err
	}

	var planned, achieved float64
	switch aggType {
	case "AVG":
		var countP, countA int
		for _, r := range rows {
			if r.PlannedValue != 0 {
				planned += r.PlannedValue
				countP++
			}
			if r.AchievedValue != 0 {
				achieved += r.AchievedValue
				countA++
			}
		}
		if countP > 0 {
			planned /= float64(countP)
		}
		if countA > 0 {
			achieved /= float64(countA)
		}
	case "LAST":
		for _, r := range rows {
			if r.PlannedValue != 0 {
				planned = r.PlannedValue
			}
			if r.AchievedValue != 0 {
				achieved = r.AchievedValue
			}
		}
	default: // SUM_UP and SUM_DOWN: summation at milestone level
		for _, r := range rows {
			planned += r.PlannedValue
			achieved += r.AchievedValue
		}
	}

	return planned, achieved, nil
}

// MonthlyRow is a per-period row used for chart aggregation.
type MonthlyRow struct {
	Period        string  `json:"period"`
	PlannedValue  float64 `json:"planned_value"`
	AchievedValue float64 `json:"achieved_value"`
	ExecPct       float64 `json:"exec_pct"`
}

// MonthlyForTask aggregates all milestones' monthly rows for a task, per
// period, using the task's aggregation_type (SUM_UP/AVG/LAST; SUM_DOWN uses
// start_value - sum(achieved)).
// The returned slice spans all months between the task's start_date and
// end_date (filling missing months with 0/0).
func (d *MilestoneMonthlyTargetDao) MonthlyForTask(taskID uint) ([]MonthlyRow, error) {
	var task model.Task
	if err := Database.Select("aggregation_type", "start_value", "start_date", "end_date").
		Where("id = ?", taskID).First(&task).Error; err != nil {
		return nil, err
	}
	aggType := task.AggregationType
	if aggType == "" {
		aggType = "SUM_UP"
	}

	// Fetch all monthly rows for all non-deleted milestones of this task
	type raw struct {
		Period        string
		PlannedValue  float64
		AchievedValue float64
	}
	var rows []raw
	Database.Raw(`
		SELECT t.period, t.planned_value, t.achieved_value
		FROM milestone_monthly_targets t
		JOIN milestones m ON m.id = t.milestone_id AND m.deleted_at IS NULL
		WHERE m.task_id = ? AND t.deleted_at IS NULL
		ORDER BY t.period ASC
	`, taskID).Scan(&rows)

	// For AVG and MANUAL: we need the TOTAL number of milestones for this task
	// so we can divide by a fixed denominator (not just the subset that reported
	// each period). This keeps chart values in the same scale as the task's
	// start_value / target_value.
	// Example: 8 milestones (one per ASC), only 1 has Jan data with value=173.2 →
	//   wrong (old): 173.2 / 1 = 173.2  (counts only reporters)
	//   correct:     173.2 / 8 = 21.65  (true average; non-reporters contribute 0)
	var totalMilestones int64
	if aggType == "AVG" || aggType == "MANUAL" {
		Database.Model(&model.Milestone{}).
			Where("task_id = ? AND deleted_at IS NULL", taskID).
			Count(&totalMilestones)
		if totalMilestones == 0 {
			totalMilestones = 1
		}
	}

	// For LAST: collect the single most-recent row per milestone so we can
	// pick up only the latest period entry, not sum all periods.
	type lastRaw struct {
		MilestoneID   uint
		Period        string
		PlannedValue  float64
		AchievedValue float64
	}
	var lastRows []lastRaw
	if aggType == "LAST" {
		Database.Raw(`
			SELECT DISTINCT ON (t.milestone_id)
				t.milestone_id, t.period, t.planned_value, t.achieved_value
			FROM milestone_monthly_targets t
			JOIN milestones m ON m.id = t.milestone_id AND m.deleted_at IS NULL
			WHERE m.task_id = ? AND t.deleted_at IS NULL
			ORDER BY t.milestone_id, t.period DESC
		`, taskID).Scan(&lastRows)
	}

	// Group per period
	byPeriod := map[string]*MonthlyRow{}
	if aggType == "LAST" {
		// For LAST type: one bar per milestone at its own latest period.
		// Aggregate the most-recent values across milestones per period.
		lastCounts := map[string]int{}
		for _, r := range lastRows {
			pd, ok := byPeriod[r.Period]
			if !ok {
				pd = &MonthlyRow{Period: r.Period}
				byPeriod[r.Period] = pd
			}
			pd.PlannedValue += r.PlannedValue
			pd.AchievedValue += r.AchievedValue
			lastCounts[r.Period]++
		}
		// Average in case multiple milestones share the same latest period.
		for k, pd := range byPeriod {
			if lastCounts[k] > 1 {
				pd.PlannedValue /= float64(lastCounts[k])
				pd.AchievedValue /= float64(lastCounts[k])
			}
		}
	} else {
		for _, r := range rows {
			pd, ok := byPeriod[r.Period]
			if !ok {
				pd = &MonthlyRow{Period: r.Period}
				byPeriod[r.Period] = pd
			}
			pd.PlannedValue += r.PlannedValue
			pd.AchievedValue += r.AchievedValue
		}
	}

	// Apply task-level aggregation
	if aggType == "AVG" || aggType == "MANUAL" {
		// Divide by TOTAL milestone count so values stay in the same unit/scale as
		// the task's start_value / target_value, regardless of how many milestones
		// have entered data for each period.
		// MANUAL tasks manage their overall current_value by hand but milestones
		// still track per-entity rates — averaging gives the meaningful composite.
		for _, pd := range byPeriod {
			pd.PlannedValue /= float64(totalMilestones)
			pd.AchievedValue /= float64(totalMilestones)
		}
	} else if aggType == "SUM_DOWN" {
		startVal := 0.0
		if task.StartValue != nil {
			startVal = *task.StartValue
		}
		for _, pd := range byPeriod {
			pd.PlannedValue = startVal - pd.PlannedValue
			pd.AchievedValue = startVal - pd.AchievedValue
		}
	}
	// SUM_UP: leave raw sum as-is (correct for cumulative totals).
	// LAST: already handled above.

	// Build the full list over the task's period range
	start := task.StartDate
	end := task.EndDate
	// fallback to the observed periods if no dates
	if start == nil || end == nil {
		keys := make([]string, 0, len(byPeriod))
		for k := range byPeriod {
			keys = append(keys, k)
		}
		if len(keys) == 0 {
			return []MonthlyRow{}, nil
		}
		// Sort via period parse
		sortPeriods(keys)
		result := make([]MonthlyRow, 0, len(keys))
		for _, k := range keys {
			pd := byPeriod[k]
			pd.ExecPct = computeExecPct(pd.PlannedValue, pd.AchievedValue)
			result = append(result, *pd)
		}
		return result, nil
	}

	s := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	e := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	result := make([]MonthlyRow, 0)
	for m := s; !m.After(e); m = m.AddDate(0, 1, 0) {
		key := m.Format("2006-01")
		pd, ok := byPeriod[key]
		if !ok {
			pd = &MonthlyRow{Period: key}
		}
		pd.ExecPct = computeExecPct(pd.PlannedValue, pd.AchievedValue)
		result = append(result, *pd)
	}
	return result, nil
}

// MonthlyForProject aggregates all task monthly rows into a project-level
// monthly series. Uses per-task weight (if set) as weighting for the planned
// and achieved values; execution % per month = (sum(achieved*w) / sum(planned*w)) * 100.
// Falls back to simple average when all weights are zero.
func (d *MilestoneMonthlyTargetDao) MonthlyForProject(projectID uint) ([]MonthlyRow, error) {
	var project model.Project
	if err := Database.Select("start_date", "end_date").
		Where("id = ?", projectID).First(&project).Error; err != nil {
		return nil, err
	}

	// Get all tasks
	var tasks []model.Task
	Database.Select("id", "weight", "aggregation_type", "start_value", "start_date", "end_date").
		Where("project_id = ? AND deleted_at IS NULL", projectID).
		Find(&tasks)

	// For each task, compute its monthly rollup and weight it
	byPeriod := map[string]*MonthlyRow{}
	weightSum := map[string]float64{}

	for _, t := range tasks {
		w := t.Weight
		if w <= 0 {
			w = 1
		}
		rows, err := d.MonthlyForTask(t.ID)
		if err != nil {
			continue
		}
		for _, r := range rows {
			pd, ok := byPeriod[r.Period]
			if !ok {
				pd = &MonthlyRow{Period: r.Period}
				byPeriod[r.Period] = pd
			}
			// We aggregate WEIGHTED-SUM of planned/achieved. Later we'll scale back.
			pd.PlannedValue += r.PlannedValue * w
			pd.AchievedValue += r.AchievedValue * w
			weightSum[r.Period] += w
		}
	}

	// Normalize by weight sum per period (gives a weighted AVERAGE)
	for k, pd := range byPeriod {
		ws := weightSum[k]
		if ws > 0 {
			pd.PlannedValue /= ws
			pd.AchievedValue /= ws
		}
		pd.ExecPct = computeExecPct(pd.PlannedValue, pd.AchievedValue)
	}

	// Build full range
	start := project.StartDate
	end := project.EndDate
	// fallback: use observed periods
	if start == nil || end == nil {
		keys := make([]string, 0, len(byPeriod))
		for k := range byPeriod {
			keys = append(keys, k)
		}
		sortPeriods(keys)
		result := make([]MonthlyRow, 0, len(keys))
		for _, k := range keys {
			result = append(result, *byPeriod[k])
		}
		return result, nil
	}

	s := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	e := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	result := make([]MonthlyRow, 0)
	for m := s; !m.After(e); m = m.AddDate(0, 1, 0) {
		key := m.Format("2006-01")
		pd, ok := byPeriod[key]
		if !ok {
			pd = &MonthlyRow{Period: key}
		}
		result = append(result, *pd)
	}
	return result, nil
}

// DeleteByMilestone removes all monthly rows for a milestone (hard delete).
// Used when a milestone is deleted.
func (d *MilestoneMonthlyTargetDao) DeleteByMilestone(milestoneID uint) error {
	return Database.Where("milestone_id = ?", milestoneID).Delete(&model.MilestoneMonthlyTarget{}).Error
}

// EnsureMonthsForTask ensures every non-deleted milestone of a task has
// monthly rows covering the [start, end] range. Called when a task's date
// range changes, so existing milestones pick up the extended months.
func (d *MilestoneMonthlyTargetDao) EnsureMonthsForTask(taskID uint, start, end time.Time, userID uint) error {
	var msIDs []uint
	Database.Model(&model.Milestone{}).
		Where("task_id = ? AND deleted_at IS NULL", taskID).
		Pluck("id", &msIDs)
	for _, id := range msIDs {
		_ = d.EnsureMonthsForMilestone(id, start, end, userID)
	}
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

func computeExecPct(planned, achieved float64) float64 {
	if planned == 0 {
		return 0
	}
	pct := (achieved / planned) * 100
	if pct < 0 {
		return 0
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// sortPeriods sorts a slice of "YYYY-MM" strings in-place (ascending).
// Uses a simple insertion sort since the list is tiny.
func sortPeriods(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j-1] > s[j]; j-- {
			s[j-1], s[j] = s[j], s[j-1]
		}
	}
}
