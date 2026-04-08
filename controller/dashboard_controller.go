package controller

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type DashboardController struct{}

func (DashboardController) Summary(c *gin.Context) {
	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	dashDao := dao.DashboardDao{}
	summary := dashDao.GetSummaryScoped(&scope)
	c.JSON(http.StatusOK, summary)
}

func (DashboardController) Performance(c *gin.Context) {
	entityType := c.Query("entity_type")
	entityIDStr := c.DefaultQuery("entity_id", "0")
	periodStr := c.DefaultQuery("period", time.Now().Format("2006-01"))

	entityID, _ := strconv.Atoi(entityIDStr)
	period, _ := time.Parse("2006-01", periodStr)

	perfDao := dao.PerformanceDao{}
	cache, err := perfDao.GetScore(entityType, uint(entityID), period)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "no performance data for this entity/period"})
		return
	}

	c.JSON(http.StatusOK, cache)
}

func (DashboardController) DrillDown(c *gin.Context) {
	level := c.Query("level")
	idStr := c.DefaultQuery("id", "0")
	periodStr := c.DefaultQuery("period", time.Now().Format("2006-01"))

	id, _ := strconv.Atoi(idStr)
	period, _ := time.Parse("2006-01", periodStr)

	perfDao := dao.PerformanceDao{}

	type DrillItem struct {
		ID             uint    `json:"id"`
		Name           string  `json:"name"`
		Type           string  `json:"type"`
		ExecutionScore float64 `json:"execution_score"`
		GoalScore      float64 `json:"goal_score"`
		TotalScore     float64 `json:"total_score"`
		TrafficLight   string  `json:"traffic_light"`
	}

	var items []DrillItem

	switch level {
	case "ALL_DIRECOES":
		// CA view: all direcções with their cached performance scores
		direcaoDao := dao.DirecaoDao{}
		direcoes, _, _ := direcaoDao.List(0, 200)
		var ids []uint
		for _, d := range direcoes {
			ids = append(ids, d.ID)
		}
		scores, _ := perfDao.GetScores("DIRECAO", ids, period)
		scoreMap := make(map[uint]dao.PerformanceCacheAlias)
		for _, s := range scores {
			scoreMap[s.EntityID] = s
		}
		for _, d := range direcoes {
			item := DrillItem{ID: d.ID, Name: d.Name, Type: "DIRECAO"}
			if s, ok := scoreMap[d.ID]; ok {
				item.ExecutionScore = s.ExecutionScore
				item.GoalScore = s.GoalScore
				item.TotalScore = s.TotalScore
				item.TrafficLight = s.TrafficLight
			} else {
				item.TrafficLight = "RED"
			}
			items = append(items, item)
		}

	case "NATIONAL":
		// Show regions — score = avg of child ASCs
		geoDao := dao.GeoDao{}
		regioes, _ := geoDao.GetAllRegioes()
		for _, r := range regioes {
			item := DrillItem{ID: r.ID, Name: r.Name, Type: "REGIAO"}
			item.ExecutionScore, item.GoalScore, item.TotalScore, item.TrafficLight = perfDao.ComputeScoreForRegiao(r.ID)
			items = append(items, item)
		}

	case "REGIONAL":
		// Show ASCs — score computed from tasks with ASC scope
		geoDao := dao.GeoDao{}
		ascs, _ := geoDao.ListASCsByRegiao(uint(id))
		for _, a := range ascs {
			item := DrillItem{ID: a.ID, Name: a.Name, Type: "ASC"}
			item.ExecutionScore, item.GoalScore, item.TotalScore, item.TrafficLight = perfDao.ComputeScoreForScope("ASC", a.ID)
			items = append(items, item)
		}

	case "PELOURO":
		direcaoDao := dao.DirecaoDao{}
		direcoes, _ := direcaoDao.ListByPelouro(uint(id))
		var ids []uint
		for _, d := range direcoes {
			ids = append(ids, d.ID)
		}
		scores, _ := perfDao.GetScores("DIRECAO", ids, period)
		scoreMap := make(map[uint]dao.PerformanceCacheAlias)
		for _, s := range scores {
			scoreMap[s.EntityID] = dao.PerformanceCacheAlias(s)
		}
		for _, d := range direcoes {
			item := DrillItem{ID: d.ID, Name: d.Name, Type: "DIRECAO"}
			if s, ok := scoreMap[d.ID]; ok {
				item.ExecutionScore = s.ExecutionScore
				item.GoalScore = s.GoalScore
				item.TotalScore = s.TotalScore
				item.TrafficLight = s.TrafficLight
			}
			items = append(items, item)
		}

	case "DIRECAO":
		deptDao := dao.DepartamentoDao{}
		depts, _ := deptDao.ListByDirecao(uint(id))
		var ids []uint
		for _, d := range depts {
			ids = append(ids, d.ID)
		}
		scores, _ := perfDao.GetScores("DEPARTAMENTO", ids, period)
		scoreMap := make(map[uint]dao.PerformanceCacheAlias)
		for _, s := range scores {
			scoreMap[s.EntityID] = dao.PerformanceCacheAlias(s)
		}
		for _, d := range depts {
			item := DrillItem{ID: d.ID, Name: d.Name, Type: "DEPARTAMENTO"}
			if s, ok := scoreMap[d.ID]; ok {
				item.ExecutionScore = s.ExecutionScore
				item.GoalScore = s.GoalScore
				item.TotalScore = s.TotalScore
				item.TrafficLight = s.TrafficLight
			}
			items = append(items, item)
		}
	}

	c.JSON(http.StatusOK, gin.H{"level": level, "items": items})
}

func (DashboardController) MapData(c *gin.Context) {
	level := c.DefaultQuery("level", "REGIONAL")

	geoDao := dao.GeoDao{}
	perfDao := dao.PerformanceDao{}

	// Resolve the caller's scope — used both to auto-filter which ASCs to show
	// (when no explicit filter param is given) and to scope the score computation.
	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))

	type Feature struct {
		Type       string      `json:"type"`
		Geometry   interface{} `json:"geometry,omitempty"`
		Properties interface{} `json:"properties"`
	}
	var features []Feature

	if level == "REGIONAL" {
		regioes, _ := geoDao.GetAllRegioes()
		for _, r := range regioes {
			var geom interface{}
			if r.Polygon != "" {
				_ = json.Unmarshal([]byte(r.Polygon), &geom)
			}
			_, _, score, light := perfDao.ComputeScoreForRegiao(r.ID)
			features = append(features, Feature{
				Type:     "Feature",
				Geometry: geom,
				Properties: gin.H{
					"id":            r.ID,
					"name":          r.Name,
					"total_score":   score,
					"traffic_light": light,
				},
			})
		}
	} else {
		// ── Determine which ASCs to display ──────────────────────────────────────
		//
		// Priority of filters:
		//   1. ?asc_ids=1,2,3   — explicit list (member/regional dashboards)
		//   2. ?regiao_id=X     — region filter (regional analytics)
		//   3. ?direcao_id=X    — direction filter (analytics/admin override)
		//   4. auth scope       — auto-resolve from JWT (preferred for dashboards)
		//   5. global fallback  — show all (ADMIN / CA)

		allAscs, _ := geoDao.GetAllASCs()
		var ascsToShow []model.ASC

		switch {
		case c.Query("asc_ids") != "":
			set := map[uint]bool{}
			for _, part := range strings.Split(c.Query("asc_ids"), ",") {
				if id, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && id > 0 {
					set[uint(id)] = true
				}
			}
			for _, a := range allAscs {
				if set[a.ID] {
					ascsToShow = append(ascsToShow, a)
				}
			}

		case c.Query("regiao_id") != "":
			regiaoID, _ := strconv.Atoi(c.Query("regiao_id"))
			regional, _ := geoDao.ListASCsByRegiao(uint(regiaoID))
			set := map[uint]bool{}
			for _, a := range regional {
				set[a.ID] = true
			}
			for _, a := range allAscs {
				if set[a.ID] {
					ascsToShow = append(ascsToShow, a)
				}
			}

		case c.Query("direcao_id") != "":
			// Explicit direction override — kept for analytics/admin pages
			direcaoID, _ := strconv.Atoi(c.Query("direcao_id"))
			taskDao := dao.TaskDao{}
			dirTasks, _ := taskDao.ListByOwner("DIRECAO", uint(direcaoID))
			ascIDSet := map[uint]bool{}
			for _, t := range dirTasks {
				for _, s := range t.Scopes {
					if s.ScopeType == "ASC" && s.ScopeID != nil {
						ascIDSet[*s.ScopeID] = true
					}
				}
			}
			for _, a := range allAscs {
				if ascIDSet[a.ID] {
					ascsToShow = append(ascsToShow, a)
				}
			}

		case !scope.IsGlobal:
			// Auto-resolve from the caller's scope — used by role-specific dashboards.
			if scope.RegiaoID != 0 {
				// Regional director: ASCs in their region
				ascsToShow, _ = geoDao.ListASCsByRegiao(scope.RegiaoID)
			} else {
				// Collect ASC IDs referenced by tasks owned within this scope
				taskDao := dao.TaskDao{}
				ascIDSet := map[uint]bool{}
				collectASCs := func(ownerType string, ownerIDs []uint) {
					for _, id := range ownerIDs {
						tasks, _ := taskDao.ListByOwner(ownerType, id)
						for _, t := range tasks {
							for _, s := range t.Scopes {
								if s.ScopeType == "ASC" && s.ScopeID != nil {
									ascIDSet[*s.ScopeID] = true
								}
							}
						}
					}
				}
				collectASCs("DIRECAO", scope.DirecaoIDs)
				collectASCs("DEPARTAMENTO", scope.DepartamentoIDs)
				for _, a := range allAscs {
					if ascIDSet[a.ID] {
						ascsToShow = append(ascsToShow, a)
					}
				}
			}

		default:
			// ADMIN / CA: show all
			ascsToShow = allAscs
		}

		// ── Compute score per ASC, scoped to the caller's tasks ──────────────────
		//
		// - ADMIN / CA:            global score (all tasks touching this ASC)
		// - DIRECAO / DEPT / etc.: score relative to their own tasks only
		// - Regional director:     global ASC score (they're read-only viewers)
		for _, a := range ascsToShow {
			var geom interface{}
			if a.Polygon != "" {
				_ = json.Unmarshal([]byte(a.Polygon), &geom)
			}
			execScore, goalScore, totalScore, light := perfDao.ComputeScoreForScopeScoped("ASC", a.ID, &scope)
			features = append(features, Feature{
				Type:     "Feature",
				Geometry: geom,
				Properties: gin.H{
					"id":              a.ID,
					"name":            a.Name,
					"execution_score": math.Round(execScore*10) / 10,
					"goal_score":      math.Round(goalScore*10) / 10,
					"total_score":     math.Round(totalScore*10) / 10,
					"traffic_light":   light,
				},
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"type":     "FeatureCollection",
		"features": features,
	})
}

// ScopeStats returns rich statistics for a single ASC or REGIAO:
//   - Performance scores
//   - Task list (with milestone progress)
//   - Project list (distinct projects that have tasks in this scope)
//   - Directions breakdown
//
// Query params: type=ASC|REGIAO  id=<uint>
func (DashboardController) ScopeStats(c *gin.Context) {
	scopeType := c.Query("type") // "ASC" or "REGIAO"
	idStr := c.Query("id")
	id, _ := strconv.Atoi(idStr)
	if id == 0 || (scopeType != "ASC" && scopeType != "REGIAO") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type and id required"})
		return
	}

	geoDao := dao.GeoDao{}
	taskDao := dao.TaskDao{}
	perfDao := dao.PerformanceDao{}

	// --- name ---
	name := ""
	if scopeType == "ASC" {
		a, err := geoDao.GetASCByID(uint(id))
		if err == nil {
			name = a.Name
		}
	} else {
		r, err := geoDao.GetRegiaoByID(uint(id))
		if err == nil {
			name = r.Name
		}
	}

	// --- score ---
	var execScore, goalScore, totalScore float64
	var trafficLight string
	if scopeType == "ASC" {
		execScore, goalScore, totalScore, trafficLight = perfDao.ComputeScoreForScope("ASC", uint(id))
	} else {
		execScore, goalScore, totalScore, trafficLight = perfDao.ComputeScoreForRegiao(uint(id))
	}

	// --- tasks scoped to this entity (or child ASCs if REGIAO) ---
	var scopedTasks []dao.TaskSummary
	if scopeType == "ASC" {
		scopedTasks, _ = taskDao.SummaryByScopeEntity("ASC", uint(id))
	} else {
		ascs, _ := geoDao.ListASCsByRegiao(uint(id))
		seen := map[uint]bool{}
		for _, a := range ascs {
			ts, _ := taskDao.SummaryByScopeEntity("ASC", a.ID)
			for _, t := range ts {
				if !seen[t.ID] {
					seen[t.ID] = true
					scopedTasks = append(scopedTasks, t)
				}
			}
		}
		// also tasks scoped directly to the region
		ts2, _ := taskDao.SummaryByScopeEntity("REGIAO", uint(id))
		for _, t := range ts2 {
			if !seen[t.ID] {
				seen[t.ID] = true
				scopedTasks = append(scopedTasks, t)
			}
		}
	}

	// --- projects & directions from tasks ---
	type ProjectInfo struct {
		ID    uint   `json:"id"`
		Title string `json:"title"`
	}
	type DirInfo struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	projectMap := map[uint]ProjectInfo{}
	dirMap := map[uint]DirInfo{}
	direcaoDao := dao.DirecaoDao{}

	for _, t := range scopedTasks {
		if t.ProjectID != 0 {
			projectMap[t.ProjectID] = ProjectInfo{ID: t.ProjectID, Title: t.ProjectTitle}
		}
		if t.OwnerType == "DIRECAO" {
			if _, ok := dirMap[t.OwnerID]; !ok {
				d, err := direcaoDao.GetByID(t.OwnerID)
				if err == nil {
					dirMap[t.OwnerID] = DirInfo{ID: d.ID, Name: d.Name}
				}
			}
		} else if t.OwnerType == "DEPARTAMENTO" {
			deptDao := dao.DepartamentoDao{}
			dept, err := deptDao.GetByID(t.OwnerID)
			if err == nil && dept.DirecaoID != 0 {
				if _, ok := dirMap[dept.DirecaoID]; !ok {
					d, err2 := direcaoDao.GetByID(dept.DirecaoID)
					if err2 == nil {
						dirMap[dept.DirecaoID] = DirInfo{ID: d.ID, Name: d.Name}
					}
				}
			}
		}
	}

	projects := make([]ProjectInfo, 0, len(projectMap))
	for _, p := range projectMap {
		if p.Title != "" { // skip deleted projects (LEFT JOIN returns empty title)
			projects = append(projects, p)
		}
	}
	dirs := make([]DirInfo, 0, len(dirMap))
	for _, d := range dirMap {
		dirs = append(dirs, d)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":              id,
		"name":            name,
		"type":            scopeType,
		"execution_score": math.Round(execScore*10) / 10,
		"goal_score":      math.Round(goalScore*10) / 10,
		"total_score":     math.Round(totalScore*10) / 10,
		"traffic_light":   trafficLight,
		"task_count":      len(scopedTasks),
		"tasks":           scopedTasks,
		"project_count":   len(projects),
		"projects":        projects,
		"direction_count": len(dirs),
		"directions":      dirs,
	})
}

func (DashboardController) Forecast(c *gin.Context) {
	taskID, _ := strconv.Atoi(c.Query("task_id"))
	taskDao := dao.TaskDao{}
	task, err := taskDao.GetByID(uint(taskID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if task.StartDate == nil || task.EndDate == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "task must have start_date and end_date"})
		return
	}

	startVal := float64(0)
	if task.StartValue != nil {
		startVal = *task.StartValue
	}

	result := util.ForecastTask(task.ID, task.Title, startVal, task.TargetValue, task.CurrentValue, *task.StartDate, *task.EndDate)
	c.JSON(http.StatusOK, result)
}

func (DashboardController) TopPerformers(c *gin.Context) {
	entityType := c.DefaultQuery("entity_type", "ASC")
	periodStr := c.DefaultQuery("period", time.Now().Format("2006-01"))
	limitStr := c.DefaultQuery("limit", "10")
	limit, _ := strconv.Atoi(limitStr)

	perfDao := dao.PerformanceDao{}

	type RankedItem struct {
		Rank         int     `json:"rank"`
		ID           uint    `json:"id"`
		Name         string  `json:"name"`
		ExecScore    float64 `json:"execution_score"`
		GoalScore    float64 `json:"goal_score"`
		TotalScore   float64 `json:"total_score"`
		TrafficLight string  `json:"traffic_light"`
	}

	// Use live computation for all entity types
	items, err := perfDao.LiveTopPerformers(entityType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	ranked := make([]RankedItem, 0, len(items))
	for i, item := range items {
		ranked = append(ranked, RankedItem{
			Rank:         i + 1,
			ID:           item.EntityID,
			Name:         item.EntityName,
			ExecScore:    math.Round(item.ExecScore*10) / 10,
			GoalScore:    math.Round(item.GoalScore*10) / 10,
			TotalScore:   math.Round(item.TotalScore*10) / 10,
			TrafficLight: item.TrafficLight,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"period":      periodStr,
		"entity_type": entityType,
		"ranking":     ranked,
	})
}

func (DashboardController) Timeline(c *gin.Context) {
	entityType := c.Query("entity_type")
	entityIDStr := c.DefaultQuery("entity_id", "0")
	fromStr := c.DefaultQuery("from", time.Now().AddDate(0, -6, 0).Format("2006-01"))
	toStr := c.DefaultQuery("to", time.Now().Format("2006-01"))

	entityID, _ := strconv.Atoi(entityIDStr)
	from, _ := time.Parse("2006-01", fromStr)
	to, _ := time.Parse("2006-01", toStr)

	perfDao := dao.PerformanceDao{}
	list, err := perfDao.GetTimeline(entityType, uint(entityID), from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	type PeriodData struct {
		Period       string  `json:"period"`
		TotalScore   float64 `json:"total_score"`
		TrafficLight string  `json:"traffic_light"`
	}

	var periods []PeriodData
	for _, item := range list {
		periods = append(periods, PeriodData{
			Period:       item.Period.Format("2006-01"),
			TotalScore:   item.TotalScore,
			TrafficLight: item.TrafficLight,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"entity_type": entityType,
		"entity_id":   entityID,
		"periods":     periods,
	})
}

func (DashboardController) Distribution(c *gin.Context) {
	dimension := c.DefaultQuery("dimension", "traffic_light")
	entityType := c.DefaultQuery("entity_type", "ASC")

	perfDao := dao.PerformanceDao{}
	items, err := perfDao.LiveTopPerformers(entityType, 0)
	if err != nil || len(items) == 0 {
		dashDao := dao.DashboardDao{}
		data := dashDao.GetDistribution(dimension)
		c.JSON(http.StatusOK, gin.H{"dimension": dimension, "data": data})
		return
	}

	type Bucket struct {
		Label      string  `json:"label"`
		Count      int     `json:"count"`
		Percentage float64 `json:"percentage"`
	}

	counts := map[string]int{"GREEN": 0, "YELLOW": 0, "RED": 0}
	for _, item := range items {
		if _, ok := counts[item.TrafficLight]; ok {
			counts[item.TrafficLight]++
		}
	}
	total := float64(len(items))
	pct := func(n int) float64 {
		if total == 0 {
			return 0
		}
		return float64(n) / total * 100
	}
	buckets := []Bucket{
		{Label: "GREEN", Count: counts["GREEN"], Percentage: pct(counts["GREEN"])},
		{Label: "YELLOW", Count: counts["YELLOW"], Percentage: pct(counts["YELLOW"])},
		{Label: "RED", Count: counts["RED"], Percentage: pct(counts["RED"])},
	}
	c.JSON(http.StatusOK, gin.H{"dimension": dimension, "entity_type": entityType, "data": buckets})
}

// EmployeeRanking returns per-user performance scores scoped to the requester's role:
//
//	CA        → all non-admin employees
//	DIRECAO   → employees in departments under that direction
//	DEPARTAMENTO → employees in that specific department
func (DashboardController) EmployeeRanking(c *gin.Context) {
	role := util.ExtractRole(c)
	userID := util.ExtractUserID(c)

	perfDao := dao.PerformanceDao{}

	// Resolve the org ID the caller belongs to
	var orgID uint
	switch role {
	case "DIRECAO":
		geoDao := dao.DirecaoDao{}
		dir, err := geoDao.GetByResponsible(userID)
		if err == nil {
			orgID = dir.ID
		}
	case "DEPARTAMENTO":
		deptDao := dao.DepartamentoDao{}
		dept, err := deptDao.GetByResponsible(userID)
		if err == nil {
			orgID = dept.ID
		}
	}

	scores, err := perfDao.EmployeeRanking(role, orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	type Item struct {
		Rank         int     `json:"rank"`
		ID           uint    `json:"id"`
		Name         string  `json:"name"`
		Role         string  `json:"role"`
		Category     string  `json:"category"`
		DeptName     string  `json:"dept_name"`
		ExecScore    float64 `json:"execution_score"`
		GoalScore    float64 `json:"goal_score"`
		TotalScore   float64 `json:"total_score"`
		TrafficLight string  `json:"traffic_light"`
		MsTotal      int     `json:"ms_total"`
		MsDone       int     `json:"ms_done"`
	}

	items := make([]Item, 0, len(scores))
	for i, s := range scores {
		items = append(items, Item{
			Rank:         i + 1,
			ID:           s.UserID,
			Name:         s.Name,
			Role:         s.Role,
			Category:     s.Category,
			DeptName:     s.DeptName,
			ExecScore:    math.Round(s.ExecScore*10) / 10,
			GoalScore:    math.Round(s.GoalScore*10) / 10,
			TotalScore:   math.Round(s.TotalScore*10) / 10,
			TrafficLight: s.TrafficLight,
			MsTotal:      s.MsTotal,
			MsDone:       s.MsDone,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"role":    role,
		"org_id":  orgID,
		"ranking": items,
	})
}

func (DashboardController) Benchmark(c *gin.Context) {
	entityType := c.Query("entity_type")
	idA, _ := strconv.Atoi(c.Query("id_a"))
	idB, _ := strconv.Atoi(c.Query("id_b"))

	if idA == 0 || idB == 0 || entityType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "entity_type, id_a and id_b required"})
		return
	}

	perfDao := dao.PerformanceDao{}

	// Live score computation (supports ASC, REGIAO, DIRECAO, DEPARTAMENTO)
	computeLive := func(id uint) (exec, goal, total float64, light string) {
		switch entityType {
		case "ASC":
			return perfDao.ComputeScoreForScope("ASC", id)
		case "REGIAO":
			return perfDao.ComputeScoreForRegiao(id)
		case "DIRECAO":
			return perfDao.ComputeScoreForOwner("DIRECAO", id)
		case "DEPARTAMENTO":
			return perfDao.ComputeScoreForOwner("DEPARTAMENTO", id)
		default:
			return 0, 0, 0, "RED"
		}
	}

	nameA := resolveEntityName(entityType, uint(idA))
	nameB := resolveEntityName(entityType, uint(idB))

	eA, gA, tA, lA := computeLive(uint(idA))
	eB, gB, tB, lB := computeLive(uint(idB))

	round1 := func(v float64) float64 { return math.Round(v*10) / 10 }

	ratio := float64(0)
	var message string
	winner := ""
	switch {
	case tA > 0 && tB > 0:
		if tA >= tB {
			ratio = math.Round((tA/tB)*100) / 100
			pct := math.Round((ratio - 1) * 100)
			message = fmt.Sprintf("%s supera %s em %.0f%%", nameA, nameB, pct)
			winner = "A"
		} else {
			ratio = math.Round((tB/tA)*100) / 100
			pct := math.Round((ratio - 1) * 100)
			message = fmt.Sprintf("%s supera %s em %.0f%%", nameB, nameA, pct)
			winner = "B"
		}
	case tA == tB:
		message = "Desempenho idêntico"
	case tA > 0:
		winner = "A"
		message = fmt.Sprintf("%s lidera — %s sem dados", nameA, nameB)
	default:
		winner = "B"
		message = fmt.Sprintf("%s lidera — %s sem dados", nameB, nameA)
	}

	c.JSON(http.StatusOK, gin.H{
		"a": gin.H{
			"id": idA, "name": nameA,
			"execution_score": round1(eA), "goal_score": round1(gA),
			"total_score": round1(tA), "traffic_light": lA,
		},
		"b": gin.H{
			"id": idB, "name": nameB,
			"execution_score": round1(eB), "goal_score": round1(gB),
			"total_score": round1(tB), "traffic_light": lB,
		},
		"winner":  winner,
		"ratio":   ratio,
		"message": message,
	})
}

func resolveEntityName(entityType string, id uint) string {
	switch entityType {
	case "REGIAO":
		geoDao := dao.GeoDao{}
		r, err := geoDao.GetRegiaoByID(id)
		if err == nil {
			return r.Name
		}
	case "ASC":
		geoDao := dao.GeoDao{}
		a, err := geoDao.GetASCByID(id)
		if err == nil {
			return a.Name
		}
	case "PELOURO":
		pelouroDao := dao.PelouroDao{}
		p, err := pelouroDao.GetByID(id)
		if err == nil {
			return p.Name
		}
	case "DIRECAO":
		direcaoDao := dao.DirecaoDao{}
		d, err := direcaoDao.GetByID(id)
		if err == nil {
			return d.Name
		}
	case "DEPARTAMENTO":
		deptDao := dao.DepartamentoDao{}
		d, err := deptDao.GetByID(id)
		if err == nil {
			return d.Name
		}
	case "USER":
		userDao := dao.UserDao{}
		u, err := userDao.GetByID(id)
		if err == nil {
			return u.Name
		}
	}
	return fmt.Sprintf("%s #%d", entityType, id)
}

// DirecaoOverview returns a comprehensive director dashboard payload scoped to
// the direction the requesting user is responsible for.
//
// Response shape:
//
//	{
//	  direction:        { id, name, execution_score, goal_score, total_score, traffic_light }
//	  projects:         [ { id, title, status, execution_score, goal_score, total_score, traffic_light } ]
//	  stalled_tasks:    [ { id, title, project_title, days_elapsed, progress_pct } ]
//	  pending_blockers: [ { id, entity_type, entity_title, blocker_type, description, created_at } ]
//	  dept_scores:      [ { id, name, execution_score, goal_score, total_score, traffic_light } ]
//	}
func (DashboardController) DirecaoOverview(c *gin.Context) {
	userID := util.ExtractUserID(c)
	role := util.ExtractRole(c)

	if role != "DIRECAO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	// ── Resolve the direction this user is responsible for ───────────────────
	geoDao := dao.DirecaoDao{}
	dir, err := geoDao.GetByResponsible(userID)
	if err != nil {
		// No direction configured for this user: return empty overview so the
		// frontend can show a helpful "not configured" message.
		c.JSON(http.StatusOK, gin.H{
			"direction":        nil,
			"projects":         []interface{}{},
			"stalled_tasks":    []interface{}{},
			"pending_blockers": []interface{}{},
			"dept_scores":      []interface{}{},
		})
		return
	}

	perfDao := dao.PerformanceDao{}

	// ── Direction live score ─────────────────────────────────────────────────
	eDir, gDir, tDir, lDir := perfDao.ComputeScoreForOwner("DIRECAO", dir.ID)

	// ── Projects linked to this direction ────────────────────────────────────
	projDao := dao.ProjectDao{}
	projects, _ := projDao.ListByDirecao(dir.ID)

	type ProjectItem struct {
		ID     uint   `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	projItems := make([]ProjectItem, 0, len(projects))
	for _, p := range projects {
		projItems = append(projItems, ProjectItem{
			ID: p.ID, Title: p.Title, Status: p.Status,
		})
	}

	// ── Stalled tasks (owned by depts under this direction, zero progress, >7 days old) ──
	type StalledTask struct {
		ID           uint    `json:"id"`
		Title        string  `json:"title"`
		ProjectTitle string  `json:"project_title"`
		DaysElapsed  int     `json:"days_elapsed"`
		ProgressPct  float64 `json:"progress_pct"`
	}
	var stalledTasks []StalledTask
	dao.Database.Raw(`
		SELECT t.id,
		       t.title,
		       COALESCE(p.title, '') AS project_title,
		       GREATEST(0, EXTRACT(DAY FROM NOW() - t.start_date)::int) AS days_elapsed,
		       CASE WHEN (t.target_value - t.start_value) = 0 THEN 0
		            ELSE ROUND(((t.current_value - t.start_value) / (t.target_value - t.start_value)) * 100, 1)
		       END AS progress_pct
		FROM tasks t
		JOIN departamentos d ON d.id = t.owner_id AND t.owner_type = 'DEPARTAMENTO'
		LEFT JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
		WHERE d.direcao_id = ?
		  AND t.deleted_at IS NULL
		  AND t.current_value <= t.start_value
		  AND t.start_date < NOW() - INTERVAL '7 days'
		ORDER BY days_elapsed DESC
		LIMIT 10
	`, dir.ID).Scan(&stalledTasks)
	if stalledTasks == nil {
		stalledTasks = []StalledTask{}
	}

	// ── Pending blockers on tasks/milestones owned by depts in this direction ─
	type BlockerItem struct {
		ID          uint   `json:"id"`
		EntityType  string `json:"entity_type"`
		EntityTitle string `json:"entity_title"`
		BlockerType string `json:"blocker_type"`
		Description string `json:"description"`
		CreatedAt   string `json:"created_at"`
	}
	var blockerItems []BlockerItem
	dao.Database.Raw(`
		SELECT bl.id,
		       bl.entity_type,
		       COALESCE(bl.entity_title, t.title, '') AS entity_title,
		       bl.blocker_type,
		       bl.description,
		       bl.created_at::text AS created_at
		FROM blockers bl
		JOIN tasks t ON (
		       (bl.entity_type = 'TASK'      AND t.id = bl.entity_id)
		    OR (bl.entity_type = 'MILESTONE' AND t.id = (
		           SELECT task_id FROM milestones WHERE id = bl.entity_id LIMIT 1
		       ))
		)
		JOIN departamentos d ON d.id = t.owner_id AND t.owner_type = 'DEPARTAMENTO'
		WHERE d.direcao_id = ?
		  AND bl.status     = 'PENDING'
		  AND bl.deleted_at IS NULL
		  AND t.deleted_at  IS NULL
		ORDER BY bl.created_at DESC
		LIMIT 10
	`, dir.ID).Scan(&blockerItems)
	if blockerItems == nil {
		blockerItems = []BlockerItem{}
	}

	// ── Department scores ────────────────────────────────────────────────────
	deptDao := dao.DepartamentoDao{}
	depts, _ := deptDao.ListByDirecao(dir.ID)

	type DeptScore struct {
		ID             uint    `json:"id"`
		Name           string  `json:"name"`
		ExecutionScore float64 `json:"execution_score"`
		GoalScore      float64 `json:"goal_score"`
		TotalScore     float64 `json:"total_score"`
		TrafficLight   string  `json:"traffic_light"`
	}
	deptScores := make([]DeptScore, 0, len(depts))
	for _, d := range depts {
		e, g, t, l := perfDao.ComputeScoreForOwner("DEPARTAMENTO", d.ID)
		deptScores = append(deptScores, DeptScore{
			ID: d.ID, Name: d.Name,
			ExecutionScore: math.Round(e*10) / 10,
			GoalScore:      math.Round(g*10) / 10,
			TotalScore:     math.Round(t*10) / 10,
			TrafficLight:   l,
		})
	}

	// ── Direction's own tasks (owner_type = DIRECAO) with milestone progress ──
	// These are tasks the director can update directly (not dept-owned tasks).
	taskDao := dao.TaskDao{}
	dirTasks, _ := taskDao.ListByOwner("DIRECAO", dir.ID)

	type MsProgress struct {
		ID            uint      `json:"id"`
		Title         string    `json:"title"`
		Status        string    `json:"status"`
		PlannedDate   time.Time `json:"planned_date"`
		PlannedValue  float64   `json:"planned_value"`
		AchievedValue float64   `json:"achieved_value"`
		ScopeType     string    `json:"scope_type,omitempty"`
		ScopeID       *uint     `json:"scope_id,omitempty"`
		ScopeName     string    `json:"scope_name,omitempty"`
	}
	type DirTask struct {
		ID           uint         `json:"id"`
		Title        string       `json:"title"`
		ProjectTitle string       `json:"project_title"`
		ProgressPct  float64      `json:"progress_pct"`
		Milestones   []MsProgress `json:"milestones"`
	}
	dirTaskItems := make([]DirTask, 0, len(dirTasks))
	for _, t := range dirTasks {
		// Determine project title
		var projTitle string
		if t.ProjectID != 0 {
			var p model.Project
			if err := dao.Database.Select("title").First(&p, t.ProjectID).Error; err == nil {
				projTitle = p.Title
			}
		}
		// Compute progress %
		startVal := float64(0)
		if t.StartValue != nil {
			startVal = *t.StartValue
		}
		pct := float64(0)
		if denom := t.TargetValue - startVal; denom > 0 {
			pct = math.Round(((t.CurrentValue-startVal)/denom)*1000) / 10
			if pct > 100 {
				pct = 100
			}
		}
		// Milestones for this task
		milestoneDao := dao.MilestoneDao{}
		msList, _, _ := milestoneDao.ListByTask(t.ID, 0, 0)
		msItems := make([]MsProgress, 0, len(msList))
		for _, m := range msList {
			item := MsProgress{
				ID:            m.ID,
				Title:         m.Title,
				Status:        m.Status,
				PlannedDate:   m.PlannedDate,
				PlannedValue:  m.PlannedValue,
				AchievedValue: m.AchievedValue,
				ScopeType:     m.ScopeType,
				ScopeID:       m.ScopeID,
			}
			// Resolve scope name for display
			if m.ScopeType == "ASC" && m.ScopeID != nil {
				var asc model.ASC
				if dao.Database.Select("name").First(&asc, *m.ScopeID).Error == nil {
					item.ScopeName = asc.Name
				}
			} else if m.ScopeType == "REGIAO" && m.ScopeID != nil {
				var reg model.Regiao
				if dao.Database.Select("name").First(&reg, *m.ScopeID).Error == nil {
					item.ScopeName = reg.Name
				}
			}
			msItems = append(msItems, item)
		}
		dirTaskItems = append(dirTaskItems, DirTask{
			ID:           t.ID,
			Title:        t.Title,
			ProjectTitle: projTitle,
			ProgressPct:  pct,
			Milestones:   msItems,
		})
	}

	// ── Director's personal score ────────────────────────────────────────────
	myScore := perfDao.ScoreForUser(userID)

	c.JSON(http.StatusOK, gin.H{
		"direction": gin.H{
			"id":              dir.ID,
			"name":            dir.Name,
			"execution_score": math.Round(eDir*10) / 10,
			"goal_score":      math.Round(gDir*10) / 10,
			"total_score":     math.Round(tDir*10) / 10,
			"traffic_light":   lDir,
		},
		"my_score": gin.H{
			"execution_score": math.Round(myScore.ExecScore*10) / 10,
			"goal_score":      math.Round(myScore.GoalScore*10) / 10,
			"total_score":     math.Round(myScore.TotalScore*10) / 10,
			"traffic_light":   myScore.TrafficLight,
			"ms_total":        myScore.MsTotal,
			"ms_done":         myScore.MsDone,
		},
		"projects":         projItems,
		"dir_tasks":        dirTaskItems,
		"stalled_tasks":    stalledTasks,
		"pending_blockers": blockerItems,
		"dept_scores":      deptScores,
	})
}

// DepartamentoOverview returns a comprehensive department-head dashboard payload
// scoped to the department the requesting user is responsible for.
func (DashboardController) DepartamentoOverview(c *gin.Context) {
	userID := util.ExtractUserID(c)
	role := util.ExtractRole(c)

	if role != "DEPARTAMENTO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	// ── Resolve the department this user is responsible for ──────────────────
	deptDao := dao.DepartamentoDao{}
	dept, err := deptDao.GetByResponsible(userID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"department":       nil,
			"tasks":            []interface{}{},
			"pending_blockers": []interface{}{},
			"stats": gin.H{
				"total":         0,
				"unassigned":    0,
				"from_director": 0,
				"active":        0,
			},
		})
		return
	}

	perfDao := dao.PerformanceDao{}

	// ── Department live score ────────────────────────────────────────────────
	eDept, gDept, tDept, lDept := perfDao.ComputeScoreForOwner("DEPARTAMENTO", dept.ID)

	round1 := func(v float64) float64 { return math.Round(v*10) / 10 }

	// ── Fetch direcao_id ─────────────────────────────────────────────────────
	var direcaoID uint
	dao.Database.Raw("SELECT direcao_id FROM departamentos WHERE id = ?", dept.ID).Scan(&direcaoID)

	// ── Tasks owned by this department ───────────────────────────────────────
	type TaskRow struct {
		ID            uint    `json:"id"`
		Title         string  `json:"title"`
		Status        string  `json:"status"`
		ProjectTitle  string  `json:"project_title"`
		GoalLabel     string  `json:"goal_label"`
		Frequency     string  `json:"frequency"`
		CreatedBy     uint    `json:"created_by"`
		CreatorRole   string  `json:"creator_role"`
		ProgressPct   float64 `json:"progress_pct"`
		DaysElapsed   int     `json:"days_elapsed"`
		DaysRemaining int     `json:"days_remaining"`
	}
	var taskRows []TaskRow
	dao.Database.Raw(`
		SELECT t.id, t.title, t.status,
		       COALESCE(p.title,'') AS project_title,
		       COALESCE(t.goal_label,'') AS goal_label,
		       COALESCE(t.frequency,'') AS frequency,
		       t.created_by,
		       u.role AS creator_role,
		       CASE WHEN (t.target_value - COALESCE(t.start_value,0)) = 0 THEN 0
		            ELSE ROUND(((t.current_value - COALESCE(t.start_value,0)) / (t.target_value - COALESCE(t.start_value,0)))*100, 1)
		       END AS progress_pct,
		       GREATEST(0, EXTRACT(DAY FROM NOW() - t.start_date)::int) AS days_elapsed,
		       CASE WHEN t.end_date IS NULL THEN 0
		            ELSE GREATEST(0, EXTRACT(DAY FROM t.end_date - NOW())::int)
		       END AS days_remaining
		FROM tasks t
		LEFT JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
		LEFT JOIN users u ON u.id = t.created_by
		WHERE t.owner_type = 'DEPARTAMENTO' AND t.owner_id = ?
		  AND t.deleted_at IS NULL
		ORDER BY t.created_at DESC
	`, dept.ID).Scan(&taskRows)
	if taskRows == nil {
		taskRows = []TaskRow{}
	}

	// ── Fetch assignees per task ─────────────────────────────────────────────
	type Assignee struct {
		UserID  uint   `json:"user_id"`
		Name    string `json:"name"`
		MsTotal int    `json:"ms_total"`
		MsDone  int    `json:"ms_done"`
	}

	type TaskItem struct {
		ID             uint       `json:"id"`
		Title          string     `json:"title"`
		Status         string     `json:"status"`
		ProjectTitle   string     `json:"project_title"`
		GoalLabel      string     `json:"goal_label"`
		Frequency      string     `json:"frequency"`
		ProgressPct    float64    `json:"progress_pct"`
		DaysElapsed    int        `json:"days_elapsed"`
		DaysRemaining  int        `json:"days_remaining"`
		IsFromDirector bool       `json:"is_from_director"`
		IsUnassigned   bool       `json:"is_unassigned"`
		Assignees      []Assignee `json:"assignees"`
	}

	var (
		taskItems         []TaskItem
		unassignedCount   int
		fromDirectorCount int
		activeCount       int
	)

	for _, tr := range taskRows {
		var assignees []Assignee
		dao.Database.Raw(`
			SELECT COALESCE(m.created_by, m.updated_by) AS user_id, u.name,
			       COUNT(*) AS ms_total,
			       SUM(CASE WHEN m.status='DONE' THEN 1 ELSE 0 END) AS ms_done
			FROM milestones m
			JOIN users u ON u.id = COALESCE(m.created_by, m.updated_by)
			WHERE m.task_id = ? AND m.deleted_at IS NULL
			  AND COALESCE(m.created_by, m.updated_by) IS NOT NULL
			GROUP BY COALESCE(m.created_by, m.updated_by), u.name
		`, tr.ID).Scan(&assignees)
		if assignees == nil {
			assignees = []Assignee{}
		}

		isFromDirector := tr.CreatorRole == "DIRECAO"
		isUnassigned := len(assignees) == 0

		if isUnassigned {
			unassignedCount++
		}
		if isFromDirector {
			fromDirectorCount++
		}
		if tr.Status == "ACTIVE" {
			activeCount++
		}

		taskItems = append(taskItems, TaskItem{
			ID:             tr.ID,
			Title:          tr.Title,
			Status:         tr.Status,
			ProjectTitle:   tr.ProjectTitle,
			GoalLabel:      tr.GoalLabel,
			Frequency:      tr.Frequency,
			ProgressPct:    tr.ProgressPct,
			DaysElapsed:    tr.DaysElapsed,
			DaysRemaining:  tr.DaysRemaining,
			IsFromDirector: isFromDirector,
			IsUnassigned:   isUnassigned,
			Assignees:      assignees,
		})
	}
	if taskItems == nil {
		taskItems = []TaskItem{}
	}

	// ── Pending blockers for this department ─────────────────────────────────
	type BlockerItem struct {
		ID          uint   `json:"id"`
		EntityType  string `json:"entity_type"`
		EntityTitle string `json:"entity_title"`
		BlockerType string `json:"blocker_type"`
		Description string `json:"description"`
		CreatedAt   string `json:"created_at"`
	}
	var blockerItems []BlockerItem
	dao.Database.Raw(`
		SELECT bl.id, bl.entity_type,
		       COALESCE(bl.entity_title, t.title, '') AS entity_title,
		       bl.blocker_type, bl.description,
		       bl.created_at::text AS created_at
		FROM blockers bl
		JOIN tasks t ON (
		    (bl.entity_type = 'TASK' AND t.id = bl.entity_id)
		 OR (bl.entity_type = 'MILESTONE' AND t.id = (SELECT task_id FROM milestones WHERE id = bl.entity_id LIMIT 1))
		)
		WHERE t.owner_type = 'DEPARTAMENTO' AND t.owner_id = ?
		  AND bl.status = 'PENDING' AND bl.deleted_at IS NULL AND t.deleted_at IS NULL
		ORDER BY bl.created_at DESC LIMIT 10
	`, dept.ID).Scan(&blockerItems)
	if blockerItems == nil {
		blockerItems = []BlockerItem{}
	}

	// ── Department users (for milestone assignment) ──────────────────────────
	type DeptUser struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
		Role string `json:"role"`
	}
	var deptUsers []DeptUser
	dao.Database.Raw(`
		SELECT u.id, u.name, u.role FROM users u WHERE u.id = ?
		UNION
		SELECT u.id, u.name, u.role FROM users u
		JOIN departamento_users du ON du.user_id = u.id
		WHERE du.departamento_id = ?
		ORDER BY name ASC
	`, userID, dept.ID).Scan(&deptUsers)
	if deptUsers == nil {
		deptUsers = []DeptUser{}
	}

	c.JSON(http.StatusOK, gin.H{
		"department": gin.H{
			"id":              dept.ID,
			"name":            dept.Name,
			"execution_score": round1(eDept),
			"goal_score":      round1(gDept),
			"total_score":     round1(tDept),
			"traffic_light":   lDept,
			"direcao_id":      direcaoID,
		},
		"tasks":            taskItems,
		"pending_blockers": blockerItems,
		"stats": gin.H{
			"total":         len(taskItems),
			"unassigned":    unassignedCount,
			"from_director": fromDirectorCount,
			"active":        activeCount,
		},
		"users": deptUsers,
	})
}

// MemberOverview returns a dashboard payload for a regular DEPARTAMENTO-role
// member (not the department head). It scopes data to milestones assigned to
// the logged-in user, the tasks those milestones belong to, and the user's
// personal performance score.
func (DashboardController) MemberOverview(c *gin.Context) {
	userID := util.ExtractUserID(c)
	role := util.ExtractRole(c)

	if role != "DEPARTAMENTO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	perfDao := dao.PerformanceDao{}

	// ── Personal score ───────────────────────────────────────────────────────
	myScore := perfDao.ScoreForUser(userID)

	// ── Department membership (may be in 0 or more departments) ─────────────
	type DeptInfo struct {
		ID   uint   `json:"id"`
		Name string `json:"name"`
	}
	var departments []DeptInfo
	dao.Database.Raw(`
		SELECT d.id, d.name
		FROM departamentos d
		JOIN departamento_users du ON du.departamento_id = d.id
		WHERE du.user_id = ? AND d.deleted_at IS NULL
		ORDER BY d.name
	`, userID).Scan(&departments)

	// ── Milestones assigned to this user ─────────────────────────────────────
	type MsRow struct {
		ID            uint    `json:"id"`
		Title         string  `json:"title"`
		Status        string  `json:"status"`
		Frequency     string  `json:"frequency"`
		PlannedDate   string  `json:"planned_date"`
		PlannedValue  float64 `json:"planned_value"`
		AchievedValue float64 `json:"achieved_value"`
		TaskID        uint    `json:"task_id"`
		TaskTitle     string  `json:"task_title"`
		GoalLabel     string  `json:"goal_label"`
		ProjectTitle  string  `json:"project_title"`
		ScopeType     string  `json:"scope_type"`
		ScopeName     string  `json:"scope_name"`
	}
	var milestones []MsRow
	dao.Database.Raw(`
		SELECT m.id, m.title, m.status,
		       COALESCE(m.frequency, t.frequency, '') AS frequency,
		       COALESCE(TO_CHAR(m.planned_date, 'YYYY-MM-DD'), '') AS planned_date,
		       COALESCE(m.planned_value, 0)   AS planned_value,
		       COALESCE(m.achieved_value, 0)  AS achieved_value,
		       t.id   AS task_id,
		       t.title AS task_title,
		       COALESCE(t.goal_label, '') AS goal_label,
		       COALESCE(p.title, '')      AS project_title,
		       COALESCE(m.scope_type, '') AS scope_type,
		       COALESCE(
		           CASE
		               WHEN m.scope_type = 'ASC'      THEN (SELECT name FROM ascs    WHERE id = m.scope_id LIMIT 1)
		               WHEN m.scope_type = 'REGIONAL' THEN (SELECT name FROM regiaos WHERE id = m.scope_id LIMIT 1)
		               ELSE ''
		           END, ''
		       ) AS scope_name
		FROM milestones m
		JOIN tasks t ON t.id = m.task_id AND t.deleted_at IS NULL
		LEFT JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
		WHERE m.assigned_to = ? AND m.deleted_at IS NULL
		ORDER BY m.planned_date ASC NULLS LAST
	`, userID).Scan(&milestones)
	if milestones == nil {
		milestones = []MsRow{}
	}

	// ── Stats ─────────────────────────────────────────────────────────────────
	var msTotal, msDone, msPending, msBlocked int
	for _, m := range milestones {
		msTotal++
		switch m.Status {
		case "DONE":
			msDone++
		case "BLOCKED":
			msBlocked++
		default:
			msPending++
		}
	}

	// ── Monthly trend (last 6 months) ─────────────────────────────────────────
	type MonthlyRow struct {
		Month string `json:"month"`
		Done  int    `json:"done"`
		Total int    `json:"total"`
	}
	var monthly []MonthlyRow
	dao.Database.Raw(`
		SELECT TO_CHAR(DATE_TRUNC('month', m.planned_date), 'YYYY-MM') AS month,
		       SUM(CASE WHEN m.status = 'DONE' THEN 1 ELSE 0 END)::int AS done,
		       COUNT(*)::int AS total
		FROM milestones m
		WHERE m.assigned_to = ?
		  AND m.deleted_at IS NULL
		  AND m.planned_date >= NOW() - INTERVAL '6 months'
		GROUP BY DATE_TRUNC('month', m.planned_date)
		ORDER BY month ASC
	`, userID).Scan(&monthly)
	if monthly == nil {
		monthly = []MonthlyRow{}
	}

	// ── Projects the user is involved in ─────────────────────────────────────
	type ProjectRow struct {
		ID      uint   `json:"id"`
		Title   string `json:"title"`
		MsCount int    `json:"ms_count"`
	}
	var projects []ProjectRow
	dao.Database.Raw(`
		SELECT p.id, p.title,
		       COUNT(m.id)::int AS ms_count
		FROM milestones m
		JOIN tasks t ON t.id = m.task_id AND t.deleted_at IS NULL
		JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
		WHERE m.assigned_to = ? AND m.deleted_at IS NULL
		GROUP BY p.id, p.title
		ORDER BY ms_count DESC
	`, userID).Scan(&projects)
	if projects == nil {
		projects = []ProjectRow{}
	}

	// ── Unique ASC IDs from this member's milestones ──────────────────────────
	ascIDSet := map[uint]bool{}
	for _, m := range milestones {
		if m.ScopeType == "ASC" && m.ScopeName != "" {
			// ScopeID is not in MsRow directly; query separately
		}
	}
	// Query directly: distinct ASC scope_ids for milestones assigned to this user
	var rawAscIDs []uint
	dao.Database.Raw(`
		SELECT DISTINCT m.scope_id
		FROM milestones m
		WHERE m.assigned_to = ? AND m.scope_type = 'ASC' AND m.scope_id IS NOT NULL AND m.deleted_at IS NULL
	`, userID).Scan(&rawAscIDs)
	for _, id := range rawAscIDs {
		ascIDSet[id] = true
	}
	ascIDs := make([]uint, 0, len(ascIDSet))
	for id := range ascIDSet {
		ascIDs = append(ascIDs, id)
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":   userID,
			"name": myScore.Name,
		},
		"departments": departments,
		"score": gin.H{
			"execution_score": math.Round(myScore.ExecScore*10) / 10,
			"goal_score":      math.Round(myScore.GoalScore*10) / 10,
			"total_score":     math.Round(myScore.TotalScore*10) / 10,
			"traffic_light":   myScore.TrafficLight,
			"ms_total":        myScore.MsTotal,
			"ms_done":         myScore.MsDone,
		},
		"milestones": milestones,
		"stats": gin.H{
			"total":   msTotal,
			"done":    msDone,
			"pending": msPending,
			"blocked": msBlocked,
		},
		"monthly":  monthly,
		"projects": projects,
		"asc_ids":  ascIDs,
	})
}

// RegionalOverview returns a read-only dashboard payload for DIRECAO users who
// are responsible for a Região (not a Direcção).
//
// Response shape:
//
//	{
//	  regiao:   { id, name, execution_score, goal_score, total_score, traffic_light }
//	  ascs:     [ { id, name, execution_score, goal_score, total_score, traffic_light } ]
//	  projects: [ { id, title, status } ]
//	  stats:    { total_milestones, done, pending, blocked }
//	}
func (DashboardController) RegionalOverview(c *gin.Context) {
	userID := util.ExtractUserID(c)
	role := util.ExtractRole(c)

	if role != "DIRECAO" {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}

	geoDao := dao.GeoDao{}
	reg, err := geoDao.GetRegiaoByResponsible(userID)
	if err != nil {
		// Not a regional director — return nil so frontend can decide
		c.JSON(http.StatusOK, gin.H{"regiao": nil, "ascs": []interface{}{}, "projects": []interface{}{}, "stats": gin.H{"total_milestones": 0, "done": 0, "pending": 0, "blocked": 0}})
		return
	}

	perfDao := dao.PerformanceDao{}
	eReg, gReg, tReg, lReg := perfDao.ComputeScoreForRegiao(reg.ID)

	round1 := func(v float64) float64 { return math.Round(v*10) / 10 }

	// ── ASCs in this region with scores ─────────────────────────────────────
	ascs, _ := geoDao.ListASCsByRegiao(reg.ID)
	type ASCScore struct {
		ID             uint    `json:"id"`
		Name           string  `json:"name"`
		ExecutionScore float64 `json:"execution_score"`
		GoalScore      float64 `json:"goal_score"`
		TotalScore     float64 `json:"total_score"`
		TrafficLight   string  `json:"traffic_light"`
	}
	ascScores := make([]ASCScore, 0, len(ascs))
	for _, a := range ascs {
		e, g, t, l := perfDao.ComputeScoreForOwner("ASC", a.ID)
		ascScores = append(ascScores, ASCScore{
			ID: a.ID, Name: a.Name,
			ExecutionScore: round1(e), GoalScore: round1(g),
			TotalScore: round1(t), TrafficLight: l,
		})
	}

	// ── Build ASC ID list ────────────────────────────────────────────────────
	ascIDList := make([]uint, 0, len(ascs))
	for _, a := range ascs {
		ascIDList = append(ascIDList, a.ID)
	}

	// ── Projects: collect via REGIAO scope + ASC scope separately ────────────
	// GORM expands slices for IN clauses; avoiding ANY(?) which GORM cannot expand.
	type ProjectItem struct {
		ID     uint   `json:"id"`
		Title  string `json:"title"`
		Status string `json:"status"`
	}
	seen := map[uint]bool{}
	rawProjects := []ProjectItem{}

	// 1. Milestones scoped directly to this regiao
	var regProjects []ProjectItem
	dao.Database.Raw(`
		SELECT DISTINCT p.id, p.title, p.status
		FROM projects p
		JOIN tasks t ON t.project_id = p.id AND t.deleted_at IS NULL
		JOIN milestones m ON m.task_id = t.id AND m.deleted_at IS NULL
		WHERE p.deleted_at IS NULL
		  AND m.scope_type = 'REGIAO'
		  AND m.scope_id   = ?
		ORDER BY p.title ASC
	`, reg.ID).Scan(&regProjects)
	for _, p := range regProjects {
		if !seen[p.ID] {
			seen[p.ID] = true
			rawProjects = append(rawProjects, p)
		}
	}

	// 2. Milestones scoped to any ASC within this regiao (only if there are ASCs)
	if len(ascIDList) > 0 {
		var ascProjects []ProjectItem
		dao.Database.Raw(`
			SELECT DISTINCT p.id, p.title, p.status
			FROM projects p
			JOIN tasks t ON t.project_id = p.id AND t.deleted_at IS NULL
			JOIN milestones m ON m.task_id = t.id AND m.deleted_at IS NULL
			WHERE p.deleted_at IS NULL
			  AND m.scope_type = 'ASC'
			  AND m.scope_id   IN ?
			ORDER BY p.title ASC
		`, ascIDList).Scan(&ascProjects)
		for _, p := range ascProjects {
			if !seen[p.ID] {
				seen[p.ID] = true
				rawProjects = append(rawProjects, p)
			}
		}
	}

	// ── Milestone stats: REGIAO scope + ASC scope summed ────────────────────
	// countMilestones returns the total matching a WHERE fragment across REGIAO
	// and (optionally) ASC scopes. We use two queries and add the results to
	// avoid ANY(?) which GORM does not expand from Go slices.
	countMs := func(statusFilter string) int64 {
		statusClause := ""
		if statusFilter != "" {
			statusClause = " AND m.status = '" + statusFilter + "'"
		}
		var nReg int64
		dao.Database.Raw(`
			SELECT COUNT(*) FROM milestones m
			JOIN tasks t ON t.id = m.task_id AND t.deleted_at IS NULL
			WHERE m.deleted_at IS NULL`+statusClause+`
			  AND m.scope_type = 'REGIAO' AND m.scope_id = ?
		`, reg.ID).Scan(&nReg)
		if len(ascIDList) == 0 {
			return nReg
		}
		var nAsc int64
		dao.Database.Raw(`
			SELECT COUNT(*) FROM milestones m
			JOIN tasks t ON t.id = m.task_id AND t.deleted_at IS NULL
			WHERE m.deleted_at IS NULL`+statusClause+`
			  AND m.scope_type = 'ASC' AND m.scope_id IN ?
		`, ascIDList).Scan(&nAsc)
		return nReg + nAsc
	}

	msTotal := countMs("")
	msDone := countMs("DONE")
	msPending := countMs("PENDING")
	msBlocked := countMs("BLOCKED")

	// ── Milestones: fetch recent milestones scoped to this region or its ASCs ──
	// NOTE: PlannedDate must be time.Time — scanning a TIMESTAMPTZ into string fails.
	// AchievedValue must be float64 (non-pointer) matching the DB column default 0.
	type MilestoneItem struct {
		ID            uint      `json:"id"`
		Title         string    `json:"title"`
		Status        string    `json:"status"`
		Frequency     string    `json:"frequency"`
		PlannedDate   time.Time `json:"planned_date"`
		PlannedValue  float64   `json:"planned_value"`
		AchievedValue float64   `json:"achieved_value"`
		ScopeType     string    `json:"scope_type"`
		ScopeID       uint      `json:"scope_id"`
		ScopeName     string    `json:"scope_name,omitempty"`
		TaskID        uint      `json:"task_id"`
		TaskTitle     string    `json:"task_title"`
		ProjectID     uint      `json:"project_id"`
		ProjectTitle  string    `json:"project_title"`
	}

	var milestones []MilestoneItem

	// Milestones scoped directly to this regiao
	var regiaoMs []MilestoneItem
	dao.Database.Raw(`
		SELECT m.id, m.title, m.status, COALESCE(m.frequency, t.frequency, '') AS frequency, m.planned_date, m.planned_value,
		       m.achieved_value, m.scope_type, m.scope_id,
		       ? AS scope_name,
		       t.id AS task_id, t.title AS task_title,
		       p.id AS project_id, p.title AS project_title
		FROM milestones m
		JOIN tasks t ON t.id = m.task_id AND t.deleted_at IS NULL
		JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
		WHERE m.deleted_at IS NULL
		  AND m.scope_type = 'REGIAO' AND m.scope_id = ?
		ORDER BY m.planned_date DESC
		LIMIT 50
	`, reg.Name, reg.ID).Scan(&regiaoMs)
	milestones = append(milestones, regiaoMs...)

	// Milestones scoped to ASCs in this region
	if len(ascIDList) > 0 {
		// Build a map of ASC id → name for scope_name lookup
		ascNameMap := make(map[uint]string, len(ascs))
		for _, a := range ascs {
			ascNameMap[a.ID] = a.Name
		}

		var ascMs []MilestoneItem
		dao.Database.Raw(`
			SELECT m.id, m.title, m.status, COALESCE(m.frequency, t.frequency, '') AS frequency, m.planned_date, m.planned_value,
			       m.achieved_value, m.scope_type, m.scope_id,
			       t.id AS task_id, t.title AS task_title,
			       p.id AS project_id, p.title AS project_title
			FROM milestones m
			JOIN tasks t ON t.id = m.task_id AND t.deleted_at IS NULL
			JOIN projects p ON p.id = t.project_id AND p.deleted_at IS NULL
			WHERE m.deleted_at IS NULL
			  AND m.scope_type = 'ASC' AND m.scope_id IN ?
			ORDER BY m.planned_date DESC
			LIMIT 100
		`, ascIDList).Scan(&ascMs)

		// Annotate scope_name from the in-memory map
		for i := range ascMs {
			if name, ok := ascNameMap[ascMs[i].ScopeID]; ok {
				ascMs[i].ScopeName = name
			}
		}
		milestones = append(milestones, ascMs...)
	}

	if milestones == nil {
		milestones = []MilestoneItem{}
	}

	c.JSON(http.StatusOK, gin.H{
		"regiao": gin.H{
			"id":              reg.ID,
			"name":            reg.Name,
			"execution_score": round1(eReg),
			"goal_score":      round1(gReg),
			"total_score":     round1(tReg),
			"traffic_light":   lReg,
		},
		"ascs":       ascScores,
		"projects":   rawProjects,
		"milestones": milestones,
		"stats": gin.H{
			"total_milestones": msTotal,
			"done":             msDone,
			"pending":          msPending,
			"blocked":          msBlocked,
		},
	})
}
