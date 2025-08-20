package repository

import (
	"gorm.io/gorm"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/models"
)

type UserRepository interface {
	FindByEmail(email string) (*models.User, error)
	FindByID(id uint) (*models.User, error)
	Create(u *models.User) error
	UpdateRole(id uint, role models.Role) error
}

type userRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) FindByEmail(email string) (*models.User, error) {
	var u models.User
	// case-insensitive (funciona em Postgres e SQLite)
	if err := r.db.Where("LOWER(email) = LOWER(?)", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) FindByID(id uint) (*models.User, error) {
	var u models.User
	if err := r.db.First(&u, id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *userRepository) Create(u *models.User) error {
	// segurança extra: todo novo usuário nasce viewer
	if u.Role == "" {
		u.Role = models.RoleViewer
	}
	return r.db.Create(u).Error
}

func (r *userRepository) UpdateRole(id uint, role models.Role) error {
	return r.db.Model(&models.User{}).
		Where("id = ?", id).
		Update("role", role).Error
}
