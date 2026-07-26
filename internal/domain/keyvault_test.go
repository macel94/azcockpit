package domain

import (
	"testing"
	"time"
)

func TestKeyVaultSecret_WithTimeFields(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	later := now.Add(24 * time.Hour)

	secret := KeyVaultSecret{
		ID:          "https://myvault.vault.azure.net/secrets/mysecret/version1",
		Name:        "mysecret",
		ContentType: "text/plain",
		Enabled:     true,
		NotBefore:   &now,
		Expires:     &later,
		Tags: map[string]string{
			"env": "production",
		},
	}

	if secret.ID == "" {
		t.Error("expected non-empty ID")
	}
	if secret.Name != "mysecret" {
		t.Errorf("expected Name 'mysecret', got %q", secret.Name)
	}
	if secret.ContentType != "text/plain" {
		t.Errorf("expected ContentType 'text/plain', got %q", secret.ContentType)
	}
	if !secret.Enabled {
		t.Error("expected Enabled to be true")
	}
	if secret.NotBefore == nil {
		t.Error("expected NotBefore to be set")
	} else if !secret.NotBefore.Equal(now) {
		t.Errorf("expected NotBefore %v, got %v", now, secret.NotBefore)
	}
	if secret.Expires == nil {
		t.Error("expected Expires to be set")
	} else if !secret.Expires.Equal(later) {
		t.Errorf("expected Expires %v, got %v", later, secret.Expires)
	}
	if secret.Tags["env"] != "production" {
		t.Errorf("expected tag 'env' = 'production', got %q", secret.Tags["env"])
	}
}

func TestKeyVaultSecret_NilTimeFields(t *testing.T) {
	secret := KeyVaultSecret{
		ID:      "https://myvault.vault.azure.net/secrets/nilsecret/version1",
		Name:    "nilsecret",
		Enabled: false,
	}

	if secret.NotBefore != nil {
		t.Error("expected NotBefore to be nil")
	}
	if secret.Expires != nil {
		t.Error("expected Expires to be nil")
	}
}

func TestKeyVault(t *testing.T) {
	kv := KeyVault{
		ID:             "/subscriptions/sub-id/resourceGroups/my-rg/providers/Microsoft.KeyVault/vaults/myvault",
		Name:           "myvault",
		Location:       "westus2",
		ResourceGroup:  "my-rg",
		SubscriptionID: "sub-id",
		Properties: KeyVaultProperties{
			SKU: KeyVaultSKU{
				Family: "A",
				Name:   "standard",
			},
			TenantID:                "tenant-id",
			VaultURI:                "https://myvault.vault.azure.net/",
			EnableSoftDelete:        true,
			EnablePurgeProtection:   false,
			EnableRBACAuthorization: true,
		},
		Tags: map[string]string{
			"cost-center": "12345",
		},
	}

	if kv.Name != "myvault" {
		t.Errorf("expected Name 'myvault', got %q", kv.Name)
	}
	if kv.Properties.SKU.Name != "standard" {
		t.Errorf("expected SKU 'standard', got %q", kv.Properties.SKU.Name)
	}
	if !kv.Properties.EnableSoftDelete {
		t.Error("expected EnableSoftDelete to be true")
	}
	if kv.Properties.EnablePurgeProtection {
		t.Error("expected EnablePurgeProtection to be false")
	}
	if !kv.Properties.EnableRBACAuthorization {
		t.Error("expected EnableRBACAuthorization to be true")
	}

	// Test with premium SKU.
	kvPremium := KeyVault{
		Name: "premium-vault",
		Properties: KeyVaultProperties{
			SKU: KeyVaultSKU{
				Family: "A",
				Name:   "premium",
			},
		},
	}
	if kvPremium.Properties.SKU.Name != "premium" {
		t.Errorf("expected SKU 'premium', got %q", kvPremium.Properties.SKU.Name)
	}
}
