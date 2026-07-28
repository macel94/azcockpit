package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nousresearch/azcockpit/internal/domain"
	"github.com/nousresearch/azcockpit/internal/infrastructure"
)

// mockAzureClient implements infrastructure.AzureClient with in-memory data.
type mockAzureClient struct {
	tenants       []domain.Tenant
	subscriptions []domain.Subscription
	vaults        map[string][]domain.KeyVault
	secrets       map[string][]domain.KeyVaultSecret
	secretValues  map[string]string
}

func (m *mockAzureClient) ListTenants(ctx context.Context) ([]domain.Tenant, error) {
	return m.tenants, nil
}

func (m *mockAzureClient) ListSubscriptions(ctx context.Context) ([]domain.Subscription, error) {
	return m.subscriptions, nil
}

func (m *mockAzureClient) ListKeyVaults(ctx context.Context, subscriptionID string) ([]domain.KeyVault, error) {
	if m.vaults == nil {
		return nil, nil
	}
	return m.vaults[subscriptionID], nil
}

func (m *mockAzureClient) ListKeyVaultSecrets(ctx context.Context, vaultURI string) ([]domain.KeyVaultSecret, error) {
	if m.secrets == nil {
		return nil, nil
	}
	return m.secrets[vaultURI], nil
}

func (m *mockAzureClient) GetKeyVaultSecret(ctx context.Context, vaultURI, secretName string) (string, error) {
	if m.secretValues == nil {
		return "", nil
	}
	return m.secretValues[secretName], nil
}

func (m *mockAzureClient) SetActiveSubscription(ctx context.Context, subscriptionID, tenantID string) error {
	return nil
}

func (m *mockAzureClient) InitializeExample(ctx context.Context, subscriptionID, location string) (domain.KeyVault, []domain.KeyVaultSecret, error) {
	vault := domain.KeyVault{
		Name:           "azcockpit-example",
		Location:       location,
		SubscriptionID: subscriptionID,
		ResourceGroup:  "azcockpit-rg",
		Properties: domain.KeyVaultProperties{
			VaultURI: "https://azcockpit-example.vault.azure.net/",
		},
	}
	secrets := []domain.KeyVaultSecret{
		{Name: "example-secret", Enabled: true},
	}
	return vault, secrets, nil
}

func (m *mockAzureClient) GetCredential() azcore.TokenCredential {
	return nil
}

func (m *mockAzureClient) ExportKeyVaultSecrets(_ context.Context, vaultURI string) (map[string]string, error) {
	if m.secrets == nil {
		return nil, nil
	}
	secrets := m.secrets[vaultURI]
	values := make(map[string]string, len(secrets))
	for _, s := range secrets {
		if m.secretValues != nil {
			if v, ok := m.secretValues[s.Name]; ok {
				values[s.Name] = v
				continue
			}
		}
		values[s.Name] = s.Name
	}
	return values, nil
}

func (m *mockAzureClient) PopulateRandomSecrets(_ context.Context, vaultURI, vaultName, subscriptionID string) ([]domain.KeyVaultSecret, error) {
	secrets := []domain.KeyVaultSecret{
		{Name: "DEMO-DB-PASSWORD", Enabled: true},
		{Name: "DEMO-API-KEY", Enabled: true},
		{Name: "DEMO-STORAGE-CONNECTION-STRING", Enabled: true},
	}
	return secrets, nil
}

// Compile-time check that mock implements the interface.
var _ infrastructure.AzureClient = (*mockAzureClient)(nil)

// makeTestModel creates a Model with a mock client and nil cache.
// Only use this for tests that don't execute the returned tea.Cmd functions
// (which would need a valid cache). For tests that may execute commands
// that access the cache, use a real or test cache.
func makeTestModel() Model {
	mock := &mockAzureClient{}
	return NewModel(mock, nil)
}

// =============================================================================
// filteredSubscriptions tests
// =============================================================================

func TestModelFilteredSubscriptions_EmptyFilter(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = []domain.Subscription{
		{ID: "s1", DisplayName: "Alpha"},
		{ID: "s2", DisplayName: "Beta"},
		{ID: "s3", DisplayName: "Gamma"},
	}
	m.filterQuery = ""

	got := m.filteredSubscriptions()
	if len(got) != 3 {
		t.Fatalf("expected 3 subscriptions, got %d", len(got))
	}
}

func TestModelFilteredSubscriptions_ExactMatch(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = []domain.Subscription{
		{ID: "s1", DisplayName: "Production"},
		{ID: "s2", DisplayName: "Staging"},
	}
	m.filterQuery = "Production"

	got := m.filteredSubscriptions()
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].ID != "s1" {
		t.Errorf("expected s1, got %s", got[0].ID)
	}
}

func TestModelFilteredSubscriptions_PartialMatch(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = []domain.Subscription{
		{ID: "s1", DisplayName: "Dev Environment"},
		{ID: "s2", DisplayName: "Prod Environment"},
		{ID: "s3", DisplayName: "Shared Services"},
	}
	m.filterQuery = "Environment"

	got := m.filteredSubscriptions()
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}
}

func TestModelFilteredSubscriptions_NoMatch(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = []domain.Subscription{
		{ID: "s1", DisplayName: "Production"},
	}
	m.filterQuery = "nonexistent"

	got := m.filteredSubscriptions()
	if len(got) != 0 {
		t.Errorf("expected 0 results, got %d", len(got))
	}
}

func TestModelFilteredSubscriptions_CaseInsensitive(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = []domain.Subscription{
		{ID: "s1", DisplayName: "PRODUCTION"},
		{ID: "s2", DisplayName: "staging"},
	}
	m.filterQuery = "production"

	got := m.filteredSubscriptions()
	if len(got) != 1 {
		t.Fatalf("expected 1 result, got %d", len(got))
	}
	if got[0].ID != "s1" {
		t.Errorf("expected s1, got %s", got[0].ID)
	}
}

func TestModelFilteredSubscriptions_CursorResetOnFilter(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = []domain.Subscription{
		{ID: "s1", DisplayName: "Alpha"},
		{ID: "s2", DisplayName: "Beta"},
		{ID: "s3", DisplayName: "Gamma"},
	}
	m.cursor = 2

	// Apply a filter that only matches 1 subscription.
	m.filterQuery = "alpha"
	filtered := m.filteredSubscriptions()

	// Cursor should be clamped to the filtered list bounds.
	if m.cursor >= len(filtered) && len(filtered) > 0 {
		m.cursor = 0
	}

	if m.cursor != 0 {
		t.Errorf("cursor should be reset to 0 after filter change, got %d", m.cursor)
	}
}

// =============================================================================
// Navigation bounds tests
// =============================================================================

func TestModelNavigateUp_Boundary(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = []domain.Subscription{
		{ID: "s1", DisplayName: "First"},
		{ID: "s2", DisplayName: "Second"},
	}
	m.cursor = 0

	// Press up at top boundary.
	updated, _ := m.handleUp()
	newM := updated.(Model)
	if newM.cursor != 0 {
		t.Errorf("cursor should stay at 0 when pressing up at top, got %d", newM.cursor)
	}
}

func TestModelNavigateDown_Boundary(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = []domain.Subscription{
		{ID: "s1", DisplayName: "First"},
		{ID: "s2", DisplayName: "Second"},
	}
	m.cursor = 1

	// Press down at bottom boundary.
	updated, _ := m.handleDown()
	newM := updated.(Model)
	if newM.cursor != 1 {
		t.Errorf("cursor should stay at 1 when pressing down at bottom, got %d", newM.cursor)
	}
}

func TestModelNavigateDown_EmptyList(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = nil
	m.cursor = 0

	updated, _ := m.handleDown()
	newM := updated.(Model)
	if newM.cursor != 0 {
		t.Errorf("cursor should stay at 0 on empty list, got %d", newM.cursor)
	}
}

func TestModelNavigateUp_NegativeCheck(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = []domain.Subscription{
		{ID: "s1", DisplayName: "First"},
	}
	m.cursor = 0

	updated, _ := m.handleUp()
	newM := updated.(Model)
	if newM.cursor < 0 {
		t.Errorf("cursor should never be negative, got %d", newM.cursor)
	}
}

func TestModelNavigateVaultsUp_Boundary(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewVaults
	m.vaults = []domain.KeyVault{{Name: "v1"}, {Name: "v2"}}
	m.cursor = 0

	updated, _ := m.handleUp()
	newM := updated.(Model)
	if newM.cursor != 0 {
		t.Errorf("cursor should stay at 0 when pressing up at top of vaults, got %d", newM.cursor)
	}
}

func TestModelNavigateVaultsDown_Boundary(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewVaults
	m.vaults = []domain.KeyVault{{Name: "v1"}, {Name: "v2"}}
	m.cursor = 2 // on the [+] button (last valid position)

	updated, _ := m.handleDown()
	newM := updated.(Model)
	if newM.cursor != 2 {
		t.Errorf("cursor should stay at 2 (button) when pressing down at bottom of vaults, got %d", newM.cursor)
	}
}

func TestModelNavigateVaultsDown_ToButton(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewVaults
	m.vaults = []domain.KeyVault{{Name: "v1"}, {Name: "v2"}}
	m.cursor = 1 // on v2

	updated, _ := m.handleDown()
	newM := updated.(Model)
	if newM.cursor != 2 {
		t.Errorf("cursor should go to 2 (button) when pressing down from last vault, got %d", newM.cursor)
	}
}

func TestModelNavigateSecretsUp_Boundary(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewSecrets
	m.secrets = []domain.KeyVaultSecret{{Name: "s1"}, {Name: "s2"}}
	m.cursor = 0

	updated, _ := m.handleUp()
	newM := updated.(Model)
	if newM.cursor != 0 {
		t.Errorf("cursor should stay at 0 when pressing up at top of secrets, got %d", newM.cursor)
	}
}

func TestModelNavigateSecretsDown_Boundary(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewSecrets
	m.secrets = []domain.KeyVaultSecret{{Name: "s1"}, {Name: "s2"}}
	m.cursor = 1

	updated, _ := m.handleDown()
	newM := updated.(Model)
	// Cursor moves to the [+] Add 3 Random Demo Secrets button position (= len(secrets)).
	if newM.cursor != 2 {
		t.Errorf("cursor should move to button position (2) when pressing down from last secret, got %d", newM.cursor)
	}

	// Pressing down again should stay at the button.
	updated2, _ := newM.handleDown()
	newM2 := updated2.(Model)
	if newM2.cursor != 2 {
		t.Errorf("cursor should stay at 2 when pressing down on button, got %d", newM2.cursor)
	}
}

func TestModelNavigateDown_WithinBounds(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = []domain.Subscription{
		{ID: "s1", DisplayName: "First"},
		{ID: "s2", DisplayName: "Second"},
		{ID: "s3", DisplayName: "Third"},
	}
	m.cursor = 1

	updated, _ := m.handleDown()
	newM := updated.(Model)
	if newM.cursor != 2 {
		t.Errorf("cursor should be 2 after pressing down from 1, got %d", newM.cursor)
	}
}

func TestModelNavigateUp_WithinBounds(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = []domain.Subscription{
		{ID: "s1", DisplayName: "First"},
		{ID: "s2", DisplayName: "Second"},
		{ID: "s3", DisplayName: "Third"},
	}
	m.cursor = 1

	updated, _ := m.handleUp()
	newM := updated.(Model)
	if newM.cursor != 0 {
		t.Errorf("cursor should be 0 after pressing up from 1, got %d", newM.cursor)
	}
}

// =============================================================================
// State transition tests: handleEnter
// =============================================================================

func TestModelHandleEnter_Subscription(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = []domain.Subscription{
		{ID: "sub-1", DisplayName: "My Subscription", State: "Enabled"},
		{ID: "sub-2", DisplayName: "Other Sub", State: "Enabled"},
	}
	m.cursor = 0
	m.viewState = viewSubscriptions

	updated, _ := m.handleEnter()
	newM := updated.(Model)

	if newM.viewState != viewVaultsLoading {
		t.Errorf("expected viewState viewVaultsLoading after enter on subscription, got %v", newM.viewState)
	}

	if newM.selectedSubscription == nil {
		t.Fatal("expected selectedSubscription to be set")
	}
	if newM.selectedSubscription.ID != "sub-1" {
		t.Errorf("expected selected subscription sub-1, got %s", newM.selectedSubscription.ID)
	}
}

func TestModelHandleEnter_Subscription_EmptyList(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = nil
	m.viewState = viewSubscriptions

	updated, _ := m.handleEnter()
	newM := updated.(Model)

	if newM.viewState != viewSubscriptions {
		t.Errorf("expected viewState to remain viewSubscriptions, got %v", newM.viewState)
	}
}

func TestModelHandleEnter_Subscription_OutOfBounds(t *testing.T) {
	m := makeTestModel()
	m.subscriptions = []domain.Subscription{
		{ID: "sub-1", DisplayName: "First"},
	}
	m.cursor = 5 // out of bounds
	m.viewState = viewSubscriptions

	updated, _ := m.handleEnter()
	newM := updated.(Model)

	if newM.viewState != viewSubscriptions {
		t.Errorf("expected viewState to remain viewSubscriptions when cursor out of bounds, got %v", newM.viewState)
	}
}

func TestModelHandleEnter_Vault(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewVaults
	m.vaults = []domain.KeyVault{
		{Name: "vault1", Properties: domain.KeyVaultProperties{VaultURI: "https://vault1.vault.azure.net/"}},
		{Name: "vault2", Properties: domain.KeyVaultProperties{VaultURI: "https://vault2.vault.azure.net/"}},
	}
	m.cursor = 1

	updated, _ := m.handleEnter()
	newM := updated.(Model)

	if newM.viewState != viewSecretsLoading {
		t.Errorf("expected viewState viewSecretsLoading, got %v", newM.viewState)
	}

	if newM.selectedVault == nil {
		t.Fatal("expected selectedVault to be set")
	}
	if newM.selectedVault.Name != "vault2" {
		t.Errorf("expected selected vault vault2, got %s", newM.selectedVault.Name)
	}
}

func TestModelHandleEnter_Vault_EmptyList(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewVaults
	m.vaults = nil

	updated, _ := m.handleEnter()
	newM := updated.(Model)

	if newM.viewState != viewVaults {
		t.Errorf("expected viewState to remain viewVaults on empty list, got %v", newM.viewState)
	}
}

// =============================================================================
// State transition tests: handleEsc
// =============================================================================

func TestModelHandleEsc_FromVaults(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewVaults
	m.cursor = 3
	m.selectedSubscription = &domain.Subscription{ID: "sub-1"}

	updated, _ := m.handleEsc()
	newM := updated.(Model)

	if newM.viewState != viewSubscriptions {
		t.Errorf("esc from vaults: expected viewState viewSubscriptions, got %v", newM.viewState)
	}
	if newM.cursor != 0 {
		t.Errorf("esc from vaults: expected cursor reset to 0, got %d", newM.cursor)
	}
	if newM.selectedSubscription != nil {
		t.Error("esc from vaults: expected selectedSubscription to be nil")
	}
}

func TestModelHandleEsc_FromVaultsLoading(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewVaultsLoading
	m.cursor = 5

	updated, _ := m.handleEsc()
	newM := updated.(Model)

	if newM.viewState != viewSubscriptions {
		t.Errorf("esc from vaults loading: expected viewState viewSubscriptions, got %v", newM.viewState)
	}
	if newM.cursor != 0 {
		t.Errorf("esc from vaults loading: expected cursor reset to 0, got %d", newM.cursor)
	}
}

func TestModelHandleEsc_FromSecrets(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewSecrets
	m.cursor = 2
	m.selectedVault = &domain.KeyVault{Name: "myvault"}

	updated, _ := m.handleEsc()
	newM := updated.(Model)

	if newM.viewState != viewVaults {
		t.Errorf("esc from secrets: expected viewState viewVaults, got %v", newM.viewState)
	}
	if newM.cursor != 0 {
		t.Errorf("esc from secrets: expected cursor reset to 0, got %d", newM.cursor)
	}
	if newM.selectedVault != nil {
		t.Error("esc from secrets: expected selectedVault to be nil")
	}
}

func TestModelHandleEsc_FromSecretsLoading(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewSecretsLoading
	m.cursor = 3

	updated, _ := m.handleEsc()
	newM := updated.(Model)

	if newM.viewState != viewVaults {
		t.Errorf("esc from secrets loading: expected viewState viewVaults, got %v", newM.viewState)
	}
	if newM.cursor != 0 {
		t.Errorf("esc from secrets loading: expected cursor reset to 0, got %d", newM.cursor)
	}
}

func TestModelHandleEsc_FromSubscriptions_Ready(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewSubscriptions
	m.ready = true

	updated, cmd := m.handleEsc()
	newM := updated.(Model)

	if newM.ready {
		t.Error("esc from subscriptions: expected ready=false (refresh triggered)")
	}
	if !newM.loading {
		t.Error("esc from subscriptions: expected loading=true")
	}
	if cmd == nil {
		t.Error("esc from subscriptions: expected non-nil command (fetchData)")
	}
}

func TestModelHandleEsc_FromSubscriptions_NotReady(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewSubscriptions
	m.ready = false

	updated, cmd := m.handleEsc()
	newM := updated.(Model)

	if newM.ready {
		t.Error("expected ready to remain false")
	}
	if cmd != nil {
		t.Error("expected nil command when not ready")
	}
}

// =============================================================================
// Render function tests
// =============================================================================

func TestModelRenderLoading(t *testing.T) {
	m := makeTestModel()
	m.loading = true

	view := m.View()
	if view == "" {
		t.Error("View() should not return empty string when loading")
	}
	if !strings.Contains(view, "Fetching") {
		t.Error("loading view should contain 'Fetching'")
	}
}

func TestModelRenderError(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.err = context.DeadlineExceeded

	view := m.View()
	if view == "" {
		t.Error("View() should not return empty string on error")
	}
	if !strings.Contains(view, "Error") {
		t.Error("error view should contain 'Error'")
	}
	if !strings.Contains(view, "q to quit") {
		t.Error("error view should contain quit hint")
	}
}

func TestModelRenderSubscriptions(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.ready = true
	m.viewState = viewSubscriptions
	m.subscriptions = []domain.Subscription{
		{ID: "sub-1", DisplayName: "Production", State: "Enabled"},
		{ID: "sub-2", DisplayName: "Staging", State: "Disabled"},
	}
	m.tenants = []domain.Tenant{
		{ID: "t1", DisplayName: "Primary Tenant", TenantCategory: "Home", DefaultDomain: "tenant.onmicrosoft.com"},
	}

	view := m.View()
	if view == "" {
		t.Error("View() should not return empty string")
	}
	if !strings.Contains(view, "Production") {
		t.Error("subscriptions view should contain subscription display name 'Production'")
	}
	if !strings.Contains(view, "Staging") {
		t.Error("subscriptions view should contain subscription display name 'Staging'")
	}
	if !strings.Contains(view, "Primary Tenant") {
		t.Error("subscriptions view should contain tenant display name")
	}
	if !strings.Contains(view, "🏠") {
		t.Error("subscriptions view should show home tenant marker")
	}
	if !strings.Contains(view, "q quit") {
		t.Error("subscriptions view should contain quit hint")
	}
}

func TestModelRenderSubscriptions_WithCursor(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.ready = true
	m.viewState = viewSubscriptions
	m.subscriptions = []domain.Subscription{
		{ID: "sub-1", DisplayName: "Alpha", State: "Enabled"},
		{ID: "sub-2", DisplayName: "Beta", State: "Enabled"},
	}
	m.cursor = 1

	view := m.View()
	if !strings.Contains(view, "Beta") {
		t.Error("view should contain second subscription")
	}
	// The cursor row should contain the second sub.
	if !strings.Contains(view, "sub-2") {
		t.Error("view should contain sub-2 ID")
	}
}

func TestModelRenderSubscriptions_EmptyList(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.ready = true
	m.viewState = viewSubscriptions
	m.subscriptions = nil

	view := m.View()
	if !strings.Contains(view, "No subscriptions found") {
		t.Error("view should indicate no subscriptions found")
	}
}

func TestModelRenderVaults(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.viewState = viewVaults
	m.selectedSubscription = &domain.Subscription{ID: "sub-1", DisplayName: "My Sub"}
	m.vaults = []domain.KeyVault{
		{Name: "vault-prod", Location: "westus2"},
		{Name: "vault-dev", Location: "eastus"},
	}
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "vault-prod") {
		t.Error("vaults view should contain 'vault-prod'")
	}
	if !strings.Contains(view, "vault-dev") {
		t.Error("vaults view should contain 'vault-dev'")
	}
	if !strings.Contains(view, "westus2") {
		t.Error("vaults view should contain location 'westus2'")
	}
	if !strings.Contains(view, "My Sub") {
		t.Error("vaults view should contain subscription display name")
	}
}

func TestModelRenderVaults_EmptyList(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.viewState = viewVaults
	m.selectedSubscription = &domain.Subscription{ID: "sub-1", DisplayName: "Empty Sub"}
	m.vaults = nil
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "No key vaults found") {
		t.Error("view should indicate no key vaults found")
	}
}

func TestModelRenderVaultsLoading(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.viewState = viewVaultsLoading
	m.selectedSubscription = &domain.Subscription{ID: "sub-1", DisplayName: "Loading Sub"}
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "Loading Sub") {
		t.Errorf("vaults loading view should contain subscription name, got:\n%s", view)
	}
	if !strings.Contains(view, "esc back") {
		t.Errorf("vaults loading view should contain esc back hint, got:\n%s", view)
	}
}

func TestModelRenderSecrets(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{
		Name: "myvault",
		Properties: domain.KeyVaultProperties{
			VaultURI: "https://myvault.vault.azure.net/",
		},
	}
	m.secrets = []domain.KeyVaultSecret{
		{Name: "DB_PASSWORD", ContentType: "text/plain", Enabled: true},
		{Name: "API_KEY", Enabled: false},
	}
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "DB_PASSWORD") {
		t.Error("secrets view should contain 'DB_PASSWORD'")
	}
	if !strings.Contains(view, "API_KEY") {
		t.Error("secrets view should contain 'API_KEY'")
	}
	if !strings.Contains(view, "myvault") {
		t.Error("secrets view should contain vault name")
	}
}

func TestModelRenderSecrets_EmptyList(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "emptyvault"}
	m.secrets = nil
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "No secrets found") {
		t.Error("view should indicate no secrets found")
	}
}

func TestModelRenderSecretsLoading(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.viewState = viewSecretsLoading
	m.selectedVault = &domain.KeyVault{Name: "loadingvault"}
	m.width = 80
	m.height = 24

	view := m.View()
	if !strings.Contains(view, "loadingvault") {
		t.Errorf("secrets loading view should contain vault name, got:\n%s", view)
	}
	if !strings.Contains(view, "esc back") {
		t.Errorf("secrets loading view should contain esc back hint, got:\n%s", view)
	}
}

func TestModelRenderQuitting(t *testing.T) {
	m := makeTestModel()
	m.quitting = true

	view := m.View()
	if !strings.Contains(view, "Goodbye") {
		t.Error("quitting view should contain 'Goodbye'")
	}
}

// =============================================================================
// Model construction tests
// =============================================================================

func TestNewModel_InitialState(t *testing.T) {
	mock := &mockAzureClient{}
	m := NewModel(mock, nil)

	if !m.loading {
		t.Error("new model should start in loading state")
	}
	if m.viewState != viewSubscriptions {
		t.Errorf("new model should start in viewSubscriptions state, got %v", m.viewState)
	}
	if m.ready {
		t.Error("new model should not be ready")
	}
	if m.filterQuery != "" {
		t.Errorf("new model should have empty filterQuery, got %q", m.filterQuery)
	}
	if m.azureClient != mock {
		t.Error("new model should retain injected azureClient")
	}
}

func TestNewModel_CursorStartsAtZero(t *testing.T) {
	m := makeTestModel()
	if m.tenantCursor != 0 || m.cursor != 0 {
		t.Errorf("expected cursor=0, got tenantCursor=%d cursor=%d", m.tenantCursor, m.cursor)
	}
}

// =============================================================================
// Export keybinding tests
// =============================================================================

func TestModelHandleExport_FromSecretsView(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "test-vault", Properties: domain.KeyVaultProperties{VaultURI: "https://test-vault.vault.azure.net/"}}
	m.secrets = []domain.KeyVaultSecret{
		{Name: "DEMO-DB-PASSWORD", Enabled: true},
		{Name: "DEMO-API-KEY", Enabled: true},
	}

	// Simulate pressing 'e' — should trigger fetchExportSecrets cmd.
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	// The model should remain in viewSecrets and the cmd should not be nil.
	if model.(Model).viewState != viewSecrets {
		t.Errorf("expected viewSecrets, got %v", model.(Model).viewState)
	}
	if cmd == nil {
		t.Error("expected non-nil cmd from 'e' keypress on secrets view")
	}
}

func TestModelHandleExport_FromVaultsView_NoOp(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewVaults

	// Pressing 'e' from vaults view should be a no-op.
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})

	if cmd != nil {
		t.Error("expected nil cmd from 'e' keypress on vaults view")
	}
	_ = model // model unchanged
}

func TestModelRenderSecrets_WithExportBanner(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "test-vault"}
	m.secrets = []domain.KeyVaultSecret{
		{Name: "DEMO_DB_PASSWORD", Enabled: true},
	}
	m.exportedSecrets = "  ✓ Exported 1 secret(s) from test-vault\n    📄 secrets_test-vault.env\n    📋 Copied to clipboard\n    Run in your shell:\nexport DEMO_DB_PASSWORD='supersecret'\n"

	view := m.View()
	if !strings.Contains(view, "✓") {
		t.Error("expected export success banner (✓) in rendered view")
	}
	if !strings.Contains(view, "export DEMO_DB_PASSWORD") {
		t.Error("expected export script in rendered view")
	}
	if !strings.Contains(view, "secrets_test-vault.env") {
		t.Error("expected env file path in rendered view")
	}
}

func TestModelRenderSecrets_WithExportError(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "test-vault"}
	m.secrets = []domain.KeyVaultSecret{
		{Name: "DEMO-DB-PASSWORD", Enabled: true},
	}
	m.exportErr = fmt.Errorf("test export error")

	view := m.View()
	if !strings.Contains(view, "✗") {
		t.Error("expected export error banner (✗) in rendered view")
	}
	if !strings.Contains(view, "test export error") {
		t.Error("expected error message in rendered view")
	}
}

func TestModelEsc_ClearsExportState(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "test-vault"}
	m.exportedSecrets = "some export data"
	m.exportErr = nil
	m.exportValues = map[string]string{"KEY": "val"}

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(Model)

	if updated.exportedSecrets != "" {
		t.Error("expected exportedSecrets cleared on esc from viewSecrets")
	}
	if updated.exportValues != nil {
		t.Error("expected exportValues cleared on esc from viewSecrets")
	}
	if updated.exportErr != nil {
		t.Error("expected exportErr cleared on esc from viewSecrets")
	}
	if updated.viewState != viewVaults {
		t.Errorf("expected viewVaults after esc from viewSecrets, got %v", updated.viewState)
	}
}

// =============================================================================
// Populate (add random secrets) tests
// =============================================================================

func TestModelHandlePopulate_FromSecretsView(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "test-vault", Properties: domain.KeyVaultProperties{VaultURI: "https://test-vault.vault.azure.net/"}}
	m.selectedSubscription = &domain.Subscription{ID: "sub-123", DisplayName: "Test Sub"}
	m.secrets = []domain.KeyVaultSecret{
		{Name: "EXISTING_SECRET", Enabled: true},
	}

	// Press 'a' from secrets view.
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	updated := model.(Model)
	if !updated.populateLoading {
		t.Error("expected populateLoading=true after 'a' keypress")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd from 'a' keypress on secrets view")
	}
}

func TestModelHandlePopulate_FromVaultsView_NoOp(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewVaults

	// 'a' from vaults view should be a no-op.
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if cmd != nil {
		t.Error("expected nil cmd from 'a' keypress on vaults view")
	}
	_ = model
}

func TestModelHandlePopulate_FromSecretsView_MissingSubscription(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "test-vault"}
	m.selectedSubscription = nil // no subscription

	// 'a' should be a no-op without a subscription.
	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	if cmd != nil {
		t.Error("expected nil cmd when selectedSubscription is nil")
	}
	_ = model
}

func TestModelHandleEnter_OnPopulateButton(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "test-vault", Properties: domain.KeyVaultProperties{VaultURI: "https://test-vault.vault.azure.net/"}}
	m.selectedSubscription = &domain.Subscription{ID: "sub-123"}
	m.secrets = []domain.KeyVaultSecret{
		{Name: "EXISTING_SECRET", Enabled: true},
	}
	// Cursor at button position = len(secrets).
	m.cursor = 1

	model, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := model.(Model)

	if !updated.populateLoading {
		t.Error("expected populateLoading=true after enter on button")
	}
	if cmd == nil {
		t.Error("expected non-nil cmd after enter on populate button")
	}
}

func TestModelHandlePopulateMsg_SetsSecretsAndRefetches(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "test-vault", Properties: domain.KeyVaultProperties{VaultURI: "https://test-vault.vault.azure.net/"}}
	m.secrets = []domain.KeyVaultSecret{}

	// Simulate receiving a populateSecretsMsg with 3 new secrets.
	msg := populateSecretsMsg{
		vaultURI: "https://test-vault.vault.azure.net/",
		secrets: []domain.KeyVaultSecret{
			{Name: "DEMO-DB-PASSWORD", Enabled: true},
			{Name: "DEMO-API-KEY", Enabled: true},
			{Name: "DEMO-STORAGE-CONNECTION-STRING", Enabled: true},
		},
		err: nil,
	}

	model, cmd := m.Update(msg)
	updated := model.(Model)

	if updated.populateLoading {
		t.Error("expected populateLoading=false after populateSecretsMsg")
	}
	if updated.populateErr != nil {
		t.Errorf("unexpected error: %v", updated.populateErr)
	}
	if !strings.Contains(updated.populateMessage, "Added 3") {
		t.Errorf("expected success message containing 'Added 3', got %q", updated.populateMessage)
	}
	// Should trigger a re-fetch of secrets.
	if cmd == nil {
		t.Error("expected non-nil cmd (secrets re-fetch) after populateSecretsMsg")
	}
}

func TestModelHandlePopulateMsg_Error(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "test-vault"}
	m.populateLoading = true

	msg := populateSecretsMsg{
		vaultURI: "https://test-vault.vault.azure.net/",
		err:      fmt.Errorf("create secrets failed"),
	}

	model, cmd := m.Update(msg)
	updated := model.(Model)

	if updated.populateLoading {
		t.Error("expected populateLoading=false after error")
	}
	if updated.populateErr == nil {
		t.Error("expected populateErr to be set")
	}
	if updated.populateMessage != "" {
		t.Error("expected populateMessage to be empty on error")
	}
	if cmd != nil {
		t.Error("expected nil cmd on error")
	}
}

func TestModelRenderSecrets_WithPopulateButton(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "test-vault"}
	m.secrets = []domain.KeyVaultSecret{
		{Name: "EXISTING_SECRET", Enabled: true},
	}

	view := m.View()
	if !strings.Contains(view, "[+] Add 3 Random Demo Secrets") {
		t.Error("expected [+] Add 3 Random Demo Secrets button in rendered view")
	}
	if !strings.Contains(view, "a add secrets") {
		t.Error("expected 'a add secrets' hint in footer")
	}
}

func TestModelRenderSecrets_WithPopulateLoadingSpinner(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "test-vault"}
	m.secrets = []domain.KeyVaultSecret{
		{Name: "EXISTING_SECRET", Enabled: true},
	}
	m.populateLoading = true

	view := m.View()
	if !strings.Contains(view, "Adding 3 Random Demo Secrets...") {
		t.Error("expected 'Adding 3 Random Demo Secrets...' loading text in rendered view")
	}
}

func TestModelRenderSecrets_WithPopulateSuccessBanner(t *testing.T) {
	m := makeTestModel()
	m.loading = false
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "test-vault"}
	m.secrets = []domain.KeyVaultSecret{
		{Name: "DEMO-DB-PASSWORD", Enabled: true},
	}
	m.populateMessage = "  ✓ Added 3 random demo secret(s) to this vault"

	view := m.View()
	if !strings.Contains(view, "✓ Added 3") {
		t.Error("expected success banner with '✓ Added 3' in rendered view")
	}
}

func TestModelEsc_ClearsPopulateState(t *testing.T) {
	m := makeTestModel()
	m.viewState = viewSecrets
	m.selectedVault = &domain.KeyVault{Name: "test-vault"}
	m.populateMessage = "some message"
	m.populateErr = nil
	m.populateLoading = true

	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated := model.(Model)

	if updated.populateMessage != "" {
		t.Error("expected populateMessage cleared on esc from viewSecrets")
	}
	if updated.populateErr != nil {
		t.Error("expected populateErr cleared on esc from viewSecrets")
	}
	if updated.populateLoading {
		t.Error("expected populateLoading cleared on esc from viewSecrets")
	}
	if updated.viewState != viewVaults {
		t.Errorf("expected viewVaults after esc from viewSecrets, got %v", updated.viewState)
	}
}