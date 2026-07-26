package domain

import "testing"

func TestIsHomeTenant(t *testing.T) {
	tests := []struct {
		name     string
		tenant   Tenant
		expected bool
	}{
		{
			name:     "home tenant",
			tenant:   Tenant{TenantCategory: "Home"},
			expected: true,
		},
		{
			name:     "managed tenant",
			tenant:   Tenant{TenantCategory: "ManagedBy"},
			expected: false,
		},
		{
			name:     "pending tenant",
			tenant:   Tenant{TenantCategory: "Pending"},
			expected: false,
		},
		{
			name:     "empty category",
			tenant:   Tenant{TenantCategory: ""},
			expected: false,
		},
		{
			name:     "case sensitive - lowercase home",
			tenant:   Tenant{TenantCategory: "home"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.tenant.IsHomeTenant()
			if got != tt.expected {
				t.Errorf("IsHomeTenant() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestTenantFields(t *testing.T) {
	t1 := Tenant{
		ID:             "tenant-id-1",
		DisplayName:    "My Tenant",
		DefaultDomain:  "mytenant.onmicrosoft.com",
		TenantCategory: "Home",
		CountryCode:    "US",
	}

	if t1.ID != "tenant-id-1" {
		t.Errorf("expected ID 'tenant-id-1', got %q", t1.ID)
	}
	if t1.DisplayName != "My Tenant" {
		t.Errorf("expected DisplayName 'My Tenant', got %q", t1.DisplayName)
	}
	if t1.DefaultDomain != "mytenant.onmicrosoft.com" {
		t.Errorf("expected DefaultDomain 'mytenant.onmicrosoft.com', got %q", t1.DefaultDomain)
	}
	if t1.CountryCode != "US" {
		t.Errorf("expected CountryCode 'US', got %q", t1.CountryCode)
	}
}
