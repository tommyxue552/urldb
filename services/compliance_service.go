package services

import (
	"time"

	pan "github.com/ctwj/urldb/common"
	"github.com/ctwj/urldb/db/entity"
	"gorm.io/gorm"
)

// ProviderComplianceStatus is the operator-visible result of the provider
// contract and deployment approval checks. Approval references are
// configuration metadata, not credentials, so they can be shown to admins.
type ProviderComplianceStatus struct {
	Provider                      string `json:"provider"`
	ImplementationAvailable       bool   `json:"implementation_available"`
	TransferContractAvailable     bool   `json:"transfer_contract_available"`
	ApprovalConfigured            bool   `json:"approval_configured"`
	EligibleForAuthorizedTransfer bool   `json:"eligible_for_authorized_transfer"`
	ApprovalReference             string `json:"approval_reference,omitempty"`
	MaxShareRetentionInDays       int    `json:"max_share_retention_days,omitempty"`
	Reason                        string `json:"reason,omitempty"`
}

// ComplianceMetrics contains only aggregate operational data. It deliberately
// does not include source URLs, share URLs, account credentials, or evidence
// contents.
type ComplianceMetrics struct {
	AuthorizationRecords         int64 `json:"authorization_records"`
	ActiveAuthorizations         int64 `json:"active_authorizations"`
	ExpiredAuthorizations        int64 `json:"expired_authorizations"`
	AuthorizationsExpiringSoon   int64 `json:"authorizations_expiring_soon"`
	OwnedShareRecords            int64 `json:"owned_share_records"`
	ActiveOwnedShares            int64 `json:"active_owned_shares"`
	InvalidOwnedShares           int64 `json:"invalid_owned_shares"`
	ExpiredOwnedShares           int64 `json:"expired_owned_shares"`
	OwnedSharesExpiringSoon      int64 `json:"owned_shares_expiring_soon"`
	AuthorizedTransferTasks      int64 `json:"authorized_transfer_tasks"`
	PendingAuthorizedTransfers   int64 `json:"pending_authorized_transfers"`
	RunningAuthorizedTransfers   int64 `json:"running_authorized_transfers"`
	CompletedAuthorizedTransfers int64 `json:"completed_authorized_transfers"`
	FailedAuthorizedTransfers    int64 `json:"failed_authorized_transfers"`
	BlockedProviderCount         int64 `json:"blocked_provider_count"`
}

// ComplianceDashboard is the aggregate P3 report used by the administrator
// operations page and can also be exported by an authenticated operator.
type ComplianceDashboard struct {
	GeneratedAt        time.Time                  `json:"generated_at"`
	ExpiringWithinDays int                        `json:"expiring_within_days"`
	Providers          []ProviderComplianceStatus `json:"providers"`
	Metrics            ComplianceMetrics          `json:"metrics"`
}

// NewComplianceDashboardService creates a read-only compliance report service.
func NewComplianceDashboardService(db *gorm.DB) *ComplianceDashboardService {
	return &ComplianceDashboardService{db: db}
}

type ComplianceDashboardService struct {
	db *gorm.DB
}

// BuildProviderComplianceStatuses checks every provider represented by the
// current provider registry. It does not make network calls or validate a
// credential; those checks remain part of account management and execution.
func BuildProviderComplianceStatuses() []ProviderComplianceStatus {
	providers := []pan.ServiceType{pan.Quark, pan.Alipan, pan.BaiduPan, pan.UC, pan.Xunlei, pan.Tianyi, pan.Pan123, pan.Pan115}
	factory := pan.NewPanFactory()
	statuses := make([]ProviderComplianceStatus, 0, len(providers))
	for _, provider := range providers {
		status := ProviderComplianceStatus{Provider: provider.String()}
		service, factoryErr := factory.CreatePanServiceByType(provider, &pan.PanConfig{})
		if factoryErr == nil && service != nil {
			status.ImplementationAvailable = true
			status.TransferContractAvailable = service.GetServiceType() == provider
		} else if factoryErr != nil {
			status.Reason = factoryErr.Error()
		}

		policy, policyErr := pan.AuthorizedTransferProviderPolicyFor(provider)
		if policyErr == nil {
			status.ApprovalConfigured = true
			status.ApprovalReference = policy.ApprovalReference
			status.MaxShareRetentionInDays = policy.MaxShareRetentionInDays
		} else if status.Reason == "" {
			status.Reason = policyErr.Error()
		} else {
			status.Reason += "; " + policyErr.Error()
		}
		status.EligibleForAuthorizedTransfer = status.ImplementationAvailable && status.TransferContractAvailable && status.ApprovalConfigured
		if status.EligibleForAuthorizedTransfer {
			status.Reason = "provider implementation and deployment approval are present"
		}
		statuses = append(statuses, status)
	}
	return statuses
}

// GetDashboard returns aggregate compliance and operations metrics. The
// seven-day window is intentionally fixed in the API response so reports are
// comparable across refreshes and cannot accidentally expose arbitrary rows.
func (s *ComplianceDashboardService) GetDashboard() (*ComplianceDashboard, error) {
	if s == nil || s.db == nil {
		return nil, gorm.ErrInvalidDB
	}
	now := time.Now()
	horizon := now.AddDate(0, 0, 7)
	metrics := ComplianceMetrics{}
	counts := []struct {
		target *int64
		model  interface{}
		query  string
		args   []interface{}
	}{
		{&metrics.AuthorizationRecords, &entity.ResourceAuthorization{}, "", nil},
		{&metrics.ActiveAuthorizations, &entity.ResourceAuthorization{}, "status = ? AND (retention_until IS NULL OR retention_until > ?)", []interface{}{"active", now}},
		{&metrics.ExpiredAuthorizations, &entity.ResourceAuthorization{}, "status = ? AND retention_until IS NOT NULL AND retention_until <= ?", []interface{}{"active", now}},
		{&metrics.AuthorizationsExpiringSoon, &entity.ResourceAuthorization{}, "status = ? AND retention_until IS NOT NULL AND retention_until > ? AND retention_until <= ?", []interface{}{"active", now, horizon}},
		{&metrics.OwnedShareRecords, &entity.OwnedShare{}, "", nil},
		{&metrics.ActiveOwnedShares, &entity.OwnedShare{}, "status = ? AND (expires_at IS NULL OR expires_at > ?)", []interface{}{"active", now}},
		{&metrics.InvalidOwnedShares, &entity.OwnedShare{}, "status = ?", []interface{}{"invalid"}},
		{&metrics.ExpiredOwnedShares, &entity.OwnedShare{}, "status = ? AND expires_at IS NOT NULL AND expires_at <= ?", []interface{}{"active", now}},
		{&metrics.OwnedSharesExpiringSoon, &entity.OwnedShare{}, "status = ? AND expires_at IS NOT NULL AND expires_at > ? AND expires_at <= ?", []interface{}{"active", now, horizon}},
		{&metrics.AuthorizedTransferTasks, &entity.Task{}, "type = ?", []interface{}{entity.TaskTypeAuthorizedTransfer}},
		{&metrics.PendingAuthorizedTransfers, &entity.Task{}, "type = ? AND status = ?", []interface{}{entity.TaskTypeAuthorizedTransfer, entity.TaskStatusPending}},
		{&metrics.RunningAuthorizedTransfers, &entity.Task{}, "type = ? AND status = ?", []interface{}{entity.TaskTypeAuthorizedTransfer, entity.TaskStatusRunning}},
		{&metrics.CompletedAuthorizedTransfers, &entity.Task{}, "type = ? AND status = ?", []interface{}{entity.TaskTypeAuthorizedTransfer, entity.TaskStatusCompleted}},
		{&metrics.FailedAuthorizedTransfers, &entity.Task{}, "type = ? AND status = ?", []interface{}{entity.TaskTypeAuthorizedTransfer, entity.TaskStatusFailed}},
	}
	for _, item := range counts {
		query := s.db.Model(item.model)
		if item.query != "" {
			query = query.Where(item.query, item.args...)
		}
		if err := query.Count(item.target).Error; err != nil {
			return nil, err
		}
	}

	providers := BuildProviderComplianceStatuses()
	for _, provider := range providers {
		if !provider.EligibleForAuthorizedTransfer {
			metrics.BlockedProviderCount++
		}
	}
	return &ComplianceDashboard{
		GeneratedAt:        now.UTC(),
		ExpiringWithinDays: 7,
		Providers:          providers,
		Metrics:            metrics,
	}, nil
}
