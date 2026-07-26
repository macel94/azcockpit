package domain

import "testing"

func TestIsActive(t *testing.T) {
	tests := []struct {
		name         string
		subscription Subscription
		expected     bool
	}{
		{
			name:         "enabled state",
			subscription: Subscription{State: "Enabled"},
			expected:     true,
		},
		{
			name:         "disabled state",
			subscription: Subscription{State: "Disabled"},
			expected:     false,
		},
		{
			name:         "warned state",
			subscription: Subscription{State: "Warned"},
			expected:     false,
		},
		{
			name:         "past due state",
			subscription: Subscription{State: "PastDue"},
			expected:     false,
		},
		{
			name:         "deleted state",
			subscription: Subscription{State: "Deleted"},
			expected:     false,
		},
		{
			name:         "empty state",
			subscription: Subscription{State: ""},
			expected:     false,
		},
		{
			name:         "unknown state",
			subscription: Subscription{State: "SomeOtherState"},
			expected:     false,
		},
		{
			name:         "case sensitive - enabled",
			subscription: Subscription{State: "enabled"},
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.subscription.IsActive()
			if got != tt.expected {
				t.Errorf("IsActive() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSubscriptionFields(t *testing.T) {
	sub := Subscription{
		ID:          "sub-guid",
		DisplayName: "Pay-As-You-Go",
		TenantID:    "tenant-guid",
		State:       "Enabled",
		SubscriptionPolicies: SubscriptionPolicies{
			LocationPlacementID: "eastus",
			QuotaID:             "quota-id",
			SpendingLimit:       "Off",
		},
	}

	if sub.ID != "sub-guid" {
		t.Errorf("expected ID 'sub-guid', got %q", sub.ID)
	}
	if sub.TenantID != "tenant-guid" {
		t.Errorf("expected TenantID 'tenant-guid', got %q", sub.TenantID)
	}
	if sub.SubscriptionPolicies.SpendingLimit != "Off" {
		t.Errorf("expected SpendingLimit 'Off', got %q", sub.SubscriptionPolicies.SpendingLimit)
	}
}
