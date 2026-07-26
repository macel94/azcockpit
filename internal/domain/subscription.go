package domain

// Subscription represents an Azure subscription within a tenant.
// Subscriptions are the billing and resource-containment boundary.
type Subscription struct {
	// ID is the unique subscription identifier (GUID).
	// This is the value used for `az account set --subscription <ID>`.
	ID string `json:"id"`

	// DisplayName is the human-readable name (e.g., "Pay-As-You-Go" or "Production").
	DisplayName string `json:"displayName"`

	// TenantID is the GUID of the Azure AD tenant this subscription belongs to.
	TenantID string `json:"tenantId"`

	// State indicates the subscription's provisioning state
	// (e.g., "Enabled", "Disabled", "Warned", "PastDue", "Deleted").
	State string `json:"state"`

	// SubscriptionPolicies holds quotas and spending limits.
	SubscriptionPolicies SubscriptionPolicies `json:"subscriptionPolicies,omitempty"`
}

// SubscriptionPolicies contains billing and quota policy metadata.
type SubscriptionPolicies struct {
	// LocationPlacementID is the region-centric grouping ID.
	LocationPlacementID string `json:"locationPlacementId,omitempty"`

	// QuotaID is the resource quota identifier.
	QuotaID string `json:"quotaId,omitempty"`

	// SpendingLimit indicates whether spending is capped
	// ("On", "Off", "CurrentPeriodOff").
	SpendingLimit string `json:"spendingLimit,omitempty"`
}

// IsActive returns true if the subscription is in a usable state.
func (s Subscription) IsActive() bool {
	return s.State == "Enabled"
}
