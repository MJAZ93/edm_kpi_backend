package service

import (
	"kpi-backend/controller"

	"github.com/gin-gonic/gin"
)

type AuthService struct {
	Route      string
	Controller controller.AuthController
}

// Login godoc
// @Summary User login
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body controller.LoginInput true "credentials"
// @Success 200 {object} controller.LoginOutput
// @Router /public/auth/login [post]
func (s AuthService) Login(r *gin.RouterGroup, route string) {
	r.POST("/"+s.Route+"/"+route, s.Controller.Login)
}

// ForgotPassword godoc
// @Summary Request password reset
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body controller.ForgotPasswordInput true "email"
// @Success 200 {object} map[string]string
// @Router /public/auth/forgot-password [post]
func (s AuthService) ForgotPassword(r *gin.RouterGroup, route string) {
	r.POST("/"+s.Route+"/"+route, s.Controller.ForgotPassword)
}

// ResetPassword godoc
// @Summary Reset password with token
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body controller.ResetPasswordInput true "token + new password"
// @Success 200 {object} map[string]string
// @Router /public/auth/reset-password [post]
func (s AuthService) ResetPassword(r *gin.RouterGroup, route string) {
	r.POST("/"+s.Route+"/"+route, s.Controller.ResetPassword)
}
