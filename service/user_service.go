package service

import (
	"kpi-backend/controller"
	"kpi-backend/middleware"

	"github.com/gin-gonic/gin"
)

type UserService struct {
	Route      string
	Controller controller.UserController
}

// Me godoc
// @Summary Current user profile
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.UserResponse
// @Router /private/users/me [get]
func (s UserService) Me(r *gin.RouterGroup, route string) {
	r.GET("/"+s.Route+"/"+route, s.Controller.Me)
}

// List godoc
// @Summary List users
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param page query int false "page"
// @Param limit query int false "limit"
// @Success 200 {object} util.PaginatedResponse
// @Router /private/users [get]
func (s UserService) List(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route, middleware.RoleMiddleware("CA"), s.Controller.List)
}

// Create godoc
// @Summary Create user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body controller.CreateUserInput true "user data"
// @Success 201 {object} model.UserResponse
// @Router /private/users [post]
func (s UserService) Create(r *gin.RouterGroup, _ string) {
	r.POST("/"+s.Route, middleware.RoleMiddleware("CA"), s.Controller.Create)
}

// Single godoc
// @Summary Get user by ID
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param id path int true "user ID"
// @Success 200 {object} model.UserResponse
// @Router /private/users/{id} [get]
func (s UserService) Single(r *gin.RouterGroup, _ string) {
	r.GET("/"+s.Route+"/:id", middleware.RoleMiddleware("CA"), s.Controller.Single)
}

// Update godoc
// @Summary Update user
// @Tags Users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "user ID"
// @Param body body controller.UpdateUserInput true "fields to update"
// @Success 200 {object} model.UserResponse
// @Router /private/users/{id} [put]
func (s UserService) Update(r *gin.RouterGroup, _ string) {
	r.PUT("/"+s.Route+"/:id", middleware.RoleMiddleware("CA"), s.Controller.Update)
}

// Delete godoc
// @Summary Delete user
// @Tags Users
// @Produce json
// @Security BearerAuth
// @Param id path int true "user ID"
// @Success 200 {object} map[string]string
// @Router /private/users/{id} [delete]
func (s UserService) Delete(r *gin.RouterGroup, _ string) {
	r.DELETE("/"+s.Route+"/:id", middleware.RoleMiddleware("CA"), s.Controller.Delete)
}
