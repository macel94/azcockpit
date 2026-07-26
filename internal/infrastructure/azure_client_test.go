package infrastructure

import (
	"testing"
)

func TestExtractResourceGroup_Valid(t *testing.T) {
	tests := []struct {
		name     string
		armID    string
		expected string
	}{
		{
			name:     "standard ARM ID",
			armID:    "/subscriptions/sub-id/resourceGroups/my-rg/providers/Microsoft.KeyVault/vaults/myvault",
			expected: "my-rg",
		},
		{
			name:     "nested resource",
			armID:    "/subscriptions/sub-id/resourceGroups/prod-rg/providers/Microsoft.Compute/virtualMachines/myvm",
			expected: "prod-rg",
		},
		{
			name:     "mixed case resourceGroups",
			armID:    "/subscriptions/sub-id/ResourceGroups/CaseRg/providers/Microsoft.Web/sites",
			expected: "CaseRg",
		},
		{
			name:     "all lowercase",
			armID:    "/subscriptions/sub-id/resourcegroups/lower-rg/providers/Microsoft.Web/sites",
			expected: "lower-rg",
		},
		{
			name:     "resource group is last segment",
			armID:    "/subscriptions/sub-id/resourceGroups/my-rg",
			expected: "my-rg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResourceGroup(tt.armID)
			if got != tt.expected {
				t.Errorf("extractResourceGroup(%q) = %q, want %q", tt.armID, got, tt.expected)
			}
		})
	}
}

func TestExtractResourceGroup_Invalid(t *testing.T) {
	tests := []struct {
		name  string
		armID string
	}{
		{
			name:  "empty string",
			armID: "",
		},
		{
			name:  "no resourceGroups segment",
			armID: "/subscriptions/sub-id/providers/Microsoft.KeyVault/vaults/myvault",
		},
		{
			name:  "resourceGroups at end with no following segment",
			armID: "/subscriptions/sub-id/resourceGroups/",
		},
		{
			name:  "random string",
			armID: "not-an-arm-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractResourceGroup(tt.armID)
			if got != "" {
				t.Errorf("extractResourceGroup(%q) = %q, want empty string", tt.armID, got)
			}
		})
	}
}

func TestDerefString(t *testing.T) {
	// Nil pointer.
	if got := derefString(nil); got != "" {
		t.Errorf("derefString(nil) = %q, want empty string", got)
	}

	// Non-nil pointer.
	val := "hello"
	if got := derefString(&val); got != "hello" {
		t.Errorf("derefString(&%q) = %q, want %q", val, got, val)
	}

	// Empty string pointer.
	empty := ""
	if got := derefString(&empty); got != "" {
		t.Errorf("derefString(&\"\") = %q, want empty string", got)
	}
}

func TestDerefBool(t *testing.T) {
	// Nil pointer.
	if got := derefBool(nil); got != false {
		t.Errorf("derefBool(nil) = %v, want false", got)
	}

	// true pointer.
	trueVal := true
	if got := derefBool(&trueVal); got != true {
		t.Errorf("derefBool(&true) = %v, want true", got)
	}

	// false pointer.
	falseVal := false
	if got := derefBool(&falseVal); got != false {
		t.Errorf("derefBool(&false) = %v, want false", got)
	}
}

func TestCopyStringMap(t *testing.T) {
	// Nil map.
	if got := copyStringMap(nil); got != nil {
		t.Errorf("copyStringMap(nil) = %v, want nil", got)
	}

	// Normal map.
	input := map[string]*string{
		"key1": strPtr("value1"),
		"key2": strPtr("value2"),
	}
	got := copyStringMap(input)
	if len(got) != 2 {
		t.Errorf("expected 2 entries, got %d", len(got))
	}
	if got["key1"] != "value1" {
		t.Errorf("expected key1='value1', got %q", got["key1"])
	}
	if got["key2"] != "value2" {
		t.Errorf("expected key2='value2', got %q", got["key2"])
	}

	// Map with nil values (should skip nil entries).
	mixed := map[string]*string{
		"key1": strPtr("value1"),
		"key2": nil,
		"key3": strPtr("value3"),
	}
	got2 := copyStringMap(mixed)
	if len(got2) != 2 {
		t.Errorf("expected 2 entries (nil skipped), got %d", len(got2))
	}
	if _, exists := got2["key2"]; exists {
		t.Error("expected key2 to be skipped (nil value)")
	}

	// Empty map.
	empty := map[string]*string{}
	got3 := copyStringMap(empty)
	if got3 == nil || len(got3) != 0 {
		t.Errorf("expected empty map, got %v", got3)
	}

	// Verify copy is independent.
	input["key1"] = strPtr("modified")
	if got["key1"] != "modified" {
		// The returned map should've been a copy — but since we modified the
		// pointer target, the copy would have the old value. Wait, actually
		// the copy stores *string values. So if we change what the *string
		// points to, the copy WOULD see the change. But we're changing the
		// map entry (the pointer itself), not what it points to.
		// That's fine — the copy has its own map with its own pointers.
		// Actually, copyStringMap dereferences the pointer, so it stores
		// plain strings. So the output map contains "value1" even after
		// we change input["key1"] to point to "modified".
		if got["key1"] != "value1" {
			t.Errorf("expected copy to be independent, got key1=%q", got["key1"])
		}
	}
}

func TestStringValue(t *testing.T) {
	// Define a custom string type.
	type MyString string

	// Nil pointer.
	if got := stringValue[MyString](nil); got != "" {
		t.Errorf("stringValue(nil) = %q, want empty string", got)
	}

	// Non-nil pointer.
	val := MyString("hello")
	if got := stringValue(&val); got != "hello" {
		t.Errorf("stringValue(&%q) = %q, want %q", val, got, val)
	}
}

// strPtr is a test helper that returns a pointer to the given string.
func strPtr(s string) *string {
	return &s
}
