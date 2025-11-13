package service

import (
	"time"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/models"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/repository"
)

type ApprovalService struct {
	Repo repository.ApprovalRepository
}

func NewApprovalService(repo repository.ApprovalRepository) *ApprovalService {
	return &ApprovalService{Repo: repo}
}

type ApprovalQuery = repository.ApprovalQuery

func (s *ApprovalService) Create(a *models.Approval) (*models.Approval, error) {
	// Aqui dá pra colocar validações de negócio, se precisar
	if a.Date.IsZero() {
		a.Date = time.Now()
	}
	return s.Repo.Create(a)
}

func (s *ApprovalService) Get(id uint) (*models.Approval, error) {
	return s.Repo.GetByID(id)
}

func (s *ApprovalService) List(q ApprovalQuery) ([]models.Approval, error) {
	return s.Repo.List(q)
}

func (s *ApprovalService) Delete(id uint) error {
	return s.Repo.Delete(id)
}
