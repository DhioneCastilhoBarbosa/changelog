// internal/models/user.go
package models

type Role string

const (
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

type User struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `gorm:"size:80"`
	Email    string `gorm:"size:120;uniqueIndex"`
	Password string `gorm:"size:200"` // hash (bcrypt/argon2)
	Role     Role   `gorm:"size:10"`
}
