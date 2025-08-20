package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/models"
)

type ReleaseFilter struct {
	Q        string
	Version  string
	DateFrom *time.Time
	DateTo   *time.Time
}

type ReleaseRepository interface {
	Create(r *models.Release) error
	GetByID(id uint) (*models.Release, error)
	List(f ReleaseFilter) ([]models.Release, error)
	ReplaceRelations(id uint, modules []models.ReleaseModule, entries []models.ChangelogEntry) (*models.Release, error)
	UpdateBaseFields(r *models.Release) error
	Delete(id uint) error
}

type releaseRepository struct {
	db *gorm.DB
}

func NewReleaseRepository(db *gorm.DB) ReleaseRepository {
	return &releaseRepository{db: db}
}

func (r *releaseRepository) Create(rel *models.Release) error {
	return r.db.Create(rel).Error
}

func (r *releaseRepository) GetByID(id uint) (*models.Release, error) {
	var out models.Release
	err := r.db.
		Preload("Modules").
		Preload("Entries", func(tx *gorm.DB) *gorm.DB { return tx.Order("item_order ASC") }).
		First(&out, id).Error
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *releaseRepository) List(f ReleaseFilter) ([]models.Release, error) {
	tx := r.db.Model(&models.Release{}).
		Preload("Modules").
		Preload("Entries", func(tx *gorm.DB) *gorm.DB { return tx.Order("item_order ASC") }).
		Order("release_date DESC")

	if f.Version != "" {
		tx = tx.Where("version = ?", f.Version)
	}
	if f.Q != "" {
		like := "%" + f.Q + "%"
		tx = tx.Where(
			r.db.Where("version ILIKE ?", like).
				Or("previous_version ILIKE ?", like).
				Or("ota_obs ILIKE ?", like).
				Or("important_note ILIKE ?", like),
		)
	}
	if f.DateFrom != nil {
		tx = tx.Where("release_date >= ?", *f.DateFrom)
	}
	if f.DateTo != nil {
		tx = tx.Where("release_date <= ?", *f.DateTo)
	}

	var list []models.Release
	if err := tx.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *releaseRepository) UpdateBaseFields(rel *models.Release) error {
	return r.db.Save(rel).Error
}

// Estratégia “replace-all” para Modules e Entries
func (r *releaseRepository) ReplaceRelations(id uint, modules []models.ReleaseModule, entries []models.ChangelogEntry) (*models.Release, error) {
	tx := r.db.Begin()

	if err := tx.Where("release_id = ?", id).Delete(&models.ReleaseModule{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	if err := tx.Where("release_id = ?", id).Delete(&models.ChangelogEntry{}).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	for i := range modules {
		modules[i].ReleaseID = id
	}
	for i := range entries {
		entries[i].ReleaseID = id
	}

	if len(modules) > 0 {
		if err := tx.Create(&modules).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}
	if len(entries) > 0 {
		if err := tx.Create(&entries).Error; err != nil {
			tx.Rollback()
			return nil, err
		}
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}
	return r.GetByID(id)
}

func (r *releaseRepository) Delete(id uint) error {
	return r.db.Delete(&models.Release{}, id).Error
}
