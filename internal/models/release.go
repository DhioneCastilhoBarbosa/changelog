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
	ReleaseDate     time.Time
	ImportantNote   string `gorm:"type:text"`

	// NOVO
	Status FirmwareStatus `json:"status" gorm:"type:varchar(20);default:producao;index"`

	CreatedByUserID uint             `json:"-"`
	CreatedBy       *User            `json:"createdBy,omitempty" gorm:"foreignKey:CreatedByUserID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT"`
	Modules         []ReleaseModule  `gorm:"constraint:OnDelete:CASCADE"`
	Entries         []ChangelogEntry `gorm:"constraint:OnDelete:CASCADE"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
