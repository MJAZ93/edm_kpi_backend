package service

import (
	"kpi-backend/controller"

	"github.com/gin-gonic/gin"
)

type FeedbackService struct {
	Route      string
	Controller controller.FeedbackController
}

func (s FeedbackService) Create(r *gin.RouterGroup, _ string) {
	r.POST("/"+s.Route, s.Controller.Create)
}

func (s FeedbackService) Reply(r *gin.RouterGroup, _ string) {
	r.POST("/"+s.Route+"/:id/reply", s.Controller.Reply)
}

func (s FeedbackService) ListReceived(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/received", s.Controller.ListReceived)
}

func (s FeedbackService) ListSent(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/sent", s.Controller.ListSent)
}

func (s FeedbackService) UnreadCount(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/unread-count", s.Controller.UnreadCount)
}

func (s FeedbackService) MarkAsRead(r *gin.RouterGroup, _ string) {
	r.PUT("/"+s.Route+"/:id/read", s.Controller.MarkAsRead)
}

func (s FeedbackService) ListByTarget(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/by-target", s.Controller.ListByTarget)
}

func (s FeedbackService) Delete(r *gin.RouterGroup, _ string) {
	r.DELETE("/"+s.Route+"/:id", s.Controller.Delete)
}
