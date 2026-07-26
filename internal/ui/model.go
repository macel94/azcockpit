package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/nousresearch/azcockpit/internal/domain"
	"github.com/nousresearch/azcockpit/internal/infrastructure"
)

// loadingTimeout is how long we wait before giving up on Azure API calls.
const loadingTimeout = 30 * time.Second

// --- View State ---

// viewState represents which screen the user is currently looking at.
type viewState int

const (
	viewSubscriptions viewState = iota
	viewVaultsLoading
	viewVaults
	viewSecretsLoading
	viewSecrets
)

// focusPane represents which pane has keyboard focus on the subscriptions screen.
type focusPane int

const (
	focusTenants focusPane = iota
	focusSubscriptions
)

// --- Messages ---

// fetchResultMsg carries the result of an async Azure fetch back to the UI.
type fetchResultMsg struct {
	tenants       []domain.Tenant
	subscriptions []domain.Subscription
	err           error
}

// vaultsMsg carries the result of a ListKeyVaults call back to the UI.
type vaultsMsg struct {
	subscriptionID string
	vaults         []domain.KeyVault
	err            error
}

// secretsMsg carries the result of a ListKeyVaultSecrets call back to the UI.
type secretsMsg struct {
	vaultURI string
	secrets  []domain.KeyVaultSecret
	err      error
}

// errMsg carries an error to the UI.
type errMsg struct {
	err error
}

// --- Model ---

// Model is the top-level Bubble Tea model for AzCockpit.
type Model struct {
	// spinner shows while data is being fetched.
	spinner spinner.Model

	// azureClient is the infrastructure layer for Azure API calls.
	azureClient infrastructure.AzureClient

	// cache is the local JSON cache for fast subsequent loads.
	cache *infrastructure.Cache

	// Data
	tenants       []domain.Tenant
	subscriptions []domain.Subscription

	// Navigation
	cursor               int // cursor for the subscriptions list
	tenantCursor         int // cursor for the tenants list
	paneFocus            focusPane
	activeTenantID       string // tenant selected as active context (visual)
	selectedSubscription *domain.Subscription
	selectedVault        *domain.KeyVault
	vaults               []domain.KeyVault
	secrets              []domain.KeyVaultSecret
	viewState            viewState

	// State
	loading  bool
	err      error
	ready    bool
	quitting bool

	// Dimensions
	width  int
	height int
}

// NewModel creates the initial application model.
// azureClient and cache are injected (DI) so tests can provide mocks.
func NewModel(azureClient infrastructure.AzureClient, cache *infrastructure.Cache) Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	return Model{
		spinner:     s,
		azureClient: azureClient,
		cache:       cache,
		loading:     true,
		viewState:   viewSubscriptions,
	}
}

// Init is the first command run by Bubble Tea.
// It kicks off the async data fetch.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.fetchData,
	)
}

// fetchData is a tea.Cmd that checks the cache first,
// then falls back to the Azure API. This runs in a goroutine
// managed by Bubble Tea, so the UI thread stays responsive.
func (m Model) fetchData() tea.Msg {
	// 1. Try the cache first.
	if m.cache.IsValid() {
		return fetchResultMsg{
			tenants:       m.cache.GetTenants(),
			subscriptions: m.cache.GetSubscriptions(),
		}
	}

	// 2. Cache miss or stale — hit the Azure API.
	ctx, cancel := context.WithTimeout(context.Background(), loadingTimeout)
	defer cancel()

	// Fetch tenants and subscriptions concurrently.
	type partial struct {
		tenants       []domain.Tenant
		subscriptions []domain.Subscription
		err           error
	}

	ch := make(chan partial, 1)

	go func() {
		var p partial

		// Fetch tenants.
		tenants, tErr := m.azureClient.ListTenants(ctx)
		if tErr != nil {
			p.err = tErr
			ch <- p
			return
		}
		p.tenants = tenants

		// Fetch subscriptions (across all tenants).
		subs, sErr := m.azureClient.ListSubscriptions(ctx)
		if sErr != nil {
			p.err = sErr
			ch <- p
			return
		}
		p.subscriptions = subs

		// Persist to cache for next launch.
		_ = m.cache.SaveTenants(tenants)
		_ = m.cache.SaveSubscriptions(subs)

		ch <- p
	}()

	select {
	case <-ctx.Done():
		return errMsg{err: fmt.Errorf("request timed out after %v", loadingTimeout)}
	case p := <-ch:
		return fetchResultMsg{
			tenants:       p.tenants,
			subscriptions: p.subscriptions,
			err:           p.err,
		}
	}
}

// Update handles incoming messages and updates the model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinner.TickMsg:
		if m.loading || m.viewState == viewVaultsLoading || m.viewState == viewSecretsLoading {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case fetchResultMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.tenants = msg.tenants
		m.subscriptions = msg.subscriptions
		m.ready = true
		return m, nil

	case vaultsMsg:
		if msg.err != nil {
			m.loading = false
			m.err = msg.err
			return m, nil
		}
		m.vaults = msg.vaults
		m.cursor = 0
		m.viewState = viewVaults
		return m, nil

	case secretsMsg:
		if msg.err != nil {
			m.loading = false
			m.err = msg.err
			return m, nil
		}
		m.secrets = msg.secrets
		m.cursor = 0
		m.viewState = viewSecrets
		return m, nil

	case errMsg:
		m.loading = false
		m.err = msg.err
		return m, nil
	}

	// Keep the spinner ticking while loading.
	if m.loading {
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleKeyMsg processes keyboard input based on the current viewState.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		m.quitting = true
		return m, tea.Quit

	case "esc":
		return m.handleEsc()

	case "up", "k":
		return m.handleUp()

	case "down", "j":
		return m.handleDown()

	case "enter":
		return m.handleEnter()
	}

	return m, nil
}

func (m Model) handleEsc() (tea.Model, tea.Cmd) {
	switch m.viewState {
	case viewVaults:
		// Go back to subscriptions.
		m.viewState = viewSubscriptions
		m.cursor = 0
		m.selectedSubscription = nil
		return m, nil

	case viewVaultsLoading:
		// Cancel loading and go back to subscriptions.
		m.viewState = viewSubscriptions
		m.cursor = 0
		return m, nil

	case viewSecrets:
		// Go back to vaults list.
		m.viewState = viewVaults
		m.cursor = 0
		m.selectedVault = nil
		return m, nil

	case viewSecretsLoading:
		// Cancel loading and go back to vaults.
		m.viewState = viewVaults
		m.cursor = 0
		return m, nil

	default:
		// viewSubscriptions: refresh data.
		if m.ready {
			m.ready = false
			m.loading = true
			m.err = nil
			m.viewState = viewSubscriptions
			return m, m.fetchData
		}
	}

	return m, nil
}

func (m Model) handleUp() (tea.Model, tea.Cmd) {
	switch m.viewState {
	case viewSubscriptions:
		if m.cursor > 0 {
			m.cursor--
		}
	case viewVaults:
		if m.cursor > 0 {
			m.cursor--
		}
	case viewSecrets:
		if m.cursor > 0 {
			m.cursor--
		}
	}
	return m, nil
}

func (m Model) handleDown() (tea.Model, tea.Cmd) {
	switch m.viewState {
	case viewSubscriptions:
		if len(m.subscriptions) > 0 && m.cursor < len(m.subscriptions)-1 {
			m.cursor++
		}
	case viewVaults:
		if len(m.vaults) > 0 && m.cursor < len(m.vaults)-1 {
			m.cursor++
		}
	case viewSecrets:
		if len(m.secrets) > 0 && m.cursor < len(m.secrets)-1 {
			m.cursor++
		}
	}
	return m, nil
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.viewState {
	case viewSubscriptions:
		if len(m.subscriptions) > 0 && m.cursor < len(m.subscriptions) {
			sub := m.subscriptions[m.cursor]
			m.selectedSubscription = &sub
			m.viewState = viewVaultsLoading
			return m, m.fetchVaults(sub.ID)
		}

	case viewVaults:
		if len(m.vaults) > 0 && m.cursor < len(m.vaults) {
			vault := m.vaults[m.cursor]
			m.selectedVault = &vault
			m.viewState = viewSecretsLoading
			return m, m.fetchSecrets(vault.Properties.VaultURI)
		}
	}

	return m, nil
}

// View renders the current UI state.
func (m Model) View() string {
	if m.quitting {
		return "Goodbye! 👋\n"
	}

	if m.loading {
		return m.renderLoading()
	}

	if m.err != nil {
		return m.renderError()
	}

	switch m.viewState {
	case viewVaultsLoading:
		return m.renderVaultsLoading()
	case viewVaults:
		return m.renderVaults()
	case viewSecretsLoading:
		return m.renderSecretsLoading()
	case viewSecrets:
		return m.renderSecrets()
	default:
		return m.renderSubscriptions()
	}
}

// renderLoading shows the spinner while data is fetched.
func (m Model) renderLoading() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render(" AzCockpit "))
	b.WriteString("\n\n")
	b.WriteString(m.spinner.View())
	b.WriteString(" Fetching Azure resources...")
	b.WriteString("\n\n")

	if m.cache.IsValid() {
		b.WriteString(HelpStyle.Render("  (using cached data)"))
	} else {
		b.WriteString(HelpStyle.Render("  (authenticating via DefaultAzureCredential...)"))
	}

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		b.String(),
	)
}

// renderError shows a formatted error message with a retry hint.
func (m Model) renderError() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render(" AzCockpit "))
	b.WriteString("\n\n")
	b.WriteString(ErrorStyle.Render(fmt.Sprintf("  ✗ Error: %v", m.err)))
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("  Press ESC to retry • q to quit"))

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		b.String(),
	)
}

// renderSubscriptions renders the subscription list with tenant grouping and cursor highlighting.
func (m Model) renderSubscriptions() string {
	var b strings.Builder

	// Header
	b.WriteString(TitleStyle.Render(" AzCockpit "))
	b.WriteString("\n\n")

	// Tenant section
	if len(m.tenants) > 0 {
		b.WriteString(TenantStyle.Render(fmt.Sprintf("Tenants (%d)", len(m.tenants))))
		b.WriteString("\n")
		for _, t := range m.tenants {
			marker := "  "
			if t.IsHomeTenant() {
				marker = "🏠 "
			}
			b.WriteString(fmt.Sprintf("%s%s", marker, t.DisplayName))
			if t.DefaultDomain != "" {
				b.WriteString(HelpStyle.Render(fmt.Sprintf("  (%s)", t.DefaultDomain)))
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Subscriptions section
	b.WriteString(TenantStyle.Render(fmt.Sprintf("Subscriptions (%d)", len(m.subscriptions))))
	b.WriteString("\n")

	if len(m.subscriptions) == 0 {
		b.WriteString(HelpStyle.Render("  No subscriptions found."))
		b.WriteString("\n")
	} else {
		for i, s := range m.subscriptions {
			style := ActiveStyle
			if !s.IsActive() {
				style = DisabledStyle
			}

			line := fmt.Sprintf("  %s %s", style.Render("●"), s.DisplayName)
			line += HelpStyle.Render(fmt.Sprintf("  (%s)", s.ID))
			line += style.Render(fmt.Sprintf("  [%s]", s.State))

			if i == m.cursor {
				line = SelectedStyle.Render(line)
			}

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  ↑/↓ navigate • enter browse vaults • esc refresh • q quit"))
	b.WriteString("\n")

	// Center content if dimensions are known.
	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			b.String(),
		)
	}

	return b.String()
}

// renderVaultsLoading shows the spinner while key vaults are being fetched.
func (m Model) renderVaultsLoading() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render(" AzCockpit "))
	b.WriteString("\n\n")
	b.WriteString(m.spinner.View())
	if m.selectedSubscription != nil {
		b.WriteString(fmt.Sprintf(" Browsing Key Vaults in: %s", m.selectedSubscription.DisplayName))
	} else {
		b.WriteString(" Browsing Key Vaults...")
	}
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("  esc back • q quit"))

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		b.String(),
	)
}

// renderVaults renders the key vaults list for the selected subscription.
func (m Model) renderVaults() string {
	var b strings.Builder

	// Header
	b.WriteString(TitleStyle.Render(" AzCockpit "))
	b.WriteString("\n\n")

	// Context breadcrumb
	if m.selectedSubscription != nil {
		b.WriteString(TenantStyle.Render(fmt.Sprintf("Key Vaults in: %s", m.selectedSubscription.DisplayName)))
		b.WriteString("\n")
		b.WriteString(HelpStyle.Render(fmt.Sprintf("Subscription: %s", m.selectedSubscription.ID)))
		b.WriteString("\n\n")
	}

	b.WriteString(TenantStyle.Render(fmt.Sprintf("Vaults (%d)", len(m.vaults))))
	b.WriteString("\n")

	if len(m.vaults) == 0 {
		b.WriteString(HelpStyle.Render("  No key vaults found in this subscription."))
		b.WriteString("\n")
	} else {
		for i, v := range m.vaults {
			line := fmt.Sprintf("  🔐 %s", v.Name)
			if v.Location != "" {
				line += HelpStyle.Render(fmt.Sprintf("  (%s)", v.Location))
			}

			if i == m.cursor {
				line = SelectedStyle.Render(line)
			}

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  ↑/↓ navigate • enter list secrets • esc back • q quit"))
	b.WriteString("\n")

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			b.String(),
		)
	}

	return b.String()
}

// renderSecretsLoading shows the spinner while secrets are being fetched.
func (m Model) renderSecretsLoading() string {
	var b strings.Builder

	b.WriteString(TitleStyle.Render(" AzCockpit "))
	b.WriteString("\n\n")
	b.WriteString(m.spinner.View())
	if m.selectedVault != nil {
		b.WriteString(fmt.Sprintf(" Browsing Secrets in: %s", m.selectedVault.Name))
	} else {
		b.WriteString(" Browsing Secrets...")
	}
	b.WriteString("\n\n")
	b.WriteString(HelpStyle.Render("  esc back • q quit"))

	return lipgloss.Place(
		m.width, m.height,
		lipgloss.Center, lipgloss.Center,
		b.String(),
	)
}

// renderSecrets renders the secrets list for the selected key vault.
func (m Model) renderSecrets() string {
	var b strings.Builder

	// Header
	b.WriteString(TitleStyle.Render(" AzCockpit "))
	b.WriteString("\n\n")

	// Context breadcrumb
	if m.selectedVault != nil {
		b.WriteString(TenantStyle.Render(fmt.Sprintf("Secrets in: %s", m.selectedVault.Name)))
		b.WriteString("\n")
		if m.selectedVault.Properties.VaultURI != "" {
			b.WriteString(HelpStyle.Render(fmt.Sprintf("URI: %s", m.selectedVault.Properties.VaultURI)))
		}
		b.WriteString("\n\n")
	}

	b.WriteString(TenantStyle.Render(fmt.Sprintf("Secrets (%d)", len(m.secrets))))
	b.WriteString("\n")

	if len(m.secrets) == 0 {
		b.WriteString(HelpStyle.Render("  No secrets found in this vault."))
		b.WriteString("\n")
	} else {
		for i, s := range m.secrets {
			style := ActiveStyle
			if !s.Enabled {
				style = DisabledStyle
			}

			line := fmt.Sprintf("  %s %s", style.Render("🔑"), s.Name)
			if s.ContentType != "" {
				line += HelpStyle.Render(fmt.Sprintf("  (%s)", s.ContentType))
			}

			if i == m.cursor {
				line = SelectedStyle.Render(line)
			}

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Footer
	b.WriteString("\n")
	b.WriteString(HelpStyle.Render("  ↑/↓ navigate • esc back • q quit"))
	b.WriteString("\n")

	if m.width > 0 && m.height > 0 {
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			b.String(),
		)
	}

	return b.String()
}

// fetchVaults returns a tea.Cmd that asynchronously fetches Key Vaults
// for the given subscription ID.
func (m Model) fetchVaults(subscriptionID string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), loadingTimeout)
		defer cancel()

		type result struct {
			vaults []domain.KeyVault
			err    error
		}

		ch := make(chan result, 1)

		go func() {
			vaults, err := m.azureClient.ListKeyVaults(ctx, subscriptionID)
			ch <- result{vaults: vaults, err: err}
		}()

		select {
		case <-ctx.Done():
			return vaultsMsg{
				subscriptionID: subscriptionID,
				err:            fmt.Errorf("request timed out after %v", loadingTimeout),
			}
		case r := <-ch:
			return vaultsMsg{
				subscriptionID: subscriptionID,
				vaults:         r.vaults,
				err:            r.err,
			}
		}
	}
}

// fetchSecrets returns a tea.Cmd that asynchronously fetches secrets
// for the given Key Vault URI.
func (m Model) fetchSecrets(vaultURI string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), loadingTimeout)
		defer cancel()

		type result struct {
			secrets []domain.KeyVaultSecret
			err     error
		}

		ch := make(chan result, 1)

		go func() {
			secrets, err := m.azureClient.ListKeyVaultSecrets(ctx, vaultURI)
			ch <- result{secrets: secrets, err: err}
		}()

		select {
		case <-ctx.Done():
			return secretsMsg{
				vaultURI: vaultURI,
				err:      fmt.Errorf("request timed out after %v", loadingTimeout),
			}
		case r := <-ch:
			return secretsMsg{
				vaultURI: vaultURI,
				secrets:  r.secrets,
				err:      r.err,
			}
		}
	}
}
