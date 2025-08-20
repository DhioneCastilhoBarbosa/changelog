// internal/models/changelog_entry.go
package models

type EntryClassification string

const (
	ClassificationNovo       EntryClassification = "Novo"
	ClassificationOtimizacao EntryClassification = "Otimização"
	ClassificationCorrecao   EntryClassification = "Correção"
	ClassificationSeguranca  EntryClassification = "Segurança"
)

type ChangelogEntry struct {
	ID             uint                `gorm:"primaryKey"`
	ReleaseID      uint                `gorm:"index"`
	ItemOrder      int                 // 1,2,3...
	Classification EntryClassification `gorm:"size:16"`
	Observation    string              `gorm:"type:text"`
}
