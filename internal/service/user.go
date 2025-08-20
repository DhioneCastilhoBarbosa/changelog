// internal/service/user.go
package service

import (
	"strings"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/models"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/repository"
)

type UserService struct {
	repo repository.UserRepository
}

func NewUserService(r repository.UserRepository) *UserService {
	return &UserService{repo: r}
}

func (s *UserService) Create(u *models.User) error {
	// garante viewer mesmo se alguém tentar forçar
	u.Role = models.RoleViewer
	return s.repo.Create(u)
}

// helper para conflito de unique (Postgres 23505)
func IsUniqueViolation(err error) bool {
	type causer interface {
		Unwrap() error
		Error() string
	}
	if err == nil {
		return false
	}
	// cheque código do driver se usar pgx ou lib/pq
	return strings.Contains(err.Error(), "23505")
}
