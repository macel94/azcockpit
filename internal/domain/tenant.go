// Package domain contains the core domain models for AzCockpit.
// These types represent Azure concepts (Tenant, Subscription, KeyVault)
// with zero dependencies on infrastructure or presentation layers.
package domain

// Tenant represents an Azure Active Directory tenant.
// A tenant is the top-level organizational unit that contains
// one or more subscriptions.
type Tenant struct {
	// ID is the unique tenant identifier (GUID).
	ID string `json:"id"`

	// DisplayName is the human-readable name shown in the Azure Portal.
	DisplayName string `json:"displayName"`

	// DefaultDomain is the primary domain for this tenant
	// (e.g., "contoso.onmicrosoft.com").
	DefaultDomain string `json:"defaultDomain,omitempty"`

	// TenantCategory describes the type of tenant
	// (e.g., "Home", "ManagedBy", "Pending").
	TenantCategory string `json:"tenantCategory,omitempty"`

	// CountryCode is the two-letter country code for the tenant.
	CountryCode string `json:"countryCode,omitempty"`
}

// IsHomeTenant returns true if this is the user's home tenant.
func (t Tenant) IsHomeTenant() bool {
	return t.TenantCategory == "Home"
}
