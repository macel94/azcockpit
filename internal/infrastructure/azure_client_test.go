package infrastructure

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"

	"github.com/nousresearch/azcockpit/internal/domain"
)

// ---------------------------------------------------------------------------
// mockAzureClient — test double for the AzureClient interface
// ---------------------------------------------------------------------------

type mockAzureClient struct {
	mu           sync.RWMutex
	vaults       map[string][]domain.KeyVault  // keyed by subscriptionID
	secrets      map[string][]domain.KeyVaultSecret // keyed by vaultURI
	secretsCalls map[string]int                // tracks how many times ListKeyVaultSecrets was called per URI
}

func (m *mockAzureClient) ListTenants(_ context.Context) ([]domain.Tenant, error) {
	return nil, nil
}

func (m *mockAzureClient) ListSubscriptions(_ context.Context) ([]domain.Subscription, error) {
	return nil, nil
}

func (m *mockAzureClient) ListKeyVaults(_ context.Context, subscriptionID string) ([]domain.KeyVault, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.vaults == nil {
		return nil, nil
	}
	vaults := m.vaults[subscriptionID]
	// Return a defensive copy so callers can't mutate cached data.
	out := make([]domain.KeyVault, len(vaults))
	copy(out, vaults)
	return out, nil
}

func (m *mockAzureClient) ListKeyVaultSecrets(_ context.Context, vaultURI string) ([]domain.KeyVaultSecret, error) {
	m.mu.Lock()
	if m.secretsCalls == nil {
		m.secretsCalls = make(map[string]int)
	}
	m.secretsCalls[vaultURI]++
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.secrets == nil {
		return nil, nil
	}
	secrets := m.secrets[vaultURI]
	out := make([]domain.KeyVaultSecret, len(secrets))
	copy(out, secrets)
	return out, nil
}

func (m *mockAzureClient) GetKeyVaultSecret(_ context.Context, _, _ string) (string, error) {
	return "", nil
}

func (m *mockAzureClient) SetActiveSubscription(_ context.Context, _, _ string) error {
	return nil
}

func (m *mockAzureClient) InitializeExample(_ context.Context, _, _ string) (domain.KeyVault, []domain.KeyVaultSecret, error) {
	return domain.KeyVault{}, nil, nil
}

func (m *mockAzureClient) GetCredential() azcore.TokenCredential {
	return nil
}

func (m *mockAzureClient) ExportKeyVaultSecrets(_ context.Context, vaultURI string) (map[string]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.secrets == nil {
		return nil, nil
	}
	secrets := m.secrets[vaultURI]
	values := make(map[string]string, len(secrets))
	for _, s := range secrets {
		values[s.Name] = s.Name // placeholder — tests don't need real values
	}
	return values, nil
}

// Compile-time interface check.
var _ AzureClient = (*mockAzureClient)(nil)

// ---------------------------------------------------------------------------
// fakeCredential — minimal azcore.TokenCredential for unit tests
// ---------------------------------------------------------------------------

type fakeCredential struct{}

func (f *fakeCredential) GetToken(_ context.Context, _ policy.TokenRequestOptions) (azcore.AccessToken, error) {
	return azcore.AccessToken{Token: "fake", ExpiresOn: time.Now().Add(time.Hour)}, nil
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestMockAzureClient_ListKeyVaults_VaultURI verifies that ListKeyVaults
// populates the VaultURI field on each returned vault.
func TestMockAzureClient_ListKeyVaults_VaultURI(t *testing.T) {
	mock := &mockAzureClient{
		vaults: map[string][]domain.KeyVault{
			"sub-1": {
				{
					Name: "vault-westus",
					Properties: domain.KeyVaultProperties{
						VaultURI: "https://vault-westus.vault.azure.net/",
					},
				},
				{
					Name: "vault-eastus",
					Properties: domain.KeyVaultProperties{
						VaultURI: "https://vault-eastus.vault.azure.net/",
					},
				},
			},
		},
	}

	vaults, err := mock.ListKeyVaults(context.Background(), "sub-1")
	if err != nil {
		t.Fatalf("ListKeyVaults failed: %v", err)
	}
	if len(vaults) != 2 {
		t.Fatalf("expected 2 vaults, got %d", len(vaults))
	}

	t.Run("first vault URI", func(t *testing.T) {
		want := "https://vault-westus.vault.azure.net/"
		if vaults[0].Properties.VaultURI != want {
			t.Errorf("VaultURI = %q, want %q", vaults[0].Properties.VaultURI, want)
		}
	})

	t.Run("second vault URI", func(t *testing.T) {
		want := "https://vault-eastus.vault.azure.net/"
		if vaults[1].Properties.VaultURI != want {
			t.Errorf("VaultURI = %q, want %q", vaults[1].Properties.VaultURI, want)
		}
	})
}

// TestMockAzureClient_ListKeyVaults_PropertiesFields verifies that
// ListKeyVaults populates TenantID and SKU on each returned vault.
func TestMockAzureClient_ListKeyVaults_PropertiesFields(t *testing.T) {
	standardSKU := domain.KeyVaultSKU{Family: "A", Name: "standard"}
	premiumSKU := domain.KeyVaultSKU{Family: "A", Name: "premium"}

	mock := &mockAzureClient{
		vaults: map[string][]domain.KeyVault{
			"sub-1": {
				{
					Name: "vault-a",
					Properties: domain.KeyVaultProperties{
						TenantID: "11111111-1111-1111-1111-111111111111",
						SKU:      standardSKU,
					},
				},
				{
					Name: "vault-b",
					Properties: domain.KeyVaultProperties{
						TenantID: "22222222-2222-2222-2222-222222222222",
						SKU:      premiumSKU,
					},
				},
			},
		},
	}

	vaults, err := mock.ListKeyVaults(context.Background(), "sub-1")
	if err != nil {
		t.Fatalf("ListKeyVaults failed: %v", err)
	}
	if len(vaults) != 2 {
		t.Fatalf("expected 2 vaults, got %d", len(vaults))
	}

	t.Run("first vault TenantID and SKU", func(t *testing.T) {
		if vaults[0].Properties.TenantID != "11111111-1111-1111-1111-111111111111" {
			t.Errorf("TenantID = %q, want 11111111-1111-1111-1111-111111111111",
				vaults[0].Properties.TenantID)
		}
		if vaults[0].Properties.SKU != standardSKU {
			t.Errorf("SKU = %+v, want %+v", vaults[0].Properties.SKU, standardSKU)
		}
	})

	t.Run("second vault TenantID and SKU", func(t *testing.T) {
		if vaults[1].Properties.TenantID != "22222222-2222-2222-2222-222222222222" {
			t.Errorf("TenantID = %q, want 22222222-2222-2222-2222-222222222222",
				vaults[1].Properties.TenantID)
		}
		if vaults[1].Properties.SKU != premiumSKU {
			t.Errorf("SKU = %+v, want %+v", vaults[1].Properties.SKU, premiumSKU)
		}
	})
}

// TestMockAzureClient_ListKeyVaultSecrets_PassesVaultURI verifies that
// ListKeyVaultSecrets uses the vaultURI to look up secrets so that
// different vault URIs return different secret sets.
func TestMockAzureClient_ListKeyVaultSecrets_PassesVaultURI(t *testing.T) {
	mock := &mockAzureClient{
		secrets: map[string][]domain.KeyVaultSecret{
			"https://vault-a.vault.azure.net/": {
				{Name: "secret-a1", Enabled: true},
				{Name: "secret-a2", Enabled: true},
			},
			"https://vault-b.vault.azure.net/": {
				{Name: "secret-b1", Enabled: true},
			},
		},
	}

	ctx := context.Background()

	t.Run("vault A returns its secrets", func(t *testing.T) {
		secrets, err := mock.ListKeyVaultSecrets(ctx, "https://vault-a.vault.azure.net/")
		if err != nil {
			t.Fatalf("ListKeyVaultSecrets failed: %v", err)
		}
		if len(secrets) != 2 {
			t.Fatalf("expected 2 secrets for vault-a, got %d", len(secrets))
		}
		if secrets[0].Name != "secret-a1" || secrets[1].Name != "secret-a2" {
			t.Errorf("unexpected secret names: %+v", secrets)
		}
	})

	t.Run("vault B returns its secrets", func(t *testing.T) {
		secrets, err := mock.ListKeyVaultSecrets(ctx, "https://vault-b.vault.azure.net/")
		if err != nil {
			t.Fatalf("ListKeyVaultSecrets failed: %v", err)
		}
		if len(secrets) != 1 {
			t.Fatalf("expected 1 secret for vault-b, got %d", len(secrets))
		}
		if secrets[0].Name != "secret-b1" {
			t.Errorf("unexpected secret name: %q", secrets[0].Name)
		}
	})

	t.Run("unknown vault returns nil", func(t *testing.T) {
		secrets, err := mock.ListKeyVaultSecrets(ctx, "https://unknown.vault.azure.net/")
		if err != nil {
			t.Fatalf("ListKeyVaultSecrets failed: %v", err)
		}
		if len(secrets) != 0 {
			t.Errorf("expected 0 secrets for unknown vault, got %d", len(secrets))
		}
	})
}

// TestCacheRoundTrip_VaultURI verifies that SaveVaults followed by GetVaults
// preserves the VaultURI field.
func TestCacheRoundTrip_VaultURI(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cache, err := NewCache(10 * time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	vaults := []domain.KeyVault{
		{
			Name: "vault-1",
			Properties: domain.KeyVaultProperties{
				VaultURI: "https://vault-1.vault.azure.net/",
			},
		},
		{
			Name: "vault-2",
			Properties: domain.KeyVaultProperties{
				VaultURI: "https://vault-2.vault.azure.net/",
			},
		},
	}

	if err := cache.SaveVaults("sub-a", vaults); err != nil {
		t.Fatalf("SaveVaults failed: %v", err)
	}

	got := cache.GetVaults("sub-a")
	if len(got) != 2 {
		t.Fatalf("expected 2 vaults, got %d", len(got))
	}

	t.Run("first vault VaultURI preserved", func(t *testing.T) {
		if got[0].Properties.VaultURI != "https://vault-1.vault.azure.net/" {
			t.Errorf("VaultURI = %q, want %q",
				got[0].Properties.VaultURI, "https://vault-1.vault.azure.net/")
		}
	})

	t.Run("second vault VaultURI preserved", func(t *testing.T) {
		if got[1].Properties.VaultURI != "https://vault-2.vault.azure.net/" {
			t.Errorf("VaultURI = %q, want %q",
				got[1].Properties.VaultURI, "https://vault-2.vault.azure.net/")
		}
	})

	// Also verify the round trip survives a second cache load (disk persistence).
	t.Run("survives disk reload", func(t *testing.T) {
		cache2, err := NewCache(10 * time.Minute)
		if err != nil {
			t.Fatalf("second NewCache failed: %v", err)
		}
		got2 := cache2.GetVaults("sub-a")
		if len(got2) != 2 {
			t.Fatalf("expected 2 vaults after reload, got %d", len(got2))
		}
		if got2[0].Properties.VaultURI != "https://vault-1.vault.azure.net/" {
			t.Errorf("after reload VaultURI = %q, want %q",
				got2[0].Properties.VaultURI, "https://vault-1.vault.azure.net/")
		}
	})
}

// TestGetSecretsClient_Caching verifies that calling getSecretsClient with
// the same vaultURI twice reuses the cached client instead of creating a new one.
func TestGetSecretsClient_Caching(t *testing.T) {
	client := &azureClient{
		credential:     &fakeCredential{},
		secretsByVault: make(map[string]*keyVaultSecretsClient),
	}

	uri := "https://testvault.vault.azure.net/"

	// First call should create and cache a new client.
	c1, err := client.getSecretsClient(uri)
	if err != nil {
		t.Fatalf("first getSecretsClient failed: %v", err)
	}
	if c1 == nil {
		t.Fatal("expected non-nil client on first call")
	}

	// Second call with the same URI should return the cached instance.
	c2, err := client.getSecretsClient(uri)
	if err != nil {
		t.Fatalf("second getSecretsClient failed: %v", err)
	}
	if c2 == nil {
		t.Fatal("expected non-nil client on second call")
	}

	// Pointer identity check — same cached object.
	if c1 != c2 {
		t.Error("expected getSecretsClient to return the same cached client for the same URI")
	}

	// Only one entry in the cache map.
	if len(client.secretsByVault) != 1 {
		t.Errorf("expected 1 entry in secretsByVault, got %d", len(client.secretsByVault))
	}

	// A different URI should create a separate client.
	uri2 := "https://othervault.vault.azure.net/"
	c3, err := client.getSecretsClient(uri2)
	if err != nil {
		t.Fatalf("getSecretsClient for different URI failed: %v", err)
	}
	if c3 == nil {
		t.Fatal("expected non-nil client for different URI")
	}

	if c1 == c3 {
		t.Error("expected a different client instance for a different vault URI")
	}

	if len(client.secretsByVault) != 2 {
		t.Errorf("expected 2 entries in secretsByVault after two distinct URIs, got %d",
			len(client.secretsByVault))
	}
}

// TestGetSecretsClient_CachingConcurrent verifies that getSecretsClient is
// safe under concurrent access (the double-checked locking in the implementation).
func TestGetSecretsClient_CachingConcurrent(t *testing.T) {
	client := &azureClient{
		credential:     &fakeCredential{},
		secretsByVault: make(map[string]*keyVaultSecretsClient),
	}

	uri := "https://concurrent.vault.azure.net/"
	var wg sync.WaitGroup

	// Fire off 20 concurrent goroutines all requesting the same vault URI.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sc, err := client.getSecretsClient(uri)
			if err != nil {
				t.Errorf("concurrent getSecretsClient failed: %v", err)
				return
			}
			if sc == nil {
				t.Error("concurrent getSecretsClient returned nil")
			}
		}()
	}
	wg.Wait()

	// Only one entry created despite concurrent calls.
	if len(client.secretsByVault) != 1 {
		t.Errorf("expected 1 entry in secretsByVault after concurrent calls, got %d",
			len(client.secretsByVault))
	}
}