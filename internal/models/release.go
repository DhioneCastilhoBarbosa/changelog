// internal/models/release.go
package models

import "time"

// 1. Tipo e constantes
type FirmwareStatus string

const (
	FirmwareStatusRevisao       FirmwareStatus = "revisao"
	FirmwareStatusProducao      FirmwareStatus = "producao"
	FirmwareStatusDescontinuado FirmwareStatus = "descontinuado"
)

func (s FirmwareStatus) Valid() bool {
	switch s {
	case FirmwareStatusRevisao, FirmwareStatusProducao, FirmwareStatusDescontinuado:
		return true
	default:
		return false
	}
}

// 2. Campo novo no model Release
type Release struct {
	ID              uint   `gorm:"primaryKey"`
	Version         string `gorm:"uniqueIndex;size:32"`
	PreviousVersion string `gorm:"size:32"`
	OTA             bool
	OTAObs          string `gorm:"size:255"`
	ReleaseDate     time.Time `json:"releaseDate" gorm:"index"`
	ImportantNote   string           `gorm:"type:text"`
	ProductCategory string           `json:"productCategory" gorm:"size:60;index"`
	ProductName     string           `json:"productName"     gorm:"size:120;index"`
	Status          FirmwareStatus   `json:"status" gorm:"type:varchar(20);default:producao;index"`
	CreatedByUserID uint             `json:"-"`
	CreatedBy       *User            `json:"createdBy,omitempty" gorm:"foreignKey:CreatedByUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Modules         []ReleaseModule  `gorm:"constraint:OnDelete:CASCADE"`
	Entries         []ChangelogEntry `gorm:"constraint:OnDelete:CASCADE"`
	Links          []FirmwareLink    `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt       time.Time         `json:"createdAt" gorm:"autoCreateTime"`
	UpdatedAt       time.Time         `json:"updatedAt" gorm:"autoUpdateTime"`
}

type FirmwareLink struct {
	ID        uint      `gorm:"primaryKey"`
	ReleaseID uint      `gorm:"index"`
	Module    string    `gorm:"size:120;not null"`
	Description string  `gorm:"size:255;not null"`
	URL       string    `gorm:"size:2048;not null"`
	CreatedAt time.Time
	UpdatedAt time.Time
}


