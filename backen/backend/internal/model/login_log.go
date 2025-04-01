package model

import "time"

type LoginLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Status    string    `json:"status"` // success, failed
	CreatedAt time.Time `json:"created_at"`
}
