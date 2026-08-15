package entity

import "time"

// CredentialAudit is append-only metadata for account credential operations.
// It intentionally contains no plaintext, ciphertext, or request payload data.
type CredentialAudit struct {
	ID         uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	AccountID  uint      `json:"account_id" gorm:"not null;index"`
	Provider   string    `json:"provider" gorm:"size:64;not null"`
	Action     string    `json:"action" gorm:"size:32;not null"`
	Outcome    string    `json:"outcome" gorm:"size:16;not null"`
	Actor      string    `json:"actor" gorm:"size:128;not null"`
	SourceIP   string    `json:"source_ip" gorm:"size:64"`
	RequestID  string    `json:"request_id" gorm:"size:128"`
	KeyVersion string    `json:"key_version" gorm:"size:16;not null"`
	CreatedAt  time.Time `json:"created_at"`
}

func (CredentialAudit) TableName() string { return "credential_audits" }
