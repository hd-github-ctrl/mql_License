package model

import "time"

type LicenseInput struct {
	Version         string    `json:"version"`
	ValidUntil      time.Time `json:"valid_until"`
	Permissions     string    `json:"permissions"`
	UserId          string    `json:"userid"`
	ProductId       string    `json:"productid"`
	LastActivatedAt time.Time `json:"last_activated_at"`
}
