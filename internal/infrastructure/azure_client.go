// Package infrastructure implements the Azure client and local cache.
// It is the only layer that imports Azure SDK types, keeping
// domain and UI layers free of vendor lock-in.
package infrastructure

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os/exec"
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
	ListSubscriptions(ctx context.Context) ([]domain.Subscription, error)

	// ListKeyVaults returns all Key Vaults in the given subscription.
	ListKeyVaults(ctx context.Context, subscriptionID string) ([]domain.KeyVault, error)

	// ListKeyVaultSecrets lists all secrets in a given vault.
	ListKeyVaultSecrets(ctx context.Context, vaultURI string) ([]domain.KeyVaultSecret, error)

	// GetKeyVaultSecret retrieves the value of a specific secret.
	GetKeyVaultSecret(ctx context.Context, vaultURI, secretName string) (string, error)

	// SetActiveSubscription switches the Azure CLI active subscription
	// to the given subscription within the specified tenant.
	// If tenantID is empty, the --tenant flag is omitted.
	SetActiveSubscription(ctx context.Context, subscriptionID, tenantID string) error

	// InitializeExample creates an example Key Vault with sample secrets
	// in the given subscription and location.
	InitializeExample(ctx context.Context, subscriptionID, location string) (domain.KeyVault, []domain.KeyVaultSecret, error)

	// GetCredential returns the underlying credential for shared use.
	GetCredential() azcore.TokenCredential
}

// azureClient is the concrete implementation of AzureClient.
type azureClient struct {
	credential     azcore.TokenCredential
	secretsMu      sync.RWMutex
	secretsByVault map[string]*keyVaultSecretsClient
}

// NewAzureClient creates a new AzureClient using DefaultAzureCredential.
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
			if tenant.ID != "" {
				tenant.DisplayName = tenant.ID
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

// ListKeyVaults returns all Key Vaults in the given subscription with full properties.
func (c *azureClient) ListKeyVaults(ctx context.Context, subscriptionID string) ([]domain.KeyVault, error) {
	client, err := armkeyvault.NewVaultsClient(subscriptionID, c.credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create vaults client: %w", err)
	}

	var vaults []domain.KeyVault

	pager := client.NewListBySubscriptionPager(nil)
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

			if rg := extractResourceGroup(kv.ID); rg != "" {
				kv.ResourceGroup = rg
			}

			if v.Properties != nil {
				kv.Properties.VaultURI = derefString(v.Properties.VaultURI)
				kv.Properties.TenantID = derefString(v.Properties.TenantID)
				kv.Properties.EnableSoftDelete = derefBool(v.Properties.EnableSoftDelete)
				kv.Properties.EnablePurgeProtection = derefBool(v.Properties.EnablePurgeProtection)
				kv.Properties.EnableRBACAuthorization = derefBool(v.Properties.EnableRbacAuthorization)
				if v.Properties.SKU != nil {
					kv.Properties.SKU = domain.KeyVaultSKU{
						Family: stringValue(v.Properties.SKU.Family),
						Name:   stringValue(v.Properties.SKU.Name),
					}
				}
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

// getSecretsClient returns a cached or newly created data-plane secrets client.
func (c *azureClient) getSecretsClient(vaultURI string) (*keyVaultSecretsClient, error) {
	c.secretsMu.RLock()
	client, ok := c.secretsByVault[vaultURI]
	c.secretsMu.RUnlock()
	if ok && client != nil {
		return client, nil
	}

	c.secretsMu.Lock()
	defer c.secretsMu.Unlock()

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

// SetActiveSubscription switches the active Azure CLI subscription
// and optionally the tenant via `az account set`.
func (c *azureClient) SetActiveSubscription(ctx context.Context, subscriptionID, tenantID string) error {
	azPath, err := exec.LookPath("az")
	if err != nil {
		return fmt.Errorf("azure CLI (az) not found in PATH: %w", err)
	}

	args := []string{"account", "set", "--subscription", subscriptionID}
	if tenantID != "" {
		args = append(args, "--tenant", tenantID)
	}

	cmd := exec.CommandContext(ctx, azPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set az account: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// InitializeExample creates an example Key Vault with a random name
// and populates it with randomly generated sample secrets.
// The vault is created with the access-policy permission model and the
// current user is granted full secret permissions so that the data-plane
// azsecrets client can list and get secrets without authorization errors.
func (c *azureClient) InitializeExample(ctx context.Context, subscriptionID, location string) (domain.KeyVault, []domain.KeyVaultSecret, error) {
	azPath, err := exec.LookPath("az")
	if err != nil {
		return domain.KeyVault{}, nil, fmt.Errorf("azure CLI (az) not found in PATH: %w", err)
	}

	// Generate a random vault name: azcockpit-demo-<8 random hex chars>
	randomSuffix := randomHex(4) // 4 bytes = 8 hex chars
	vaultName := "azcockpit-demo-" + randomSuffix

	// Ensure resource group exists (ignore error if it already exists).
	rgName := "azcockpit-demo-rg"
	_ = exec.CommandContext(ctx, azPath, "group", "create",
		"--name", rgName,
		"--location", location,
		"--subscription", subscriptionID,
	).Run()

	// Register the Microsoft.KeyVault resource provider if not already
	// registered. This is idempotent — the subscription's RP registration
	// state is unaffected by repeated calls.
	_ = exec.CommandContext(ctx, azPath, "provider", "register",
		"--namespace", "Microsoft.KeyVault",
		"--subscription", subscriptionID,
	).Run()

	// Create the Key Vault with explicit access-policy permission model.
	// NOTE: newer Azure CLI versions default to RBAC, which does NOT
	// automatically grant the creator data-plane permissions. Using
	// --enable-rbac-authorization false ensures the classic access-policy
	// model is used, and we then add the user to the access policies below.
	cmd := exec.CommandContext(ctx, azPath, "keyvault", "create",
		"--name", vaultName,
		"--subscription", subscriptionID,
		"--location", location,
		"--resource-group", rgName,
		"--enable-rbac-authorization", "false",
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return domain.KeyVault{}, nil, fmt.Errorf("failed to create key vault: %s: %w", strings.TrimSpace(string(output)), err)
	}

	// Best-effort: add the current user to the vault's access policies
	// with full secret permissions. This is idempotent and ensures the
	// data-plane azsecrets client can list and get secrets.
	if oid := getCurrentUserObjectID(ctx, azPath); oid != "" {
		_ = exec.CommandContext(ctx, azPath, "keyvault", "set-policy",
			"--name", vaultName,
			"--subscription", subscriptionID,
			"--object-id", oid,
			"--secret-permissions", "get", "list",
		).Run()
	}

	// Create sample secrets with random values.
	type secretDef struct {
		name  string
		value string
	}
	secretDefs := []secretDef{
		{name: "DEMO_DB_PASSWORD", value: randomAlphaNumeric(16)},
		{name: "DEMO_API_KEY", value: randomHex(16)},
		{name: "DEMO_STORAGE_CONNECTION_STRING", value: "DefaultEndpointsProtocol=https;AccountName=demo;AccountKey=" + randomHex(16)},
	}

	var secrets []domain.KeyVaultSecret
	for _, sd := range secretDefs {
		secretCmd := exec.CommandContext(ctx, azPath, "keyvault", "secret", "set",
			"--vault-name", vaultName,
			"--subscription", subscriptionID,
			"--name", sd.name,
			"--value", sd.value,
		)
		secretOutput, err := secretCmd.CombinedOutput()
		if err != nil {
			// Log warning but continue — best-effort secret creation.
			_ = secretOutput
			continue
		}
		secrets = append(secrets, domain.KeyVaultSecret{
			Name:    sd.name,
			Enabled: true,
		})
	}

	vaultURI := fmt.Sprintf("https://%s.vault.azure.net/", vaultName)

	vault := domain.KeyVault{
		Name:           vaultName,
		Location:       location,
		SubscriptionID: subscriptionID,
		ResourceGroup:  rgName,
		Properties: domain.KeyVaultProperties{
			VaultURI: vaultURI,
		},
	}

	return vault, secrets, nil
}

// randomAlphaNumeric returns a random alphanumeric string of the given length.
func randomAlphaNumeric(n int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("rand.Read failed: %v", err))
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b)
}

// randomHex returns a random hex-encoded string of the given byte length.
func randomHex(byteLen int) string {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Sprintf("rand.Read failed: %v", err))
	}
	return hex.EncodeToString(b)
}

// ----- helpers -----

// getCurrentUserObjectID returns the object ID of the currently authenticated
// Azure user by calling `az ad signed-in-user show`. Returns empty string if
// the user cannot be determined (e.g. service principal, no `az` CLI, or
// missing permissions).
func getCurrentUserObjectID(ctx context.Context, azPath string) string {
	cmd := exec.CommandContext(ctx, azPath, "ad", "signed-in-user", "show", "--query", "id", "-o", "tsv")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

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

// stringValue converts a typed string pointer to a plain string.
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