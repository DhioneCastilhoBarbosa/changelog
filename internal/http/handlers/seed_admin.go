// internal/http/handlers/seed_admin.go
package handlers

import (
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/models"
)

func SeedAdmin(db *gorm.DB) {
	email := os.Getenv("ADMIN_EMAIL")
	pass := os.Getenv("ADMIN_PASSWORD")
	name := os.Getenv("ADMIN_NAME")
	if email == "" || pass == "" {
		return // sem variáveis -> não faz seed
	}
	if name == "" {
		name = "Admin"
	}

	var existing models.User
	if err := db.Where("email = ?", email).First(&existing).Error; err == nil {
		return
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(pass), 12)
	u := models.User{
		Name:     name,
		Email:    email,
		Password: string(hash),
		Role:     models.RoleAdmin,
	}
	if err := db.Create(&u).Error; err != nil {
		log.Printf("seed admin erro: %v", err)
	} else {
		log.Printf("seed admin criado: %s", email)
	}
}
