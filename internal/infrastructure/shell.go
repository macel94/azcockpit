package infrastructure

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/nousresearch/azcockpit/internal/domain"
)

// GenerateExportScript returns a shell-sourcable snippet that exports
// AZURE_SUBSCRIPTION_ID and AZURE_SUBSCRIPTION_NAME as environment variables.
func GenerateExportScript(sub domain.Subscription) string {
	var b strings.Builder
	fmt.Fprintf(&b, "export AZURE_SUBSCRIPTION_ID=%q\n", sub.ID)
	fmt.Fprintf(&b, "export AZURE_SUBSCRIPTION_NAME=%q\n", sub.DisplayName)
	return b.String()
}

// SetAzAccount switches the active Azure CLI subscription by running
// `az account set --subscription <id>`. Returns an error if `az` is not
// found in PATH or the command fails.
func SetAzAccount(sub domain.Subscription) error {
	azPath, err := exec.LookPath("az")
	if err != nil {
		return fmt.Errorf("azure CLI (az) not found in PATH: %w", err)
	}

	cmd := exec.Command(azPath, "account", "set", "--subscription", sub.ID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set az account: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// SetAzAccountWithTenant switches the active Azure CLI subscription and
// tenant by running `az account set --subscription <sub.ID> --tenant <sub.TenantID>`.
// If sub.TenantID is empty, the --tenant flag is omitted for backward
// compatibility.
func SetAzAccountWithTenant(sub domain.Subscription) error {
	azPath, err := exec.LookPath("az")
	if err != nil {
		return fmt.Errorf("azure CLI (az) not found in PATH: %w", err)
	}

	args := []string{"account", "set", "--subscription", sub.ID}
	if sub.TenantID != "" {
		args = append(args, "--tenant", sub.TenantID)
	}

	cmd := exec.Command(azPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to set az account: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// WriteEnvFile appends KEY=VALUE entries to the given .env file path.
// Entries whose keys already exist in the file are skipped.
// Creates the file (and parent directories) if they don't exist.
// The `secrets` slice provides KeyVaultSecret entries (written as
// <secret.Name>=<secret.Name> placeholder — callers must fetch real values).
// The `values` map provides additional KEY=VALUE overrides.
// Returns the number of new lines written.
func WriteEnvFile(path string, secrets []domain.KeyVaultSecret, values map[string]string) (int, error) {
	// Create parent directories if needed.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Read existing keys (case-insensitive matching).
	existing := make(map[string]bool)
	f, err := os.Open(path)
	if err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			// Skip empty lines and comments.
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if idx := strings.Index(line, "="); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				existing[strings.ToUpper(key)] = true
			}
		}
		f.Close()
	}

	// Open file for appending (create if not exists).
	out, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return 0, fmt.Errorf("failed to open env file for writing: %w", err)
	}
	defer out.Close()

	written := 0

	// Write secrets (placeholder values).
	for _, s := range secrets {
		key := s.Name
		if existing[strings.ToUpper(key)] {
			continue
		}
		fmt.Fprintf(out, "%s=%s\n", key, key) // placeholder: value same as key
		existing[strings.ToUpper(key)] = true
		written++
	}

	// Write additional values.
	for key, val := range values {
		if existing[strings.ToUpper(key)] {
			continue
		}
		fmt.Fprintf(out, "%s=%s\n", key, val)
		existing[strings.ToUpper(key)] = true
		written++
	}

	return written, nil
}

// ExportToClipboard pipes the given text to the system clipboard.
// On macOS it uses pbcopy, on Linux xclip (fallback xsel), on Windows clip.exe.
func ExportToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Prefer xclip, fall back to xsel.
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard tool found: install xclip or xsel")
		}
	case "windows":
		cmd = exec.Command("clip.exe")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to copy to clipboard: %s: %w", strings.TrimSpace(string(output)), err)
	}

	return nil
}

// GenerateExportScriptForSecrets produces a shell-sourcable snippet that
// exports each secret as an environment variable, e.g.:
//
//	export DEMO_DB_PASSWORD='supersecret'
//	export DEMO_API_KEY='abc123'
//
// The envValues map is name→value (e.g. from ExportKeyVaultSecrets).
// The vaultName (if non-empty) is included as a comment in the output.
func GenerateExportScriptForSecrets(vaultName string, envValues map[string]string) string {
	var b strings.Builder

	if vaultName != "" {
		fmt.Fprintf(&b, "# Secrets from Azure Key Vault: %s\n", vaultName)
	}

	for name, val := range envValues {
		// Escape single quotes for safe shell export.
		escaped := strings.ReplaceAll(val, "'", "'\\''")
		fmt.Fprintf(&b, "export %s='%s'\n", name, escaped)
	}

	return b.String()
}
