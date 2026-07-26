// AzCockpit — Azure Context & Vault Explorer.
// A lightning-fast TUI for navigating Azure tenants, subscriptions,
// and Key Vaults, built with Go and Bubble Tea.
package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/nousresearch/azcockpit/internal/infrastructure"
	"github.com/nousresearch/azcockpit/internal/ui"
)

const (
	// cacheTTL is how long cached Azure data remains valid.
	// After this period, the TUI will re-fetch from the API.
	cacheTTL = 5 * time.Minute
)

func main() {
	// 1. Initialize the local cache (~/.azcockpit/cache.json).
	cache, err := infrastructure.NewCache(cacheTTL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize cache: %v\n", err)
		os.Exit(1)
	}

	// 2. Create the Azure client using DefaultAzureCredential.
	// This reads from: AZURE_CLIENT_ID/SECRET/TENANT_ID env vars,
	// managed identity (if on Azure), or `az login` session.
	azureClient, err := infrastructure.NewAzureClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create Azure client: %v\n", err)
		fmt.Fprintf(os.Stderr, "\nMake sure you're authenticated. Try:\n")
		fmt.Fprintf(os.Stderr, "  az login\n")
		fmt.Fprintf(os.Stderr, "Or set AZURE_CLIENT_ID, AZURE_CLIENT_SECRET, and AZURE_TENANT_ID.\n")
		os.Exit(1)
	}

	// 3. Create the Bubble Tea model with dependency injection.
	model := ui.NewModel(azureClient, cache)

	// 4. Run the TUI.
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "AzCockpit crashed: %v\n", err)
		os.Exit(1)
	}
}
