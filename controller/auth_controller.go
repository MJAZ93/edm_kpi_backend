package controller

import (
	"net/http"

	"kpi-backend/dao"
	"kpi-backend/util"

	"github.com/gin-gonic/gin"
)

type AuthController struct{}

type LoginInput struct {
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginOutput struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
}

func (AuthController) Login(c *gin.Context) {
	var input LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	userDao := dao.UserDao{}
	user, err := userDao.GetByEmailAndPassword(input.Email, input.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid_credentials"})
		return
	}

	token, err := util.GenerateJWT(user.ID, gin.H{
		"email": user.Email,
		"role":  user.Role,
		"name":  user.Name,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "token_error"})
		return
	}

	userDao.UpdateLastLogin(user.ID)

	c.JSON(http.StatusOK, LoginOutput{
		Token: token,
		User:  user.ToResponse(),
	})
}

type ForgotPasswordInput struct {
	Email string `json:"email" binding:"required"`
}

func (AuthController) ForgotPassword(c *gin.Context) {
	var input ForgotPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	userDao := dao.UserDao{}
	user, err := userDao.GetByEmail(input.Email)
	if err != nil {
		// Don't reveal whether email exists
		c.JSON(http.StatusOK, gin.H{"message": "reset email sent"})
		return
	}

	token, err := userDao.SetPasswordResetToken(user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	go util.EmailPasswordReset(user.Email, user.Name, token)

	c.JSON(http.StatusOK, gin.H{"message": "reset email sent"})
}

type ResetPasswordInput struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
}

func (AuthController) ResetPassword(c *gin.Context) {
	var input ResetPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad_request", "message": err.Error()})
		return
	}

	userDao := dao.UserDao{}
	user, err := userDao.GetByResetToken(input.Token)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_or_expired_token"})
		return
	}

	user.Password = input.Password
	if err := userDao.Update(&user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal_error"})
		return
	}

	userDao.ClearResetToken(user.ID)

	c.JSON(http.StatusOK, gin.H{"message": "password updated"})
}
