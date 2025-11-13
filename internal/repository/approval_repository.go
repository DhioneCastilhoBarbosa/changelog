package repository

import (
	"time"

	"github.com/DhioneCastilhoBarbosa/firmware-changelog/internal/models"
	"gorm.io/gorm"
)

type ApprovalQuery struct {
	Q             string
	Establishment string
	ProductName   string
	Category      string
	DateFrom      *time.Time
	DateTo        *time.Time
}

type ApprovalRepository interface {
	Create(a *models.Approval) (*models.Approval, error)
	GetByID(id uint) (*models.Approval, error)
	List(q ApprovalQuery) ([]models.Approval, error)
	Delete(id uint) error
}

type approvalRepository struct {
	db *gorm.DB
}

func NewApprovalRepository(db *gorm.DB) ApprovalRepository {
	return &approvalRepository{db: db}
}

func (r *approvalRepository) Create(a *models.Approval) (*models.Approval, error) {
	if err := r.db.Create(a).Error; err != nil {
		return nil, err
	}
	return a, nil
}

func (r *approvalRepository) GetByID(id uint) (*models.Approval, error) {
	var a models.Approval
	if err := r.db.Preload("CreatedBy").First(&a, id).Error; err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *approvalRepository) List(q ApprovalQuery) ([]models.Approval, error) {
	tx := r.db.Model(&models.Approval{}).Preload("CreatedBy")

	if q.Establishment != "" {
		tx = tx.Where("establishment ILIKE ?", "%"+q.Establishment+"%")
	}
	if q.ProductName != "" {
		tx = tx.Where("product_name ILIKE ?", "%"+q.ProductName+"%")
	}
	if q.Category != "" {
		tx = tx.Where("category ILIKE ?", "%"+q.Category+"%")
	}
	if q.Q != "" {
		like := "%" + q.Q + "%"
		tx = tx.Where(
			r.db.Where("establishment ILIKE ?", like).
				Or("product_name ILIKE ?", like).
				Or("category ILIKE ?", like).
				Or("description ILIKE ?", like),
		)
	}
	if q.DateFrom != nil {
		tx = tx.Where("date >= ?", *q.DateFrom)
	}
	if q.DateTo != nil {
		tx = tx.Where("date <= ?", *q.DateTo)
	}

	tx = tx.Order("date DESC, id DESC")

	var list []models.Approval
	if err := tx.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *approvalRepository) Delete(id uint) error {
	return r.db.Delete(&models.Approval{}, id).Error
}
