package infrastructure

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nousresearch/azcockpit/internal/domain"
)

func TestGenerateExportScript(t *testing.T) {
	sub := domain.Subscription{
		ID:          "00000000-0000-0000-0000-000000000001",
		DisplayName: "Test Subscription",
	}

	script := GenerateExportScript(sub)

	if !strings.Contains(script, `export AZURE_SUBSCRIPTION_ID="00000000-0000-0000-0000-000000000001"`) {
		t.Errorf("expected subscription ID in script, got:\n%s", script)
	}

	if !strings.Contains(script, `export AZURE_SUBSCRIPTION_NAME="Test Subscription"`) {
		t.Errorf("expected subscription name in script, got:\n%s", script)
	}

	// Verify the script contains exactly two export lines.
	lines := strings.Split(strings.TrimSpace(script), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 export lines, got %d", len(lines))
	}

	for _, line := range lines {
		if !strings.HasPrefix(line, "export ") {
			t.Errorf("expected line to start with 'export ', got: %s", line)
		}
	}
}

func TestGenerateExportScript_SpecialCharacters(t *testing.T) {
	sub := domain.Subscription{
		ID:          "sub-id",
		DisplayName: "My \"Special\" Subscription",
	}

	script := GenerateExportScript(sub)

	// The %q format escapes quotes, so the display name should contain escaped quotes.
	if !strings.Contains(script, `My \"Special\" Subscription`) {
		t.Errorf("expected escaped quotes in display name, got:\n%s", script)
	}
}

func TestWriteEnvFile_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	n, err := WriteEnvFile(envPath, nil, map[string]string{"FOO": "bar"})
	if err != nil {
		t.Fatalf("WriteEnvFile failed: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 entry written, got %d", n)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read env file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "FOO=bar") {
		t.Errorf("expected FOO=bar in file, got:\n%s", content)
	}
}

func TestWriteEnvFile_WritesEntries(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	secrets := []domain.KeyVaultSecret{
		{Name: "DB_PASSWORD"},
		{Name: "API_KEY"},
	}

	values := map[string]string{
		"APP_ENV": "production",
	}

	n, err := WriteEnvFile(envPath, secrets, values)
	if err != nil {
		t.Fatalf("WriteEnvFile failed: %v", err)
	}
	if n != 3 {
		t.Errorf("expected 3 entries written, got %d", n)
	}

	data, err := os.ReadFile(envPath)
	if err != nil {
		t.Fatalf("failed to read env file: %v", err)
	}

	content := string(data)
	for _, expected := range []string{"DB_PASSWORD=DB_PASSWORD", "API_KEY=API_KEY", "APP_ENV=production"} {
		if !strings.Contains(content, expected) {
			t.Errorf("expected %q in file, got:\n%s", expected, content)
		}
	}
}

func TestWriteEnvFile_SkipsDuplicates(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	// Write initial entry.
	n, err := WriteEnvFile(envPath, nil, map[string]string{"FOO": "bar"})
	if err != nil {
		t.Fatalf("first WriteEnvFile failed: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 entry on first write, got %d", n)
	}

	// Try writing the same key again.
	n, err = WriteEnvFile(envPath, nil, map[string]string{"FOO": "baz"})
	if err != nil {
		t.Fatalf("second WriteEnvFile failed: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 entries on second write (duplicate), got %d", n)
	}
}

func TestWriteEnvFile_CaseInsensitiveDuplicate(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	// Write initial entry.
	_, err := WriteEnvFile(envPath, nil, map[string]string{"MY_VAR": "value1"})
	if err != nil {
		t.Fatalf("first WriteEnvFile failed: %v", err)
	}

	// Try writing the same key with different case.
	n, err := WriteEnvFile(envPath, nil, map[string]string{"my_var": "value2"})
	if err != nil {
		t.Fatalf("second WriteEnvFile failed: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 entries (case-insensitive duplicate), got %d", n)
	}
}

func TestWriteEnvFile_ReturnsCorrectCount(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	// Empty write.
	n, err := WriteEnvFile(envPath, nil, nil)
	if err != nil {
		t.Fatalf("WriteEnvFile with no entries failed: %v", err)
	}
	if n != 0 {
		t.Errorf("expected 0 entries, got %d", n)
	}

	// Write 2 values.
	n, err = WriteEnvFile(envPath, nil, map[string]string{"A": "1", "B": "2"})
	if err != nil {
		t.Fatalf("WriteEnvFile failed: %v", err)
	}
	if n != 2 {
		t.Errorf("expected 2 entries, got %d", n)
	}
}

func TestWriteEnvFile_NestedDirectories(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, "nested", "path", ".env")

	n, err := WriteEnvFile(envPath, nil, map[string]string{"X": "y"})
	if err != nil {
		t.Fatalf("WriteEnvFile with nested dirs failed: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 entry, got %d", n)
	}

	// Verify file exists at the right path.
	if _, err := os.Stat(envPath); os.IsNotExist(err) {
		t.Errorf("expected file at %s to exist", envPath)
	}
}

func TestWriteEnvFile_SkipsCommentsAndEmptyLines(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")

	// Pre-populate with comments and an existing key.
	os.WriteFile(envPath, []byte("# This is a comment\nEXISTING=value\n\n"), 0o644)

	n, err := WriteEnvFile(envPath, nil, map[string]string{"NEW_KEY": "new_value", "EXISTING": "ignored"})
	if err != nil {
		t.Fatalf("WriteEnvFile failed: %v", err)
	}
	if n != 1 {
		t.Errorf("expected 1 new entry (only NEW_KEY), got %d", n)
	}

	data, _ := os.ReadFile(envPath)
	content := string(data)
	if !strings.Contains(content, "NEW_KEY=new_value") {
		t.Errorf("expected NEW_KEY in file, got:\n%s", content)
	}
	if strings.Count(content, "EXISTING=value") != 1 {
		t.Errorf("expected EXISTING to appear exactly once")
	}
}
