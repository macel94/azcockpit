package infrastructure

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/nousresearch/azcockpit/internal/domain"
)

func TestNewCache_CreatesDir(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cache, err := NewCache(10 * time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}
	if cache == nil {
		t.Fatal("expected non-nil cache")
	}

	// Verify the .azcockpit directory was created.
	cacheDir := filepath.Join(tmpDir, ".azcockpit")
	if info, err := os.Stat(cacheDir); os.IsNotExist(err) {
		t.Errorf("expected cache directory %s to exist", cacheDir)
	} else if !info.IsDir() {
		t.Errorf("expected %s to be a directory", cacheDir)
	}
}

func TestNewCache_LoadsExisting(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Create a cache and save data.
	cache1, err := NewCache(10 * time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	tenant := domain.Tenant{ID: "t1", DisplayName: "Tenant One", TenantCategory: "Home"}
	if err := cache1.SaveTenants([]domain.Tenant{tenant}); err != nil {
		t.Fatalf("SaveTenants failed: %v", err)
	}

	// Create a new cache instance that loads the existing file.
	cache2, err := NewCache(10 * time.Minute)
	if err != nil {
		t.Fatalf("second NewCache failed: %v", err)
	}

	tenants := cache2.GetTenants()
	if len(tenants) != 1 {
		t.Fatalf("expected 1 tenant loaded, got %d", len(tenants))
	}
	if tenants[0].ID != "t1" {
		t.Errorf("expected tenant ID 't1', got %q", tenants[0].ID)
	}
}

func TestNewCache_HandlesEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cache, err := NewCache(10 * time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	// With no saved data, get methods should return empty slices.
	tenants := cache.GetTenants()
	if tenants == nil || len(tenants) != 0 {
		t.Errorf("expected empty tenants, got %v", tenants)
	}

	subs := cache.GetSubscriptions()
	if subs == nil || len(subs) != 0 {
		t.Errorf("expected empty subscriptions, got %v", subs)
	}

	vaults := cache.GetVaults("any-sub")
	if vaults == nil || len(vaults) != 0 {
		t.Errorf("expected empty vaults, got %v", vaults)
	}
}

func TestCache_IsValid_Empty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cache, err := NewCache(10 * time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	// Fresh cache with no data should not be valid.
	if cache.IsValid() {
		t.Error("expected IsValid() to be false for empty cache")
	}
}

func TestCache_IsValid_PopulatedNotExpired(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cache, err := NewCache(10 * time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	// Save tenants to populate the cache.
	if err := cache.SaveTenants([]domain.Tenant{{ID: "t1"}}); err != nil {
		t.Fatalf("SaveTenants failed: %v", err)
	}

	if !cache.IsValid() {
		t.Error("expected IsValid() to be true for populated, non-expired cache")
	}
}

func TestCache_IsValid_Expired(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	// Use a very short TTL (1 nanosecond) so it expires immediately.
	cache, err := NewCache(1 * time.Nanosecond)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	// Save data with current time as CachedAt.
	if err := cache.SaveTenants([]domain.Tenant{{ID: "t1"}}); err != nil {
		t.Fatalf("SaveTenants failed: %v", err)
	}

	// Sleep just a bit to ensure expiration.
	time.Sleep(10 * time.Millisecond)

	if cache.IsValid() {
		t.Error("expected IsValid() to be false for expired cache")
	}
}

func TestCache_IsValid_SubscriptionsOnly(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cache, err := NewCache(10 * time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	// Populate only subscriptions.
	if err := cache.SaveSubscriptions([]domain.Subscription{{ID: "s1"}}); err != nil {
		t.Fatalf("SaveSubscriptions failed: %v", err)
	}

	if !cache.IsValid() {
		t.Error("expected IsValid() to be true when only subscriptions are populated")
	}
}

func TestCache_SaveGetTenants(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cache, err := NewCache(10 * time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	tenants := []domain.Tenant{
		{ID: "t1", DisplayName: "Tenant One"},
		{ID: "t2", DisplayName: "Tenant Two"},
	}

	if err := cache.SaveTenants(tenants); err != nil {
		t.Fatalf("SaveTenants failed: %v", err)
	}

	got := cache.GetTenants()
	if len(got) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(got))
	}
	if got[0].ID != "t1" || got[1].ID != "t2" {
		t.Errorf("unexpected tenant data: %+v", got)
	}
}

func TestCache_SaveGetSubscriptions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cache, err := NewCache(10 * time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	subs := []domain.Subscription{
		{ID: "s1", DisplayName: "Sub One"},
		{ID: "s2", DisplayName: "Sub Two"},
	}

	if err := cache.SaveSubscriptions(subs); err != nil {
		t.Fatalf("SaveSubscriptions failed: %v", err)
	}

	got := cache.GetSubscriptions()
	if len(got) != 2 {
		t.Fatalf("expected 2 subscriptions, got %d", len(got))
	}
	if got[0].ID != "s1" || got[1].ID != "s2" {
		t.Errorf("unexpected subscription data: %+v", got)
	}
}

func TestCache_SaveGetVaults(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cache, err := NewCache(10 * time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	vaults := []domain.KeyVault{
		{Name: "vault1", SubscriptionID: "sub-a"},
		{Name: "vault2", SubscriptionID: "sub-a"},
	}

	if err := cache.SaveVaults("sub-a", vaults); err != nil {
		t.Fatalf("SaveVaults failed: %v", err)
	}

	got := cache.GetVaults("sub-a")
	if len(got) != 2 {
		t.Fatalf("expected 2 vaults, got %d", len(got))
	}
	if got[0].Name != "vault1" {
		t.Errorf("expected vault1, got %s", got[0].Name)
	}

	// Non-existent subscription should return empty.
	empty := cache.GetVaults("non-existent")
	if len(empty) != 0 {
		t.Errorf("expected 0 vaults for non-existent sub, got %d", len(empty))
	}
}

func TestCache_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cache, err := NewCache(10 * time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	// Save some data.
	if err := cache.SaveTenants([]domain.Tenant{{ID: "t1"}}); err != nil {
		t.Fatalf("SaveTenants failed: %v", err)
	}

	// Clear it.
	if err := cache.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	// After clear, should be empty and invalid.
	if cache.IsValid() {
		t.Error("expected IsValid() to be false after Clear()")
	}

	tenants := cache.GetTenants()
	if tenants == nil || len(tenants) != 0 {
		t.Errorf("expected empty tenants after clear, got %v", tenants)
	}
}

func TestCache_ConcurrentReads(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	cache, err := NewCache(10 * time.Minute)
	if err != nil {
		t.Fatalf("NewCache failed: %v", err)
	}

	// Populate with data.
	tenants := []domain.Tenant{{ID: "t1", DisplayName: "T1"}}
	subs := []domain.Subscription{{ID: "s1", DisplayName: "S1"}}
	vaults := []domain.KeyVault{{Name: "v1", SubscriptionID: "s1"}}

	if err := cache.SaveTenants(tenants); err != nil {
		t.Fatalf("SaveTenants failed: %v", err)
	}
	if err := cache.SaveSubscriptions(subs); err != nil {
		t.Fatalf("SaveSubscriptions failed: %v", err)
	}
	if err := cache.SaveVaults("s1", vaults); err != nil {
		t.Fatalf("SaveVaults failed: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Mix of reads.
			t := cache.GetTenants()
			if len(t) != 1 || t[0].ID != "t1" {
				// Don't call t.Error from goroutine — just panic.
				panic("unexpected tenants")
			}

			s := cache.GetSubscriptions()
			if len(s) != 1 || s[0].ID != "s1" {
				panic("unexpected subscriptions")
			}

			v := cache.GetVaults("s1")
			if len(v) != 1 || v[0].Name != "v1" {
				panic("unexpected vaults")
			}

			if !cache.IsValid() {
				panic("cache should be valid")
			}
		}()
	}

	wg.Wait()
}
