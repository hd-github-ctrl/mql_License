package model

import (
	"time"
)

type User struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"unique;not null"`
	Password  string    `json:"-" gorm:"not null"`
	Email     string    `json:"email" gorm:"unique;not null"`
	Role      string    `json:"role" gorm:"default:'user'"`
	Status    string    `json:"status" gorm:"default:'active'"`
	Company   string    `json:"company"`
	CreatedAt time.Time `json:"createdat"`
	UpdatedAt time.Time `json:"updatedat"`
	LastLogin time.Time `json:"lastlogin"`
}
