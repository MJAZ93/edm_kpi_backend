package controller

import (
	"net/http"
	"strconv"

	"kpi-backend/dao"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type NotificationController struct{}

func (NotificationController) List(c *gin.Context) {
	userID := util.ExtractUserID(c)
	params := util.ParsePagination(c)

	var isRead *bool
	if v := c.Query("is_read"); v != "" {
		b := v == "true"
		isRead = &b
	}

	notifDao := dao.NotificationDao{}
	list, total, err := notifDao.ListByUser(userID, params.Page, params.Limit, isRead)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	unread := notifDao.UnreadCount(userID)

	c.JSON(http.StatusOK, gin.H{
		"data":         list,
		"total":        total,
		"page":         params.Page,
		"limit":        params.Limit,
		"unread_count": unread,
	})
}

func (NotificationController) MarkRead(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userID := util.ExtractUserID(c)

	notifDao := dao.NotificationDao{}
	if err := notifDao.MarkRead(uint(id), userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "marked as read"})
}

func (NotificationController) MarkAllRead(c *gin.Context) {
	userID := util.ExtractUserID(c)

	notifDao := dao.NotificationDao{}
	if err := notifDao.MarkAllRead(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all marked as read"})
}
