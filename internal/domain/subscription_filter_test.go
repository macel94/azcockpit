package domain

import "testing"

func TestFilterSubscriptionsByName(t *testing.T) {
	subs := []Subscription{
		{ID: "sub-001", DisplayName: "Production Environment", State: "Enabled"},
		{ID: "sub-002", DisplayName: "Staging Environment", State: "Enabled"},
		{ID: "sub-003", DisplayName: "Development", State: "Enabled"},
		{ID: "sub-004", DisplayName: "Pay-As-You-Go", State: "Enabled"},
		{ID: "sub-005", DisplayName: "PRODUCTION backup", State: "Disabled"},
	}

	tests := []struct {
		name     string
		query    string
		input    []Subscription
		wantIDs  []string
	}{
		{
			name:    "empty query returns all",
			query:   "",
			input:   subs,
			wantIDs: []string{"sub-001", "sub-002", "sub-003", "sub-004", "sub-005"},
		},
		{
			name:    "exact match",
			query:   "Development",
			input:   subs,
			wantIDs: []string{"sub-003"},
		},
		{
			name:    "substring match",
			query:   "Environment",
			input:   subs,
			wantIDs: []string{"sub-001", "sub-002"},
		},
		{
			name:    "case insensitive",
			query:   "production",
			input:   subs,
			wantIDs: []string{"sub-001", "sub-005"},
		},
		{
			name:    "mixed case query",
			query:   "PrOdUcTiOn",
			input:   subs,
			wantIDs: []string{"sub-001", "sub-005"},
		},
		{
			name:    "no match returns empty",
			query:   "nonexistent",
			input:   subs,
			wantIDs: []string{},
		},
		{
			name:    "empty subscription list with query",
			query:   "anything",
			input:   []Subscription{},
			wantIDs: []string{},
		},
		{
			name:    "empty subscription list with empty query",
			query:   "",
			input:   []Subscription{},
			wantIDs: []string{},
		},
		{
			name:    "query with spaces",
			query:   "Pay As",
			input:   subs,
			wantIDs: []string{},
		},
		{
			name:    "query with leading spaces trimmed by strings.Contains",
			query:   "Go",
			input:   subs,
			wantIDs: []string{"sub-004"},
		},
		{
			name:    "nil input with query",
			query:   "test",
			input:   nil,
			wantIDs: []string{},
		},
		{
			name:    "nil input with empty query",
			query:   "",
			input:   nil,
			wantIDs: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterSubscriptionsByName(tt.input, tt.query)

			if len(got) != len(tt.wantIDs) {
				t.Errorf("FilterSubscriptionsByName() returned %d results, want %d", len(got), len(tt.wantIDs))
				return
			}

			for i, s := range got {
				if s.ID != tt.wantIDs[i] {
					t.Errorf("result[%d].ID = %q, want %q", i, s.ID, tt.wantIDs[i])
				}
			}
		})
	}
}

func TestFilterSubscriptionsByName_PreservesOrder(t *testing.T) {
	subs := []Subscription{
		{ID: "s1", DisplayName: "Apple"},
		{ID: "s2", DisplayName: "Banana"},
		{ID: "s3", DisplayName: "Cherry"},
	}

	got := FilterSubscriptionsByName(subs, "a")
	if len(got) != 2 {
		t.Fatalf("expected 2 results, got %d", len(got))
	}

	// Apple comes before Banana in input order.
	if got[0].ID != "s1" || got[1].ID != "s2" {
		t.Errorf("order not preserved: got %v", got)
	}
}

func TestFilterSubscriptionsByName_NonOverlapping(t *testing.T) {
	// Ensure returned slice is independent of input.
	subs := []Subscription{
		{ID: "s1", DisplayName: "Original", State: "Enabled"},
	}

	got := FilterSubscriptionsByName(subs, "Original")
	if len(got) != 1 {
		t.Fatal("expected 1 result")
	}

	// Modify the returned slice.
	got[0].ID = "modified"

	// Original should be unchanged.
	if subs[0].ID != "s1" {
		t.Errorf("original modified: got %q, want %q", subs[0].ID, "s1")
	}
}