package model

import "time"

type OperationLog struct {
	ID        uint      `json:"id" gorm:"primaryKey"`
	UserID    uint      `json:"user_id"`
	Action    string    `json:"action"`
	Target    string    `json:"target"`
	TargetID  string    `json:"target_id"`
	Details   string    `json:"details"`
	CreatedAt time.Time `json:"created_at"`
}
