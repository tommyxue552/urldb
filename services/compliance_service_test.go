package services

import (
	"testing"

	pan "github.com/ctwj/urldb/common"
)

func TestBuildProviderComplianceStatusesFailsClosedByDefault(t *testing.T) {
	t.Setenv("AUTHORIZED_TRANSFER_APPROVED_PROVIDERS", "")
	statuses := BuildProviderComplianceStatuses()
	if len(statuses) != 8 {
		t.Fatalf("provider status count = %d, want 8", len(statuses))
	}
	for _, status := range statuses {
		if status.EligibleForAuthorizedTransfer {
			t.Fatalf("provider %s should be blocked without deployment approval", status.Provider)
		}
	}
}

func TestBuildProviderComplianceStatusesReportsImplementationAndApprovalSeparately(t *testing.T) {
	t.Setenv("AUTHORIZED_TRANSFER_APPROVED_PROVIDERS", "baidu")
	t.Setenv("AUTHORIZED_TRANSFER_PROVIDER_BAIDU_APPROVAL_REF", "https://example.test/approval")
	t.Setenv("AUTHORIZED_TRANSFER_PROVIDER_BAIDU_MAX_SHARE_RETENTION_DAYS", "30")

	statuses := BuildProviderComplianceStatuses()
	var baidu, tianyi *ProviderComplianceStatus
	for index := range statuses {
		if statuses[index].Provider == pan.BaiduPan.String() {
			baidu = &statuses[index]
		}
		if statuses[index].Provider == pan.Tianyi.String() {
			tianyi = &statuses[index]
		}
	}
	if baidu == nil || !baidu.ImplementationAvailable || !baidu.TransferContractAvailable || !baidu.ApprovalConfigured || !baidu.EligibleForAuthorizedTransfer {
		t.Fatalf("unexpected approved baidu status: %#v", baidu)
	}
	if tianyi == nil || tianyi.ImplementationAvailable || tianyi.ApprovalConfigured || tianyi.EligibleForAuthorizedTransfer {
		t.Fatalf("unexpected unavailable tianyi status: %#v", tianyi)
	}
}
