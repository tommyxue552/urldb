package pan

import "testing"

func TestAuthorizedTransferProviderPolicyForFailsClosed(t *testing.T) {
	t.Setenv("AUTHORIZED_TRANSFER_APPROVED_PROVIDERS", "")
	if _, err := AuthorizedTransferProviderPolicyFor(BaiduPan); err == nil {
		t.Fatal("expected an unconfigured provider to be rejected")
	}
}

func TestAuthorizedTransferProviderPolicyForRequiresApprovalAndRetention(t *testing.T) {
	t.Setenv("AUTHORIZED_TRANSFER_APPROVED_PROVIDERS", "baidu")
	t.Setenv("AUTHORIZED_TRANSFER_PROVIDER_BAIDU_APPROVAL_REF", "https://example.test/provider-agreement")
	t.Setenv("AUTHORIZED_TRANSFER_PROVIDER_BAIDU_MAX_SHARE_RETENTION_DAYS", "30")

	policy, err := AuthorizedTransferProviderPolicyFor(BaiduPan)
	if err != nil {
		t.Fatalf("expected approved provider policy, got %v", err)
	}
	if policy.ApprovalReference == "" || policy.MaxShareRetentionInDays != 30 {
		t.Fatalf("unexpected policy: %#v", policy)
	}
}

func TestAuthorizedTransferProviderPolicyForRejectsInvalidRetention(t *testing.T) {
	t.Setenv("AUTHORIZED_TRANSFER_APPROVED_PROVIDERS", "baidu")
	t.Setenv("AUTHORIZED_TRANSFER_PROVIDER_BAIDU_APPROVAL_REF", "https://example.test/provider-agreement")
	t.Setenv("AUTHORIZED_TRANSFER_PROVIDER_BAIDU_MAX_SHARE_RETENTION_DAYS", "0")
	if _, err := AuthorizedTransferProviderPolicyFor(BaiduPan); err == nil {
		t.Fatal("expected non-positive retention to be rejected")
	}
}
