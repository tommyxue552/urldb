package pan

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// AuthorizedTransferProviderPolicy is the deployment-time attestation needed
// before an automated transfer provider may be used. Existing provider clients
// are not implicitly approved: an operator must record the official API or
// provider-granted integration reference and the maximum share retention.
//
// The values deliberately live outside the database because they are a
// deployment/compliance control, not a resource-level authorization. Resource
// authorization still supplies the evidence and retention for each resource.
type AuthorizedTransferProviderPolicy struct {
	Provider                ServiceType
	ApprovalReference       string
	MaxShareRetentionInDays int
}

// ErrAuthorizedTransferProviderNotApproved is returned when a provider has no
// explicit deployment approval. This is fail-closed by design.
type ErrAuthorizedTransferProviderNotApproved struct {
	Provider string
	Reason   string
}

func (e *ErrAuthorizedTransferProviderNotApproved) Error() string {
	return fmt.Sprintf("authorized transfer provider %q is not approved: %s", e.Provider, e.Reason)
}

// AuthorizedTransferProviderPolicyFor reads the provider approval declaration.
// A provider must appear in AUTHORIZED_TRANSFER_APPROVED_PROVIDERS and define:
//
//	AUTHORIZED_TRANSFER_PROVIDER_<NAME>_APPROVAL_REF
//	AUTHORIZED_TRANSFER_PROVIDER_<NAME>_MAX_SHARE_RETENTION_DAYS
//
// The reference must point to the applicable official API documentation or a
// provider-granted integration agreement. A non-positive retention is refused
// so a deployment cannot silently assume permanent public sharing.
func AuthorizedTransferProviderPolicyFor(serviceType ServiceType) (AuthorizedTransferProviderPolicy, error) {
	provider := serviceType.String()
	if provider == "unknown" {
		return AuthorizedTransferProviderPolicy{}, &ErrAuthorizedTransferProviderNotApproved{Provider: provider, Reason: "unsupported provider"}
	}
	if !isProviderListed(provider, os.Getenv("AUTHORIZED_TRANSFER_APPROVED_PROVIDERS")) {
		return AuthorizedTransferProviderPolicy{}, &ErrAuthorizedTransferProviderNotApproved{Provider: provider, Reason: "not listed in AUTHORIZED_TRANSFER_APPROVED_PROVIDERS"}
	}

	prefix := "AUTHORIZED_TRANSFER_PROVIDER_" + strings.ToUpper(strings.ReplaceAll(provider, "-", "_")) + "_"
	approvalReference := strings.TrimSpace(os.Getenv(prefix + "APPROVAL_REF"))
	if approvalReference == "" {
		return AuthorizedTransferProviderPolicy{}, &ErrAuthorizedTransferProviderNotApproved{Provider: provider, Reason: "missing " + prefix + "APPROVAL_REF"}
	}
	retentionDays, err := strconv.Atoi(strings.TrimSpace(os.Getenv(prefix + "MAX_SHARE_RETENTION_DAYS")))
	if err != nil || retentionDays <= 0 {
		return AuthorizedTransferProviderPolicy{}, &ErrAuthorizedTransferProviderNotApproved{Provider: provider, Reason: "missing or invalid " + prefix + "MAX_SHARE_RETENTION_DAYS"}
	}

	return AuthorizedTransferProviderPolicy{
		Provider:                serviceType,
		ApprovalReference:       approvalReference,
		MaxShareRetentionInDays: retentionDays,
	}, nil
}

func isProviderListed(provider, configured string) bool {
	for _, value := range strings.Split(configured, ",") {
		if strings.EqualFold(strings.TrimSpace(value), provider) {
			return true
		}
	}
	return false
}
