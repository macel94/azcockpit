# Repository Guidelines for AzCockpit

## Project Overview
AzCockpit is a lightning-fast TUI (Terminal User Interface) built in Go for navigating Azure tenants, subscriptions, and Key Vaults. It acts as a local cache and navigation layer over the Azure API, allowing developers to switch CLI contexts instantly and pull Key Vault secrets into local `.env` files.

## Architecture
- **Language:** Go 1.26+
- **Module:** `github.com/macel94/azcockpit`
- **UI Framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss)
- **Azure SDK:** `github.com/Azure/azure-sdk-for-go/sdk/azidentity` + `armsubscription` + `armkeyvault` + `azsecrets`
- **Design:** Lightweight DDD. Domain (`internal/domain/`) knows nothing about Azure SDKs or Bubble Tea. Infrastructure (`internal/infrastructure/`) is the only layer importing external packages. UI (`internal/ui/`) depends only on the `AzureClient` interface.

## Development Setup

```bash
git clone https://github.com/macel94/azcockpit.git
cd azcockpit
go build ./...
go run ./cmd/azcockpit/
go test ./... -v -count=1
```

## Project Structure

```
cmd/azcockpit/main.go              # Entry point — wires DI, runs TUI
internal/
  domain/                          # Pure Go models (zero SDK imports)
    tenant.go        Tenant        # Azure AD tenant
    subscription.go  Subscription  # Azure subscription
    keyvault.go      KeyVault      # Key Vault + Secret models
  infrastructure/                  # Azure SDK, cache, shell integration
    azure_client.go  AzureClient interface + ARM implementation
    keyvault_secrets.go            # azsecrets data-plane client
    cache.go         Thread-safe JSON cache (~/.azcockpit/cache.json)
    shell.go         az account set, .env export, clipboard
  ui/                              # Bubble Tea TUI
    model.go         Model (Init/Update/View) + navigation state machine
    styles.go        Azure-branded lipgloss theme
```

## Key Design Decisions

### Cache-First Architecture
`Model.fetchData()` checks `Cache.IsValid()` before hitting Azure APIs. Subsequent launches load from `~/.azcockpit/cache.json` with a 5-minute TTL. Vaults are cached per-subscription. Cache writes are atomic (tmp + rename).

### Async API Calls
All Azure API calls run in goroutines via `tea.Cmd` to keep the UI responsive. Results are delivered to the Bubble Tea event loop through typed messages (`fetchResultMsg`, `vaultsMsg`, `secretsMsg`). A 30-second timeout via `context.WithTimeout` prevents hanging.

### Interface-Based DI
`AzureClient` is an interface. The concrete `azureClient` implements it using the Azure SDK. The UI layer receives `AzureClient` via constructor injection, making it testable with mocks.

### TenantID Limitations
The ARM tenants API only returns `TenantID` and a resource ID — not display name, domain, or category. The ARM subscriptions API does not include a `TenantID` field. Tenant-scoped subscription filtering requires the Microsoft Graph API (future enhancement).

## Shell Integration

```bash
eval $(azcockpit export)    # Set AZURE_SUBSCRIPTION_ID in current shell
azcockpit set               # Run az account set directly
```

## Testing

```bash
go test ./... -v -count=1 -race
```

- **Domain tests:** Model behavior (`IsActive`, `IsHomeTenant`, struct fields)
- **Infrastructure tests:** Cache TTL/expiry, concurrent reads, ARM ID parsing, shell helpers
- **Go race detector enabled** for cache concurrency tests

## Conventions

- **Error handling:** Always `fmt.Errorf("context: %w", err)` to wrap
- **Nil safety:** SDK types use pointers extensively — always deref with helper functions (`derefString`, `derefBool`)
- **Pager pattern:** Azure list operations use the SDK pager (`pager.More()` / `pager.NextPage(ctx)`)
- **No external test frameworks:** Standard library `testing` only. `t.Run()` for subtests.
- **Commits:** Conventional commit format (`feat:`, `fix:`, `test:`, `docs:`)

## Authentication

AzCockpit uses `DefaultAzureCredential` which chains:
1. Environment variables (`AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`)
2. Managed Identity (when running on Azure)
3. Azure CLI (`az login`)

No API keys are stored by AzCockpit itself.

## Roadmap

- [x] Phase 1: Domain models, ARM client, cache, basic TUI
- [x] Phase 2: Key Vault secrets, TUI navigation, shell integration, tests
- [x] Phase 3: Initialize Example vault with random secrets, export as env vars
- [x] Phase 3b: In-app button to add 3 random demo secrets to any existing vault
- [ ] Phase 4: Microsoft Graph enrichment (tenant display names), multi-select secrets, search/filter
- [ ] Phase 5: Cross-platform release binaries (goreleaser), Homebrew tap
