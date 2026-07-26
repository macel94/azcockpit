# AzCockpit — Azure Context & Vault Explorer

[![Go Version](https://img.shields.io/badge/Go-1.26+-00ADD8?logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)
[![Tests](https://img.shields.io/badge/tests-40%20passing-brightgreen)](https://github.com/macel94/azcockpit/actions)

A lightning-fast TUI for navigating Azure tenants, subscriptions, and Key Vaults — built in Go with [Bubble Tea](https://github.com/charmbracelet/bubbletea).

> **Switch Azure contexts instantly. Pull Key Vault secrets into `.env` files. No Azure Portal needed.**

---

## Why AzCockpit?

| Problem | AzCockpit Solution |
|---------|-------------------|
| `az cli` is Python-based and sluggish | Single Go binary, sub-second startup |
| `az account set` requires remembering subscription GUIDs | Visual TUI with arrow-key navigation |
| Copying secrets from Azure Portal -> `.env` is tedious | Browse Key Vaults, select secrets, export in one flow |
| `azd` is project-centric, not developer-centric | AzCockpit is **your** cockpit — global view across all tenants |
| No offline/cached access | Local JSON cache at `~/.azcockpit/cache.json` (5-min TTL) |

## Features

- **DefaultAzureCredential** — works with `az login`, service principals, and managed identities
- **Tenant Browser** — navigate all Azure AD tenants your identity can see
- **Subscription List** — active/disabled state indicators, spending limits
- **Key Vault Explorer** — browse vaults per subscription, list secrets
- **`.env` Export** — select secrets and append to local `.env` files
- **Shell Integration** — `az account set` wrapper, `export` script generation
- **Cache-First** — 5-minute TTL local cache for sub-second subsequent launches
- **Concurrent API Calls** — goroutine-based async fetches keep the UI responsive

## Quick Start

### Prerequisites
- **Go 1.26+**
- **Azure authentication** (any of):
  - `az login` (Azure CLI)
  - Environment variables: `AZURE_CLIENT_ID`, `AZURE_CLIENT_SECRET`, `AZURE_TENANT_ID`
  - Managed Identity (on Azure VMs/App Service)

### Install

```bash
git clone https://github.com/macel94/azcockpit.git
cd azcockpit
go build -o azcockpit ./cmd/azcockpit/
./azcockpit
```

Or run directly:

```bash
go run ./cmd/azcockpit/
```

## Navigation

| Key | Action |
|-----|--------|
| Up / Down | Move cursor |
| Enter | Select / drill down |
| Esc | Go back / refresh |
| q | Quit |

### Views

1. **Subscriptions** — browse all subscriptions across tenants. Active (green dot) vs disabled (grey dot).
2. **Vaults** — Key Vaults in the selected subscription. Name, location, resource group.
3. **Secrets** — secret names in the selected vault. Select to export to `.env`.

## Shell Integration

```bash
# Switch Azure context from the TUI selection
eval $(azcockpit export)

# Or wrap az account set directly
azcockpit set
```

## Architecture

```
UI Layer (Bubble Tea)
  model.go  styles.go
  arrow-key cursor, view state machine

Infrastructure Layer
  azure_client.go  cache.go  shell.go  keyvault_secrets.go
  Azure SDK, JSON cache, os/exec

Domain Layer
  tenant.go  subscription.go  keyvault.go
  Pure Go models — zero SDK imports
```

## Testing

```bash
go test ./... -v -count=1
```

**40 tests** across 2 packages covering domain models, cache TTL/expiry, concurrent reads, ARM ID parsing, and shell helpers.

## Roadmap

- [x] **Phase 1:** Domain models, ARM client, local cache, basic TUI
- [x] **Phase 2:** Key Vault secrets, full TUI navigation, shell integration, 40 tests
- [ ] **Phase 3:** Microsoft Graph enrichment (tenant display names), multi-select, search/filter
- [ ] **Phase 4:** Cross-platform binaries (goreleaser), Homebrew tap

## Contributing

See [AGENTS.md](AGENTS.md) for architecture details, design decisions, and development conventions.

## License

MIT © [macel94](https://github.com/macel94)
