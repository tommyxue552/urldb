package entity

import (
	"time"

	"gorm.io/gorm"
)

// OwnedShare is a share link created by an account controlled by this service.
// Resource.SaveURL remains untouched for legacy compatibility.
type OwnedShare struct {
	ID                  uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	ResourceID          uint           `json:"resource_id" gorm:"not null;uniqueIndex:idx_owned_share_target,priority:1"`
	PanID               uint           `json:"pan_id" gorm:"not null;uniqueIndex:idx_owned_share_target,priority:2"`
	CkID                uint           `json:"ck_id" gorm:"not null;uniqueIndex:idx_owned_share_target,priority:3"`
	URL                 string         `json:"url" gorm:"size:500;not null"`
	Fid                 string         `json:"fid" gorm:"size:128;not null"`
	Status              string         `json:"status" gorm:"size:20;not null;default:active;index"`
	Channel             string         `json:"channel" gorm:"size:50;not null;default:system"`
	ExpiresAt           *time.Time     `json:"expires_at" gorm:"index"`
	LastCheckedAt       *time.Time `json:"last_checked_at" gorm:"index"`
	LastCheckStatus     string     `json:"last_check_status" gorm:"size:20"`
	LastCheckMethod     string     `json:"last_check_method" gorm:"size:30"`
	LastCheckFailReason string     `json:"last_check_fail_reason" gorm:"size:500"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
	DeletedAt           gorm.DeletedAt `json:"-" gorm:"index"`
}

func (OwnedShare) TableName() string { return "owned_shares" }
