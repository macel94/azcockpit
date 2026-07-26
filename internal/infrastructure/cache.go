package infrastructure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/nousresearch/azcockpit/internal/domain"
)

const (
	cacheDir      = ".azcockpit"
	cacheFileName = "cache.json"
)

// CacheEntry wraps cached data with metadata for TTL checks.
type CacheEntry struct {
	// Tenants holds the cached tenant list.
	Tenants []domain.Tenant `json:"tenants"`

	// Subscriptions holds the cached subscription list.
	Subscriptions []domain.Subscription `json:"subscriptions"`

	// VaultsBySubscription maps subscription ID → cached vaults.
	VaultsBySubscription map[string][]domain.KeyVault `json:"vaultsBySubscription"`

	// CachedAt is the timestamp when this cache was written.
	CachedAt time.Time `json:"cachedAt"`
}

// Cache provides thread-safe read/write access to the local JSON cache file.
// The cache lives at ~/.azcockpit/cache.json and is used to avoid repeated
// Azure API calls on subsequent TUI launches.
type Cache struct {
	mu    sync.RWMutex
	path  string
	entry CacheEntry
	ttl   time.Duration
}

// NewCache creates or loads the cache from disk.
// ttl is the maximum age before a cache entry is considered stale.
func NewCache(ttl time.Duration) (*Cache, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	cacheDirPath := filepath.Join(homeDir, cacheDir)
	if err := os.MkdirAll(cacheDirPath, 0o700); err != nil {
		return nil, fmt.Errorf("failed to create cache directory %s: %w", cacheDirPath, err)
	}

	c := &Cache{
		path: filepath.Join(cacheDirPath, cacheFileName),
		ttl:  ttl,
	}

	// Load existing cache if present and valid.
	if err := c.load(); err != nil {
		// Fresh start — no cache yet. This is not an error.
		c.entry = CacheEntry{
			VaultsBySubscription: make(map[string][]domain.KeyVault),
		}
	}

	return c, nil
}

// IsValid returns true if the cache is populated and not expired.
func (c *Cache) IsValid() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.entry.CachedAt.IsZero() {
		return false
	}

	if time.Since(c.entry.CachedAt) > c.ttl {
		return false
	}

	// Must have at least tenants or subscriptions.
	return len(c.entry.Tenants) > 0 || len(c.entry.Subscriptions) > 0
}

// GetTenants returns the cached tenants.
func (c *Cache) GetTenants() []domain.Tenant {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]domain.Tenant, len(c.entry.Tenants))
	copy(result, c.entry.Tenants)
	return result
}

// GetSubscriptions returns the cached subscriptions.
func (c *Cache) GetSubscriptions() []domain.Subscription {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]domain.Subscription, len(c.entry.Subscriptions))
	copy(result, c.entry.Subscriptions)
	return result
}

// GetVaults returns the cached vaults for a given subscription.
func (c *Cache) GetVaults(subscriptionID string) []domain.KeyVault {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.entry.VaultsBySubscription == nil {
		return nil
	}

	vaults := c.entry.VaultsBySubscription[subscriptionID]
	result := make([]domain.KeyVault, len(vaults))
	copy(result, vaults)
	return result
}

// SaveTenants persists the tenant list to cache.
func (c *Cache) SaveTenants(tenants []domain.Tenant) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entry.Tenants = tenants
	c.entry.CachedAt = time.Now()
	return c.persist()
}

// SaveSubscriptions persists the subscription list to cache.
func (c *Cache) SaveSubscriptions(subscriptions []domain.Subscription) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entry.Subscriptions = subscriptions
	c.entry.CachedAt = time.Now()
	return c.persist()
}

// SaveVaults persists vaults for a specific subscription to cache.
func (c *Cache) SaveVaults(subscriptionID string, vaults []domain.KeyVault) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.entry.VaultsBySubscription == nil {
		c.entry.VaultsBySubscription = make(map[string][]domain.KeyVault)
	}
	c.entry.VaultsBySubscription[subscriptionID] = vaults
	c.entry.CachedAt = time.Now()
	return c.persist()
}

// Clear removes the entire cache file.
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entry = CacheEntry{
		VaultsBySubscription: make(map[string][]domain.KeyVault),
	}

	if err := os.Remove(c.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}

	return nil
}

// load reads the cache from disk.
func (c *Cache) load() error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return err // file doesn't exist or can't be read
	}

	var entry CacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return fmt.Errorf("failed to parse cache file: %w", err)
	}

	c.entry = entry
	if c.entry.VaultsBySubscription == nil {
		c.entry.VaultsBySubscription = make(map[string][]domain.KeyVault)
	}

	return nil
}

// persist writes the cache to disk atomically.
func (c *Cache) persist() error {
	data, err := json.MarshalIndent(c.entry, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	// Write to a temp file first, then rename for atomicity.
	tmpPath := c.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("failed to write cache: %w", err)
	}

	if err := os.Rename(tmpPath, c.path); err != nil {
		return fmt.Errorf("failed to commit cache: %w", err)
	}

	return nil
}
