package service

import (
	"errors"
	"time"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/models"
	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/repository"
)

type ReleaseService struct {
	repo repository.ReleaseRepository
}

func NewReleaseService(repo repository.ReleaseRepository) *ReleaseService {
	return &ReleaseService{repo: repo}
}

type ReleaseQuery struct {
	Q        string
	Version  string
	DateFrom *time.Time
	DateTo   *time.Time
}

func (s *ReleaseService) Create(in *models.Release) (*models.Release, error) {
	if in.Version == "" {
		return nil, errors.New("version é obrigatória")
	}
	if err := s.repo.Create(in); err != nil {
		return nil, err
	}
	return s.repo.GetByID(in.ID)
}

func (s *ReleaseService) Get(id uint) (*models.Release, error) {
	return s.repo.GetByID(id)
}

func (s *ReleaseService) List(q ReleaseQuery) ([]models.Release, error) {
	f := repository.ReleaseFilter{
		Q:        q.Q,
		Version:  q.Version,
		DateFrom: q.DateFrom,
		DateTo:   q.DateTo,
	}
	return s.repo.List(f)
}

func (s *ReleaseService) UpdateFull(id uint, base models.Release, modules []models.ReleaseModule, entries []models.ChangelogEntry, links []models.FirmwareLink,) (*models.Release, error) {
	// Atualiza campos simples
	base.ID = id
	if err := s.repo.UpdateBaseFields(&base); err != nil {
		return nil, err
	}
	// Substitui relações
	return s.repo.ReplaceRelations(id, modules, entries)
}

func (s *ReleaseService) Delete(id uint) error {
	return s.repo.Delete(id)
}
