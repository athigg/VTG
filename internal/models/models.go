package models

import "time"

const (
	RoleUser  = "user"
	RoleAdmin = "admin"
)

type User struct {
	ID           uint   `gorm:"primaryKey"`
	Name         string `gorm:"size:100;not null"`
	Email        string `gorm:"size:255;uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	Role         string `gorm:"size:20;not null;default:user"`
	Active       bool   `gorm:"not null;default:true"`
	Bio          string `gorm:"size:500"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Products     []Product `gorm:"constraint:OnDelete:CASCADE"`
}

type Product struct {
	ID          uint   `gorm:"primaryKey"`
	UserID      uint   `gorm:"not null;index"`
	User        User   `gorm:"constraint:OnDelete:CASCADE"`
	Title       string `gorm:"size:120;not null"`
	Description string `gorm:"size:2000;not null"`
	PriceCents  int64  `gorm:"not null"`
	Category    string `gorm:"size:50;not null"`
	Condition   string `gorm:"size:30;not null"`
	ImagePath   string `gorm:"size:255"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
