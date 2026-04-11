package controller

import (
	"fmt"
	"net/http"
	"strconv"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type TaskController struct{}

func (TaskController) ListByProject(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Query("project_id"))
	params := util.ParsePagination(c)
	scope := dao.ResolveScope(util.ExtractUserID(c), util.ExtractRole(c))
	taskDao := dao.TaskDao{}
	list, total, err := taskDao.ListByProjectScoped(uint(projectID), params.Page, params.Limit, &scope)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, util.NewPaginatedResponse(list, total, params))
}

type TaskInput struct {
	Title           string       `json:"title" binding:"required"`
	Description     string       `json:"description"`
	OwnerType       string       `json:"owner_type" binding:"required,oneof=DIRECAO DEPARTAMENTO"`
	OwnerID         uint         `json:"owner_id" binding:"required"`
	AssignedTo      *uint        `json:"assigned_to"`
	Frequency       string       `json:"frequency" binding:"required,oneof=DAILY WEEKLY MONTHLY QUARTERLY BIANNUAL ANNUAL"`
	GoalLabel       string       `json:"goal_label"`
	StartValue      *float64     `json:"start_value"`
	TargetValue     float64      `json:"target_value" binding:"required"`
	AggregationType string       `json:"aggregation_type"` // SUM_UP (default), SUM_DOWN, AVG
	Weight          float64      `json:"weight"`
	StartDate       *string      `json:"start_date"`
	EndDate         *string      `json:"end_date"`
	ParentTaskID    *uint        `json:"parent_task_id"`
	Scopes          []ScopeInput `json:"scopes"`
}

type ScopeInput struct {
	ScopeType string `json:"scope_type" binding:"required,oneof=NACIONAL REGIONAL ASC"`
	ScopeID   *uint  `json:"scope_id"`
}

func validateTaskAssignee(ownerType string, ownerID uint, assignedTo *uint) error {
	if assignedTo == nil {
		return nil
	}
	if ownerType != "DEPARTAMENTO" {
		return fmt.Errorf("assigned_to requires owner_type DEPARTAMENTO")
	}

	deptDao := dao.DepartamentoDao{}
	dept, err := deptDao.GetByID(ownerID)
	if err != nil {
		return fmt.Errorf("department not found")
	}

	if dept.ResponsibleID != nil && *dept.ResponsibleID == *assignedTo {
		return nil
	}
	for _, u := range dept.Users {
		if u.ID == *assignedTo {
			return nil
		}
	}

	return fmt.Errorf("assigned user must belong to selected department")
}

func (TaskController) Create(c *gin.Context) {
	projectID, _ := strconv.Atoi(c.Query("project_id"))

	var input TaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}
	if err := validateTaskAssignee(input.OwnerType, input.OwnerID, input.AssignedTo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	aggType := input.AggregationType
	if aggType == "" {
		aggType = "SUM_UP"
	}

	task := model.Task{
		ProjectID:       uint(projectID),
		Title:           input.Title,
		Description:     input.Description,
		OwnerType:       input.OwnerType,
		OwnerID:         input.OwnerID,
		AssignedTo:      input.AssignedTo,
		Frequency:       input.Frequency,
		GoalLabel:       input.GoalLabel,
		StartValue:      input.StartValue,
		TargetValue:     input.TargetValue,
		AggregationType: aggType,
		Weight:          input.Weight,
		ParentTaskID:    input.ParentTaskID,
		Status:          "ACTIVE",
		CreatedBy:       util.ExtractUserID(c),
	}

	if task.Weight == 0 {
		task.Weight = 100.0
	}

	if input.StartDate != nil {
		t, _ := parseDate(*input.StartDate)
		task.StartDate = t
	}
	if input.EndDate != nil {
		t, _ := parseDate(*input.EndDate)
		task.EndDate = t
	}

	taskDao := dao.TaskDao{}
	if err := taskDao.Create(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// Create scopes
	if len(input.Scopes) > 0 {
		var scopes []model.TaskScope
		for _, s := range input.Scopes {
			scopes = append(scopes, model.TaskScope{
				TaskID:    task.ID,
				ScopeType: s.ScopeType,
				ScopeID:   s.ScopeID,
			})
		}
		taskDao.CreateScopes(scopes)
	}

	// Notify ASC directors for scoped tasks
	go notifyASCDirectorsForTask(task, input.Scopes, util.ExtractUserID(c))

	auditDao := dao.AuditDao{}
	auditDao.Write("task", task.ID, util.ExtractUserID(c), "CREATE", nil, task, c.ClientIP())

	// Reload with relations
	result, _ := taskDao.GetByID(task.ID)
	c.JSON(http.StatusCreated, result)
}

func (TaskController) Single(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	taskDao := dao.TaskDao{}
	task, err := taskDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

func (TaskController) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	taskDao := dao.TaskDao{}
	task, err := taskDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	oldData := task

	var input TaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}
	if err := validateTaskAssignee(input.OwnerType, input.OwnerID, input.AssignedTo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	task.Title = input.Title
	task.Description = input.Description
	task.OwnerType = input.OwnerType
	task.OwnerID = input.OwnerID
	task.AssignedTo = input.AssignedTo
	task.Frequency = input.Frequency
	task.GoalLabel = input.GoalLabel
	task.StartValue = input.StartValue
	task.TargetValue = input.TargetValue
	if input.AggregationType != "" {
		task.AggregationType = input.AggregationType
	}
	if input.Weight > 0 {
		task.Weight = input.Weight
	}
	task.ParentTaskID = input.ParentTaskID

	if input.StartDate != nil {
		t, _ := parseDate(*input.StartDate)
		task.StartDate = t
	}
	if input.EndDate != nil {
		t, _ := parseDate(*input.EndDate)
		task.EndDate = t
	}

	if err := taskDao.Update(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// Recalculate current_value if aggregation type or start_value changed
	startChanged := (task.StartValue == nil) != (oldData.StartValue == nil) ||
		(task.StartValue != nil && oldData.StartValue != nil && *task.StartValue != *oldData.StartValue)
	if task.AggregationType != oldData.AggregationType || startChanged {
		taskDao.RecalcCurrentValue(task.ID)
		// Also refresh performance cache
		perfDao := dao.PerformanceDao{}
		go perfDao.RefreshForTask(task.ID)
		go perfDao.RefreshForProject(task.ProjectID)
	}

	// Update scopes
	if len(input.Scopes) > 0 {
		taskDao.DeleteScopes(task.ID)
		var scopes []model.TaskScope
		for _, s := range input.Scopes {
			scopes = append(scopes, model.TaskScope{
				TaskID:    task.ID,
				ScopeType: s.ScopeType,
				ScopeID:   s.ScopeID,
			})
		}
		taskDao.CreateScopes(scopes)
	}

	// Notify up the chain
	go notifyTaskUpdateChain(task, util.ExtractUserID(c))

	auditDao := dao.AuditDao{}
	auditDao.Write("task", task.ID, util.ExtractUserID(c), "UPDATE", oldData, task, c.ClientIP())

	result, _ := taskDao.GetByID(task.ID)
	c.JSON(http.StatusOK, result)
}

func (TaskController) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	taskDao := dao.TaskDao{}

	task, err := taskDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if err := taskDao.SoftDelete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("task", uint(id), util.ExtractUserID(c), "DELETE", task, nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func notifyASCDirectorsForTask(task model.Task, scopes []ScopeInput, actorID uint) {
	geoDao := dao.GeoDao{}
	userDao := dao.UserDao{}
	notifDao := dao.NotificationDao{}

	actor, _ := userDao.GetByID(actorID)

	for _, s := range scopes {
		if s.ScopeType == "ASC" && s.ScopeID != nil {
			asc, err := geoDao.GetASCByID(*s.ScopeID)
			if err != nil || asc.DirectorID == nil {
				continue
			}
			director, err := userDao.GetByID(*asc.DirectorID)
			if err != nil {
				continue
			}
			notifDao.CreateAndEmail(director.ID, "Nova tarefa na sua ASC",
				"A tarefa '"+task.Title+"' foi atribuída à "+asc.Name, "TASK_UPDATE", "task", &task.ID)
			go util.EmailTaskUpdated(director.Email, director.Name, task.Title, actor.Name)
		}
	}
}

func notifyTaskUpdateChain(task model.Task, actorID uint) {
	userDao := dao.UserDao{}
	notifDao := dao.NotificationDao{}
	direcaoDao := dao.DirecaoDao{}
	pelouroDao := dao.PelouroDao{}

	actor, _ := userDao.GetByID(actorID)

	// Notify up: DEPARTAMENTO → DIRECAO → PELOURO → CA
	if task.OwnerType == "DEPARTAMENTO" {
		deptDao := dao.DepartamentoDao{}
		dept, err := deptDao.GetByID(task.OwnerID)
		if err == nil {
			// Notify Direção responsible
			direcao, err := direcaoDao.GetByID(dept.DirecaoID)
			if err == nil && direcao.ResponsibleID != nil {
				notifDao.CreateAndEmail(*direcao.ResponsibleID, "Tarefa actualizada",
					"A tarefa '"+task.Title+"' foi actualizada", "TASK_UPDATE", "task", &task.ID)
				resp, _ := userDao.GetByID(*direcao.ResponsibleID)
				go util.EmailTaskUpdated(resp.Email, resp.Name, task.Title, actor.Name)

				// Notify Pelouro
				pelouro, err := pelouroDao.GetByID(direcao.PelouroID)
				if err == nil && pelouro.ResponsibleID != nil {
					notifDao.CreateAndEmail(*pelouro.ResponsibleID, "Tarefa actualizada",
						"A tarefa '"+task.Title+"' foi actualizada", "TASK_UPDATE", "task", &task.ID)
					resp, _ := userDao.GetByID(*pelouro.ResponsibleID)
					go util.EmailTaskUpdated(resp.Email, resp.Name, task.Title, actor.Name)
				}
			}
		}
	} else if task.OwnerType == "DIRECAO" {
		direcao, err := direcaoDao.GetByID(task.OwnerID)
		if err == nil {
			pelouro, err := pelouroDao.GetByID(direcao.PelouroID)
			if err == nil && pelouro.ResponsibleID != nil {
				notifDao.CreateAndEmail(*pelouro.ResponsibleID, "Tarefa actualizada",
					"A tarefa '"+task.Title+"' foi actualizada", "TASK_UPDATE", "task", &task.ID)
				resp, _ := userDao.GetByID(*pelouro.ResponsibleID)
				go util.EmailTaskUpdated(resp.Email, resp.Name, task.Title, actor.Name)
			}
		}
	}

	// Always notify CA and ADMIN users
	caUsers, _ := userDao.GetByRole("CA")
	adminUsers, _ := userDao.GetByRole("ADMIN")
	allSuperUsers := append(caUsers, adminUsers...)
	for _, su := range allSuperUsers {
		if su.ID != actorID {
			notifDao.CreateAndEmail(su.ID, "Tarefa actualizada",
				"A tarefa '"+task.Title+"' foi actualizada", "TASK_UPDATE", "task", &task.ID)
			go util.EmailTaskUpdated(su.Email, su.Name, task.Title, actor.Name)
		}
	}
}
