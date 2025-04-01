package model

import (
	"time"

	"gorm.io/gorm"
)

type License struct {
	gorm.Model
	Key             string    `json:"key" gorm:"primaryKey"`
	Status          string    `json:"status" gorm:"not null"`
	ValidUntil      time.Time `json:"valid_until"`
	IssuedTo        uint      `json:"issued_to"`
	Version         string    `json:"version"`
	Permissions     string    `json:"permissions"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	UserId          string    `json:"userid"`
	ProductId       string    `json:"productid"`
	LastActivatedAt time.Time `json:"last_activated_at"`
}
