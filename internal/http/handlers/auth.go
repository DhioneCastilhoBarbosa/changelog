package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/service"
)

type AuthHandler struct {
	Svc *service.AuthService
}

type loginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func (h AuthHandler) Login(c *gin.Context) {
	var in loginRequest
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	token, user, err := h.Svc.Login(in.Email, in.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "credenciais inválidas"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}
