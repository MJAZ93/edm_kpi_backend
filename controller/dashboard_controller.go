package controller

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"kpi-backend/dao"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type DashboardController struct{}

func (DashboardController) Summary(c *gin.Context) {
	dashDao := dao.DashboardDao{}
	summary := dashDao.GetSummary()
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
		ID            uint    `json:"id"`
		Name          string  `json:"name"`
		Type          string  `json:"type"`
		ExecutionScore float64 `json:"execution_score"`
		GoalScore     float64 `json:"goal_score"`
		TotalScore    float64 `json:"total_score"`
		TrafficLight  string  `json:"traffic_light"`
	}

	var items []DrillItem

	switch level {
	case "NATIONAL":
		// Show regions
		geoDao := dao.GeoDao{}
		regioes, _ := geoDao.GetAllRegioes()
		var ids []uint
		for _, r := range regioes {
			ids = append(ids, r.ID)
		}
		scores, _ := perfDao.GetScores("REGIAO", ids, period)
		scoreMap := make(map[uint]dao.PerformanceCacheAlias)
		for _, s := range scores {
			scoreMap[s.EntityID] = dao.PerformanceCacheAlias(s)
		}
		for _, r := range regioes {
			item := DrillItem{ID: r.ID, Name: r.Name, Type: "REGIAO"}
			if s, ok := scoreMap[r.ID]; ok {
				item.ExecutionScore = s.ExecutionScore
				item.GoalScore = s.GoalScore
				item.TotalScore = s.TotalScore
				item.TrafficLight = s.TrafficLight
			}
			items = append(items, item)
		}

	case "REGIONAL":
		geoDao := dao.GeoDao{}
		ascs, _ := geoDao.ListASCsByRegiao(uint(id))
		var ids []uint
		for _, a := range ascs {
			ids = append(ids, a.ID)
		}
		scores, _ := perfDao.GetScores("ASC", ids, period)
		scoreMap := make(map[uint]dao.PerformanceCacheAlias)
		for _, s := range scores {
			scoreMap[s.EntityID] = dao.PerformanceCacheAlias(s)
		}
		for _, a := range ascs {
			item := DrillItem{ID: a.ID, Name: a.Name, Type: "ASC"}
			if s, ok := scoreMap[a.ID]; ok {
				item.ExecutionScore = s.ExecutionScore
				item.GoalScore = s.GoalScore
				item.TotalScore = s.TotalScore
				item.TrafficLight = s.TrafficLight
			}
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
	periodStr := c.DefaultQuery("period", time.Now().Format("2006-01"))
	period, _ := time.Parse("2006-01", periodStr)

	geoDao := dao.GeoDao{}
	perfDao := dao.PerformanceDao{}

	type Feature struct {
		Type       string      `json:"type"`
		Geometry   interface{} `json:"geometry,omitempty"`
		Properties interface{} `json:"properties"`
	}

	var features []Feature

	if level == "REGIONAL" {
		regioes, _ := geoDao.GetAllRegioes()
		var ids []uint
		for _, r := range regioes {
			ids = append(ids, r.ID)
		}
		scores, _ := perfDao.GetScores("REGIAO", ids, period)
		scoreMap := make(map[uint]float64)
		lightMap := make(map[uint]string)
		for _, s := range scores {
			scoreMap[s.EntityID] = s.TotalScore
			lightMap[s.EntityID] = s.TrafficLight
		}

		for _, r := range regioes {
			features = append(features, Feature{
				Type: "Feature",
				Properties: gin.H{
					"id":            r.ID,
					"name":          r.Name,
					"total_score":   scoreMap[r.ID],
					"traffic_light": lightMap[r.ID],
				},
			})
		}
	} else {
		ascs, _ := geoDao.GetAllASCs()
		var ids []uint
		for _, a := range ascs {
			ids = append(ids, a.ID)
		}
		scores, _ := perfDao.GetScores("ASC", ids, period)
		scoreMap := make(map[uint]float64)
		lightMap := make(map[uint]string)
		for _, s := range scores {
			scoreMap[s.EntityID] = s.TotalScore
			lightMap[s.EntityID] = s.TrafficLight
		}

		for _, a := range ascs {
			features = append(features, Feature{
				Type: "Feature",
				Properties: gin.H{
					"id":            a.ID,
					"name":          a.Name,
					"total_score":   scoreMap[a.ID],
					"traffic_light": lightMap[a.ID],
				},
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"type":     "FeatureCollection",
		"features": features,
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

	period, _ := time.Parse("2006-01", periodStr)
	limit, _ := strconv.Atoi(limitStr)

	perfDao := dao.PerformanceDao{}
	list, err := perfDao.GetTopPerformers(entityType, period, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	type RankedItem struct {
		Rank         int     `json:"rank"`
		ID           uint    `json:"id"`
		Name         string  `json:"name"`
		TotalScore   float64 `json:"total_score"`
		TrafficLight string  `json:"traffic_light"`
	}

	var ranked []RankedItem
	for i, item := range list {
		name := resolveEntityName(entityType, item.EntityID)
		ranked = append(ranked, RankedItem{
			Rank:         i + 1,
			ID:           item.EntityID,
			Name:         name,
			TotalScore:   item.TotalScore,
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
	dimension := c.DefaultQuery("dimension", "BY_STATUS")
	dashDao := dao.DashboardDao{}
	data := dashDao.GetDistribution(dimension)
	c.JSON(http.StatusOK, gin.H{"dimension": dimension, "data": data})
}

func (DashboardController) Benchmark(c *gin.Context) {
	entityType := c.Query("entity_type")
	idA, _ := strconv.Atoi(c.Query("id_a"))
	idB, _ := strconv.Atoi(c.Query("id_b"))
	periodStr := c.DefaultQuery("period", time.Now().Format("2006-01"))
	period, _ := time.Parse("2006-01", periodStr)

	perfDao := dao.PerformanceDao{}
	scoreA, errA := perfDao.GetScore(entityType, uint(idA), period)
	scoreB, errB := perfDao.GetScore(entityType, uint(idB), period)

	if errA != nil || errB != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "performance data not found for one or both entities"})
		return
	}

	nameA := resolveEntityName(entityType, uint(idA))
	nameB := resolveEntityName(entityType, uint(idB))

	ratio := float64(0)
	var message string
	if scoreA.TotalScore > 0 && scoreB.TotalScore > 0 {
		if scoreA.TotalScore >= scoreB.TotalScore {
			ratio = math.Round((scoreA.TotalScore/scoreB.TotalScore)*100) / 100
			pct := math.Round((ratio - 1) * 100)
			message = fmt.Sprintf("%s é %.0f%% mais eficiente que %s", nameA, pct, nameB)
		} else {
			ratio = math.Round((scoreB.TotalScore/scoreA.TotalScore)*100) / 100
			pct := math.Round((ratio - 1) * 100)
			message = fmt.Sprintf("%s é %.0f%% mais eficiente que %s", nameB, pct, nameA)
		}
	}

	c.JSON(http.StatusOK, dao.BenchmarkResult{
		A:       dao.BenchmarkEntity{ID: uint(idA), Name: nameA, TotalScore: scoreA.TotalScore},
		B:       dao.BenchmarkEntity{ID: uint(idB), Name: nameB, TotalScore: scoreB.TotalScore},
		Ratio:   ratio,
		Message: message,
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
