package domain

import "time"

// KeyVault represents an Azure Key Vault instance.
type KeyVault struct {
	// ID is the fully qualified Azure resource ID.
	ID string `json:"id"`

	// Name is the vault name (must be globally unique within Azure).
	Name string `json:"name"`

	// Location is the Azure region (e.g., "westus2").
	Location string `json:"location"`

	// ResourceGroup is the name of the containing resource group.
	ResourceGroup string `json:"resourceGroup"`

	// SubscriptionID is the subscription this vault belongs to.
	SubscriptionID string `json:"subscriptionId"`

	// Properties contains vault-specific metadata (SKU, tenant, URIs).
	Properties KeyVaultProperties `json:"properties"`

	// Tags are user-defined Azure tags on the resource.
	Tags map[string]string `json:"tags,omitempty"`
}

// KeyVaultProperties holds the core configuration of a Key Vault.
type KeyVaultProperties struct {
	// SKU is the pricing tier ("standard" or "premium").
	SKU KeyVaultSKU `json:"sku"`

	// TenantID is the Azure AD tenant the vault is associated with.
	TenantID string `json:"tenantId"`

	// VaultURI is the DNS name of the vault
	// (e.g., "https://myvault.vault.azure.net/").
	VaultURI string `json:"vaultUri"`

	// EnableSoftDelete indicates whether soft-delete is enabled.
	EnableSoftDelete bool `json:"enableSoftDelete,omitempty"`

	// EnablePurgeProtection indicates whether purge protection is enabled.
	EnablePurgeProtection bool `json:"enablePurgeProtection,omitempty"`

	// EnableRBACAuthorization means the vault uses Azure RBAC
	// instead of access policies.
	EnableRBACAuthorization bool `json:"enableRbacAuthorization,omitempty"`
}

// KeyVaultSKU defines the pricing tier.
type KeyVaultSKU struct {
	// Family is always "A".
	Family string `json:"family"`

	// Name is "standard" or "premium".
	Name string `json:"name"`
}

// KeyVaultSecret represents a secret stored within a Key Vault.
// This is NOT persisted to the cache — it's fetched on-demand when
// the user browses a specific vault.
type KeyVaultSecret struct {
	// ID is the secret's full resource URI.
	ID string `json:"id"`

	// Name is the secret's name (key within the vault).
	Name string `json:"name"`

	// ContentType is an optional content type hint
	// (e.g., "application/x-pkcs12").
	ContentType string `json:"contentType,omitempty"`

	// Enabled indicates whether the secret is enabled for retrieval.
	Enabled bool `json:"enabled"`

	// NotBefore is the earliest date the secret can be used.
	NotBefore *time.Time `json:"notBefore,omitempty"`

	// Expires is the expiration date of the secret.
	Expires *time.Time `json:"expires,omitempty"`

	// Tags are secret-level metadata tags.
	Tags map[string]string `json:"tags,omitempty"`
}
