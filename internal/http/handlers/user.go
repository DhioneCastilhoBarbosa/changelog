// internal/http/handlers/user.go
package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/models"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/service"
)

type UserHandler struct {
	Svc *service.UserService
}

/* ===== DTOs ===== */

type CreateUserDTO struct {
	Name     string `json:"name" binding:"required,min=2"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6,max=72"`
}

type UserPublicCreated struct {
	ID    uint        `json:"id"`
	Name  string      `json:"name"`
	Email string      `json:"email"`
	Role  models.Role `json:"role"`
}

/* ===== Mappers ===== */

func toUserPublic(u *models.User) UserPublicCreated {
	return UserPublicCreated{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role}
}

/* ===== Handlers ===== */

// Rota pública: cria usuário sempre como viewer.
func (h UserHandler) Create(c *gin.Context) {
	var in CreateUserDTO
	if err := c.ShouldBindJSON(&in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	name := strings.TrimSpace(in.Name)
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if name == "" || email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "nome e email são obrigatórios"})
		return
	}

	// hash da senha
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "hash error"})
		return
	}

	u := &models.User{
		Name:     name,
		Email:    email,
		Password: string(hash),
		Role:     models.RoleViewer, // força viewer
	}

	if err := h.Svc.Create(u); err != nil {
		if service.IsUniqueViolation(err) {
			c.JSON(http.StatusConflict, gin.H{"error": "email já cadastrado"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toUserPublic(u))
}
