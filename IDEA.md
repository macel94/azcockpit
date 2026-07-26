# AzCockpit — Azure Context & Vault Explorer

Lightning-fast, globally-scoped TUI in Go for navigating tenants, subscriptions, and Key Vaults.

## Core Goals
1. **Instant Context Switching** — Navigate and switch active Azure Tenants/Subscriptions instantly. Aggressively cache data locally (~/.azcockpit/cache.json) to eliminate network latency.
2. **Key Vault to `.env`** — Browse Key Vaults visually, select secrets, and generate/append to local `.env` files.
3. **Developer Ergonomics** — Sub-second startup. Single statically linked binary.
4. **Auth** — `DefaultAzureCredential` from Azure SDK for Go.

## Non-Goals
- No resource provisioning (leave to Bicep/Terraform)
- No project-centric deployment (leave to `azd`)
- No generic Resource Graph queries beyond Subscriptions + Key Vaults

## Tech Stack
- **Language:** Go (latest)
- **UI:** `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss`
- **Azure SDK:** `github.com/Azure/azure-sdk-for-go/sdk/azidentity`, Resource Manager, Key Vault SDKs
- **Arch:** Lightweight DDD — Core Domain (Identity, Vaults) / Infrastructure (Azure API, local JSON cache) / Presentation (Bubble Tea models)

## Architecture
- Azure API calls in goroutines via `tea.Cmd` to avoid blocking UI thread
- Local cache at `~/.azcockpit/cache.json`
- Shell integration: output `export AZURE_SUBSCRIPTION_ID=...` script for `eval`, or wrap `az account set`

## Directory Structure
```
cmd/azcockpit/main.go
internal/domain/      — Tenant, Subscription, KeyVault models
internal/infrastructure/ — Azure client, cache layer
internal/ui/          — Bubble Tea models, styles
go.mod
go.sum
```
