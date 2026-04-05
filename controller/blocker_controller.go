package controller

import (
	"net/http"
	"strconv"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type BlockerController struct{}

func (BlockerController) List(c *gin.Context) {
	params := util.ParsePagination(c)
	filters := make(map[string]interface{})

	if et := c.Query("entity_type"); et != "" {
		filters["entity_type"] = et
	}
	if eid := c.Query("entity_id"); eid != "" {
		id, _ := strconv.Atoi(eid)
		filters["entity_id"] = id
	}
	if st := c.Query("status"); st != "" {
		filters["status"] = st
	}

	blockerDao := dao.BlockerDao{}
	list, total, err := blockerDao.List(params.Page, params.Limit, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	c.JSON(http.StatusOK, util.NewPaginatedResponse(list, total, params))
}

type BlockerInput struct {
	EntityType  string `json:"entity_type" binding:"required,oneof=TASK MILESTONE"`
	EntityID    uint   `json:"entity_id" binding:"required"`
	BlockerType string `json:"blocker_type" binding:"required,oneof=LOGISTIC FINANCIAL TECHNICAL LEGAL"`
	Description string `json:"description" binding:"required"`
	SLADays     int    `json:"sla_days"`
}

func (BlockerController) Create(c *gin.Context) {
	var input BlockerInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	slaDays := input.SLADays
	if slaDays <= 0 {
		slaDays = 3
	}

	blocker := model.Blocker{
		EntityType:  input.EntityType,
		EntityID:    input.EntityID,
		BlockerType: input.BlockerType,
		Description: input.Description,
		ReportedBy:  util.ExtractUserID(c),
		Status:      "PENDING",
		SLADays:     slaDays,
	}

	blockerDao := dao.BlockerDao{}
	if err := blockerDao.Create(&blocker); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// Notify superior
	go notifyBlockerCreated(blocker, util.ExtractUserID(c))

	auditDao := dao.AuditDao{}
	auditDao.Write("blocker", blocker.ID, util.ExtractUserID(c), "CREATE", nil, blocker, c.ClientIP())

	c.JSON(http.StatusCreated, blocker)
}

func (BlockerController) Single(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	blockerDao := dao.BlockerDao{}
	blocker, err := blockerDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, blocker)
}

func (BlockerController) Approve(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	blockerDao := dao.BlockerDao{}

	blocker, err := blockerDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if blocker.Status != "PENDING" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "blocker already resolved"})
		return
	}

	userID := util.ExtractUserID(c)
	if err := blockerDao.Approve(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	// If milestone, set milestone status to BLOCKED
	if blocker.EntityType == "MILESTONE" {
		milestoneDao := dao.MilestoneDao{}
		ms, err := milestoneDao.GetByID(blocker.EntityID)
		if err == nil {
			ms.Status = "BLOCKED"
			milestoneDao.Update(&ms)
			taskDao := dao.TaskDao{}
			taskDao.RecalcCurrentValue(ms.TaskID)
		}
	}

	// Notify reporter
	go notifyBlockerResolved(blocker, "APROVADO")

	auditDao := dao.AuditDao{}
	auditDao.Write("blocker", uint(id), userID, "UPDATE", blocker, gin.H{"status": "APPROVED"}, c.ClientIP())

	updated, _ := blockerDao.GetByID(uint(id))
	c.JSON(http.StatusOK, updated)
}

type RejectInput struct {
	RejectionReason string `json:"rejection_reason" binding:"required"`
}

func (BlockerController) Reject(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	blockerDao := dao.BlockerDao{}

	blocker, err := blockerDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if blocker.Status != "PENDING" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "blocker already resolved"})
		return
	}

	var input RejectInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	userID := util.ExtractUserID(c)
	if err := blockerDao.Reject(uint(id), userID, input.RejectionReason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	go notifyBlockerResolved(blocker, "REJEITADO")

	auditDao := dao.AuditDao{}
	auditDao.Write("blocker", uint(id), userID, "UPDATE", blocker, gin.H{"status": "REJECTED", "reason": input.RejectionReason}, c.ClientIP())

	updated, _ := blockerDao.GetByID(uint(id))
	c.JSON(http.StatusOK, updated)
}

func notifyBlockerCreated(blocker model.Blocker, actorID uint) {
	userDao := dao.UserDao{}
	notifDao := dao.NotificationDao{}

	// Notify superior users (CA always, and relevant pelouro/direcao)
	caUsers, _ := userDao.GetByRole("CA")
	for _, ca := range caUsers {
		if ca.ID != actorID {
			notifDao.CreateAndEmail(ca.ID, "Impedimento reportado",
				"Novo impedimento ("+blocker.BlockerType+"): "+blocker.Description,
				"BLOCKER_CREATED", "blocker", &blocker.ID)
		}
	}

	// Get task title for email
	taskTitle := "tarefa"
	if blocker.EntityType == "TASK" {
		taskDao := dao.TaskDao{}
		t, err := taskDao.GetByID(blocker.EntityID)
		if err == nil {
			taskTitle = t.Title
		}
	} else if blocker.EntityType == "MILESTONE" {
		milestoneDao := dao.MilestoneDao{}
		m, err := milestoneDao.GetByID(blocker.EntityID)
		if err == nil {
			taskTitle = m.Title
		}
	}

	reporter, _ := userDao.GetByID(actorID)
	for _, ca := range caUsers {
		if ca.ID != actorID {
			go util.EmailBlockerCreated(ca.Email, ca.Name, blocker.Description, taskTitle)
		}
	}
	_ = reporter
}

func notifyBlockerResolved(blocker model.Blocker, status string) {
	userDao := dao.UserDao{}
	notifDao := dao.NotificationDao{}

	reporter, err := userDao.GetByID(blocker.ReportedBy)
	if err != nil {
		return
	}

	notifDao.CreateAndEmail(reporter.ID, "Impedimento "+status,
		"O impedimento que reportou foi "+status, "BLOCKER_RESOLVED", "blocker", &blocker.ID)

	taskTitle := "tarefa"
	if blocker.EntityType == "TASK" {
		taskDao := dao.TaskDao{}
		t, err := taskDao.GetByID(blocker.EntityID)
		if err == nil {
			taskTitle = t.Title
		}
	}

	go util.EmailBlockerResolved(reporter.Email, reporter.Name, taskTitle, status)
}
