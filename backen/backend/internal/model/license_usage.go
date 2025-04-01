package model

import (
	"time"

	"gorm.io/gorm"
)

type LicenseUsage struct {
	gorm.Model
	LicenseKey string    `json:"license_key" gorm:"index"`
	Action     string    `json:"action"` // "verify", "activate", etc.
	IPAddress  string    `json:"ip_address"`
	UserAgent  string    `json:"user_agent"`
	Timestamp  time.Time `json:"timestamp"`
}
