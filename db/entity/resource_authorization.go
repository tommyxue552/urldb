package entity

import (
	"time"

	"gorm.io/gorm"
)

// ResourceAuthorization records the evidence permitting an owned share.
// Evidence itself is never exposed from public resource endpoints.
type ResourceAuthorization struct {
	ID             uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	ResourceID     uint           `json:"resource_id" gorm:"not null;uniqueIndex"`
	Status         string         `json:"status" gorm:"size:20;not null;default:pending;index"`
	EvidenceType   string         `json:"evidence_type" gorm:"size:50;not null"`
	EvidenceRef    string         `json:"evidence_ref" gorm:"size:500;not null"`
	RetentionUntil *time.Time     `json:"retention_until"`
	VerifiedBy     string         `json:"verified_by" gorm:"size:100"`
	VerifiedAt     *time.Time     `json:"verified_at"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `json:"-" gorm:"index"`
}

func (ResourceAuthorization) TableName() string { return "resource_authorizations" }

