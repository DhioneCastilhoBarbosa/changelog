package models

import "time"

type Approval struct {
	ID uint `gorm:"primaryKey"`

	Establishment string 	`gorm:"size:255;not null"`
	Date 				time.Time `gorm:"not null"`
	ProductName 	string 	`gorm:"size:255;not null"`
	Category 		string 	`gorm:"size:255;not null"`
	Description 	string 	`gorm:"type:text;not null"`

	FileURL 		string 	`gorm:"size:1024;not null"`

	CreatedByUserID uint `gorm:"not null"`
	CreatedBy  *User `gorm:"foreignKey:CreatedByUserID"`

	CreatedAt time.Time
	UpdatedAt time.Time
}