package controller

import (
	"net/http"
	"strconv"

	"kpi-backend/dao"
	"kpi-backend/model"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type UserController struct{}

func (UserController) Me(c *gin.Context) {
	userID := util.ExtractUserID(c)
	userDao := dao.UserDao{}
	user, err := userDao.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, user.ToResponse())
}

func (UserController) List(c *gin.Context) {
	params := util.ParsePagination(c)
	userDao := dao.UserDao{}
	users, total, err := userDao.List(params.Page, params.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	var resp []model.UserResponse
	for _, u := range users {
		resp = append(resp, u.ToResponse())
	}
	c.JSON(http.StatusOK, util.NewPaginatedResponse(resp, total, params))
}

type CreateUserInput struct {
	Name     string `json:"name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Role     string `json:"role" binding:"required,oneof=CA PELOURO DIRECAO DEPARTAMENTO"`
}

func (UserController) Create(c *gin.Context) {
	var input CreateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	user := model.User{
		Name:     input.Name,
		Email:    input.Email,
		Password: input.Password,
		Role:     input.Role,
		Active:   true,
	}

	userDao := dao.UserDao{}
	if err := userDao.Create(&user); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "conflict", "message": "email already in use"})
		return
	}

	go util.EmailWelcome(user.Email, user.Name, input.Password)

	auditDao := dao.AuditDao{}
	auditDao.Write("user", user.ID, util.ExtractUserID(c), "CREATE", nil, user.ToResponse(), c.ClientIP())

	c.JSON(http.StatusCreated, user.ToResponse())
}

func (UserController) Single(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userDao := dao.UserDao{}
	user, err := userDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}
	c.JSON(http.StatusOK, user.ToResponse())
}

type UpdateUserInput struct {
	Name   *string `json:"name"`
	Email  *string `json:"email"`
	Role   *string `json:"role"`
	Active *bool   `json:"active"`
}

func (UserController) Update(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userDao := dao.UserDao{}
	user, err := userDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	oldData := user.ToResponse()

	var input UpdateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	if input.Name != nil {
		user.Name = *input.Name
	}
	if input.Email != nil {
		user.Email = *input.Email
	}
	if input.Role != nil {
		user.Role = *input.Role
	}
	if input.Active != nil {
		user.Active = *input.Active
	}

	if err := userDao.Update(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("user", user.ID, util.ExtractUserID(c), "UPDATE", oldData, user.ToResponse(), c.ClientIP())

	c.JSON(http.StatusOK, user.ToResponse())
}

func (UserController) Delete(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userDao := dao.UserDao{}

	user, err := userDao.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	if err := userDao.SoftDelete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	auditDao := dao.AuditDao{}
	auditDao.Write("user", uint(id), util.ExtractUserID(c), "DELETE", user.ToResponse(), nil, c.ClientIP())

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}
