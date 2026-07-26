// Package infrastructure implements the Azure client and local cache.
// It is the only layer that imports Azure SDK types, keeping
// domain and UI layers free of vendor lock-in.
package infrastructure

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/keyvault/armkeyvault"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"

	"github.com/nousresearch/azcockpit/internal/domain"
)

// AzureClient is the top-level interface that the UI layer depends on.
// This enforces DDD — the domain and UI never import Azure SDK types directly.
type AzureClient interface {
	// ListTenants returns all tenants the authenticated identity has access to.
	ListTenants(ctx context.Context) ([]domain.Tenant, error)

	// ListSubscriptions returns all subscriptions across all tenants.
	// NOTE: The ARM subscription model does not include a TenantID field,
	// so tenant-scoped filtering is not supported at this layer.
	ListSubscriptions(ctx context.Context) ([]domain.Subscription, error)

	// ListKeyVaults returns all Key Vaults in the given subscription.
	ListKeyVaults(ctx context.Context, subscriptionID string) ([]domain.KeyVault, error)

	// ListKeyVaultSecrets lists all secrets in a given vault.
	ListKeyVaultSecrets(ctx context.Context, vaultURI string) ([]domain.KeyVaultSecret, error)

	// GetKeyVaultSecret retrieves the value of a specific secret.
	GetKeyVaultSecret(ctx context.Context, vaultURI, secretName string) (string, error)

	// GetCredential returns the underlying credential for shared use.
	GetCredential() azcore.TokenCredential
}

// azureClient is the concrete implementation of AzureClient.
type azureClient struct {
	credential  azcore.TokenCredential
	secretsMu   sync.RWMutex
	secretsByVault map[string]*keyVaultSecretsClient
}

// NewAzureClient creates a new AzureClient using DefaultAzureCredential.
// DefaultAzureCredential chains:
//
//	EnvironmentCredential → ManagedIdentityCredential → AzureCLICredential
//
// This means it works seamlessly with `az login`, managed identities,
// and service principals.
func NewAzureClient() (AzureClient, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create DefaultAzureCredential: %w", err)
	}

	return &azureClient{
		credential: cred,
	}, nil
}

func (c *azureClient) GetCredential() azcore.TokenCredential {
	return c.credential
}

// ListTenants returns all Azure AD tenants visible to the authenticated user.
// NOTE: The ARM tenants API only returns TenantID and a resource ID — it does
// not include DisplayName, DefaultDomain, or TenantCategory.
// Those fields will be empty in the returned domain.Tenant values unless
// a future enhancement uses the Microsoft Graph API to enrich them.
func (c *azureClient) ListTenants(ctx context.Context) ([]domain.Tenant, error) {
	client, err := armsubscription.NewTenantsClient(c.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create tenants client: %w", err)
	}

	var tenants []domain.Tenant

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list tenants: %w", err)
		}

		for _, t := range page.Value {
			if t == nil {
				continue
			}
			tenant := domain.Tenant{
				ID: derefString(t.TenantID),
			}
			// ARM tenants API does not return DisplayName, DefaultDomain,
			// TenantCategory, or CountryCode. Set a placeholder for now.
			if tenant.ID != "" {
				tenant.DisplayName = tenant.ID // fallback: show GUID until enriched
			}
			tenants = append(tenants, tenant)
		}
	}

	return tenants, nil
}

// ListSubscriptions returns all subscriptions the authenticated user
// has access to across all tenants.
func (c *azureClient) ListSubscriptions(ctx context.Context) ([]domain.Subscription, error) {
	client, err := armsubscription.NewSubscriptionsClient(c.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriptions client: %w", err)
	}

	var subscriptions []domain.Subscription

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list subscriptions: %w", err)
		}

		for _, s := range page.Value {
			if s == nil {
				continue
			}

			sub := domain.Subscription{
				ID:          derefString(s.SubscriptionID),
				DisplayName: derefString(s.DisplayName),
			}

			if s.State != nil {
				sub.State = string(*s.State)
			}

			if s.SubscriptionPolicies != nil {
				sub.SubscriptionPolicies = domain.SubscriptionPolicies{
					LocationPlacementID: derefString(s.SubscriptionPolicies.LocationPlacementID),
					QuotaID:             derefString(s.SubscriptionPolicies.QuotaID),
				}
				if s.SubscriptionPolicies.SpendingLimit != nil {
					sub.SubscriptionPolicies.SpendingLimit = string(*s.SubscriptionPolicies.SpendingLimit)
				}
			}

			subscriptions = append(subscriptions, sub)
		}
	}

	return subscriptions, nil
}

// ListKeyVaults returns all Key Vaults in the given subscription.
// Uses the ARM Key Vault management plane to enumerate vaults.
// NOTE: The list operation only returns basic resource metadata
// (ID, Name, Location, Tags). Full properties (SKU, VaultURI, etc.)
// require a per-vault Get call and are not populated here for Phase 1.
func (c *azureClient) ListKeyVaults(ctx context.Context, subscriptionID string) ([]domain.KeyVault, error) {
	client, err := armkeyvault.NewVaultsClient(subscriptionID, c.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create vaults client: %w", err)
	}

	var vaults []domain.KeyVault

	pager := client.NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list vaults: %w", err)
		}

		for _, v := range page.Value {
			if v == nil {
				continue
			}

			kv := domain.KeyVault{
				ID:             derefString(v.ID),
				Name:           derefString(v.Name),
				Location:       derefString(v.Location),
				SubscriptionID: subscriptionID,
				Tags:           copyStringMap(v.Tags),
			}

			// Extract resource group from the ARM ID.
			if rg := extractResourceGroup(kv.ID); rg != "" {
				kv.ResourceGroup = rg
			}

			vaults = append(vaults, kv)
		}
	}

	return vaults, nil
}

// ListKeyVaultSecrets lists all secrets in the given vault by name (not value).
func (c *azureClient) ListKeyVaultSecrets(ctx context.Context, vaultURI string) ([]domain.KeyVaultSecret, error) {
	client, err := c.getSecretsClient(vaultURI)
	if err != nil {
		return nil, fmt.Errorf("ListKeyVaultSecrets: %w", err)
	}
	return client.listSecrets(ctx)
}

// GetKeyVaultSecret retrieves the current value of a secret.
func (c *azureClient) GetKeyVaultSecret(ctx context.Context, vaultURI, secretName string) (string, error) {
	client, err := c.getSecretsClient(vaultURI)
	if err != nil {
		return "", fmt.Errorf("GetKeyVaultSecret: %w", err)
	}
	return client.getSecret(ctx, secretName)
}

// getSecretsClient returns a cached or newly created data-plane secrets
// client for the given vault URI.
func (c *azureClient) getSecretsClient(vaultURI string) (*keyVaultSecretsClient, error) {
	c.secretsMu.RLock()
	client, ok := c.secretsByVault[vaultURI]
	c.secretsMu.RUnlock()
	if ok && client != nil {
		return client, nil
	}

	c.secretsMu.Lock()
	defer c.secretsMu.Unlock()

	// Double-check after acquiring write lock.
	if client, ok = c.secretsByVault[vaultURI]; ok && client != nil {
		return client, nil
	}

	client, err := newKeyVaultSecretsClient(vaultURI, c.credential)
	if err != nil {
		return nil, err
	}

	if c.secretsByVault == nil {
		c.secretsByVault = make(map[string]*keyVaultSecretsClient)
	}
	c.secretsByVault[vaultURI] = client

	return client, nil
}

// ----- helpers -----

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

// stringValue converts a typed string pointer (e.g., *SKUFamily, *SKUName)
// to a plain string.
func stringValue[T ~string](s *T) string {
	if s == nil {
		return ""
	}
	return string(*s)
}

func copyStringMap(m map[string]*string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if v != nil {
			out[k] = *v
		}
	}
	return out
}

// extractResourceGroup pulls the resource group name from an ARM resource ID.
// Format: /subscriptions/{guid}/resourceGroups/{rgName}/providers/...
func extractResourceGroup(armID string) string {
	const rgSegment = "resourceGroups"
	parts := strings.Split(armID, "/")
	for i, part := range parts {
		if strings.EqualFold(part, rgSegment) && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
