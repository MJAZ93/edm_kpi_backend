package controller

import (
	"net/http"
	"strconv"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type FeedbackController struct{}

// resolveTargetNames fills TargetName for a list of feedbacks by looking up the entity.
func resolveTargetNames(list []model.Feedback) {
	for i := range list {
		if list[i].TargetType == "" || list[i].TargetID == nil {
			continue
		}
		tid := *list[i].TargetID
		switch list[i].TargetType {
		case "DEPARTAMENTO":
			var d model.Departamento
			if dao.Database.Select("name").Where("id = ?", tid).First(&d).Error == nil {
				list[i].TargetName = d.Name
			}
		case "DIRECAO":
			var d model.Direcao
			if dao.Database.Select("name").Where("id = ?", tid).First(&d).Error == nil {
				list[i].TargetName = d.Name
			}
		case "PELOURO":
			var p model.Pelouro
			if dao.Database.Select("name").Where("id = ?", tid).First(&p).Error == nil {
				list[i].TargetName = p.Name
			}
		case "USER":
			var u model.User
			if dao.Database.Select("name").Where("id = ?", tid).First(&u).Error == nil {
				list[i].TargetName = u.Name
			}
		case "PROJECT":
			var p model.Project
			if dao.Database.Select("title").Where("id = ?", tid).First(&p).Error == nil {
				list[i].TargetName = p.Title
			}
		case "TASK":
			var t model.Task
			if dao.Database.Select("title").Where("id = ?", tid).First(&t).Error == nil {
				list[i].TargetName = t.Title
			}
		case "MILESTONE":
			var m model.Milestone
			if dao.Database.Select("title").Where("id = ?", tid).First(&m).Error == nil {
				list[i].TargetName = m.Title
			}
		}
	}
}

type FeedbackInput struct {
	ReceiverID uint   `json:"receiver_id" binding:"required"`
	Message    string `json:"message" binding:"required"`
	Category   string `json:"category"`
	TargetType string `json:"target_type"` // DEPARTAMENTO, DIRECAO, PELOURO, USER
	TargetID   *uint  `json:"target_id"`
}

type FeedbackReplyInput struct {
	Message string `json:"message" binding:"required"`
}

func (FeedbackController) Create(c *gin.Context) {
	var input FeedbackInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	senderID := util.ExtractUserID(c)

	if input.Category == "" {
		input.Category = "GENERAL"
	}

	feedback := model.Feedback{
		SenderID:   senderID,
		ReceiverID: input.ReceiverID,
		Message:    input.Message,
		Category:   input.Category,
		TargetType: input.TargetType,
		TargetID:   input.TargetID,
	}

	feedbackDao := dao.FeedbackDao{}
	if err := feedbackDao.Create(&feedback); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusCreated, feedback)
}

func (FeedbackController) Reply(c *gin.Context) {
	parentID, _ := strconv.Atoi(c.Param("id"))

	var input FeedbackReplyInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	feedbackDao := dao.FeedbackDao{}
	parent, err := feedbackDao.GetByID(uint(parentID))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found", "message": "feedback not found"})
		return
	}

	senderID := util.ExtractUserID(c)
	pid := uint(parentID)

	reply := model.Feedback{
		SenderID:   senderID,
		ReceiverID: parent.SenderID, // Reply goes back to the original sender
		ParentID:   &pid,
		Message:    input.Message,
		Category:   parent.Category,
	}

	if senderID == parent.SenderID {
		reply.ReceiverID = parent.ReceiverID
	}

	if err := feedbackDao.Create(&reply); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusCreated, reply)
}

func (FeedbackController) ListReceived(c *gin.Context) {
	userID := util.ExtractUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	feedbackDao := dao.FeedbackDao{}
	list, total, err := feedbackDao.ListReceived(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	resolveTargetNames(list)

	c.JSON(http.StatusOK, gin.H{"feedback": list, "total": total, "page": page, "limit": limit})
}

func (FeedbackController) ListSent(c *gin.Context) {
	userID := util.ExtractUserID(c)
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	feedbackDao := dao.FeedbackDao{}
	list, total, err := feedbackDao.ListSent(userID, page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	resolveTargetNames(list)

	c.JSON(http.StatusOK, gin.H{"feedback": list, "total": total, "page": page, "limit": limit})
}

func (FeedbackController) UnreadCount(c *gin.Context) {
	userID := util.ExtractUserID(c)

	feedbackDao := dao.FeedbackDao{}
	count, err := feedbackDao.CountUnread(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"unread_count": count})
}

func (FeedbackController) MarkAsRead(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	feedbackDao := dao.FeedbackDao{}
	feedback, err := feedbackDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	userID := util.ExtractUserID(c)
	if feedback.ReceiverID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "only the receiver can mark as read"})
		return
	}

	if err := feedbackDao.MarkAsRead(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "marked as read"})
}

func (FeedbackController) ListByTarget(c *gin.Context) {
	targetType := c.Query("target_type")
	targetID, _ := strconv.Atoi(c.Query("target_id"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if targetType == "" || targetID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": "target_type and target_id are required"})
		return
	}

	feedbackDao := dao.FeedbackDao{}
	list, total, err := feedbackDao.ListByTarget(targetType, uint(targetID), page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}
	resolveTargetNames(list)

	c.JSON(http.StatusOK, gin.H{"data": list, "total": total, "page": page, "limit": limit})
}

func (FeedbackController) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))

	feedbackDao := dao.FeedbackDao{}
	feedback, err := feedbackDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	userID := util.ExtractUserID(c)
	if feedback.SenderID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "forbidden", "message": "only the sender can delete feedback"})
		return
	}

	if err := feedbackDao.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
