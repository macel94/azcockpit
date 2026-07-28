//go:build e2e_azure

// Run with: go test -tags e2e_azure ./internal/infrastructure -run TestE2E_Populate -v
package infrastructure

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nousresearch/azcockpit/internal/domain"
)

func TestE2E_PopulateRandomSecrets_RealAzure(t *testing.T) {
	// Skip unless explicitly opted in via env var, to keep CI safe.
	if os.Getenv("AZCOCKPIT_E2E") == "" {
		t.Skip("set AZCOCKPIT_E2E=1 to run the real Azure end-to-end test")
	}

	client, err := NewAzureClient()
	if err != nil {
		t.Fatalf("auth failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// Use the existing demo vault the user already provisioned.
	vaultName := os.Getenv("AZCOCKPIT_E2E_VAULT")
	subscriptionID := os.Getenv("AZCOCKPIT_E2E_SUB")
	if vaultName == "" || subscriptionID == "" {
		t.Skip("set AZCOCKPIT_E2E_VAULT and AZCOCKPIT_E2E_SUB to run")
	}
	vaultURI := "https://" + vaultName + ".vault.azure.net/"

	// 1. Verify the vault exists.
	vaults, err := client.ListKeyVaults(ctx, subscriptionID)
	if err != nil {
		t.Fatalf("list vaults: %v", err)
	}
	var found *domain.KeyVault
	for i, v := range vaults {
		if v.Name == vaultName {
			found = &vaults[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("vault %q not found in subscription", vaultName)
	}
	t.Logf("✓ Vault found: %s (%s)", found.Name, found.Properties.VaultURI)

	// 2. Populate 3 random secrets.
	secrets, err := client.PopulateRandomSecrets(ctx, vaultURI, vaultName, subscriptionID)
	if err != nil {
		t.Fatalf("PopulateRandomSecrets: %v", err)
	}
	if len(secrets) != 3 {
		t.Errorf("expected 3 secrets, got %d", len(secrets))
	}
	t.Logf("✓ Created %d secrets:", len(secrets))
	for _, s := range secrets {
		t.Logf("  - %s", s.Name)
	}

	// 3. List secrets in the vault (confirms they're really there).
	listed, err := client.ListKeyVaultSecrets(ctx, vaultURI)
	if err != nil {
		t.Fatalf("ListKeyVaultSecrets: %v", err)
	}
	hasDemo := map[string]bool{}
	for _, s := range listed {
		hasDemo[s.Name] = true
		t.Logf("  listed: %s", s.Name)
	}
	if !hasDemo["DEMO-DB-PASSWORD"] || !hasDemo["DEMO-API-KEY"] || !hasDemo["DEMO-STORAGE-CONNECTION-STRING"] {
		t.Errorf("expected the 3 demo secrets in the vault, got: %v", hasDemo)
	}

	// 4. Export and verify the script converts hyphens to underscores.
	values, err := client.ExportKeyVaultSecrets(ctx, vaultURI)
	if err != nil {
		t.Fatalf("ExportKeyVaultSecrets: %v", err)
	}
	script := GenerateExportScriptForSecrets(vaultName, values)
	fmt.Println("=== Export script ===")
	fmt.Println(script)
	fmt.Println("====================")

	must := []string{
		"export DEMO_DB_PASSWORD='",
		"export DEMO_API_KEY='",
		"export DEMO_STORAGE_CONNECTION_STRING='",
	}
	for _, want := range must {
		if !strings.Contains(script, want) {
			t.Errorf("expected %q in script, got:\n%s", want, script)
		}
	}
	t.Logf("✓ Export script correctly converts hyphens to underscores")
}
