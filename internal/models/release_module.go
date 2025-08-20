// internal/models/release_module.go
package models

type ReleaseModule struct {
	ID        uint   `gorm:"primaryKey"`
	ReleaseID uint   `gorm:"index"`
	Module    string `gorm:"size:32"` // ex: "PCB A7", "PCB M4"
	Version   string `gorm:"size:32"` // ex: "1.3033.0"
	Updated   bool
}
