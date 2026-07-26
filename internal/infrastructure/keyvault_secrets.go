package infrastructure

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/security/keyvault/azsecrets"

	"github.com/nousresearch/azcockpit/internal/domain"
)

// keyVaultSecretsClient wraps the azsecrets data-plane client for a single vault.
type keyVaultSecretsClient struct {
	client *azsecrets.Client
}

// newKeyVaultSecretsClient creates a new data-plane secrets client for the
// given vault URI.
func newKeyVaultSecretsClient(vaultURI string, credential azcore.TokenCredential) (*keyVaultSecretsClient, error) {
	client, err := azsecrets.NewClient(vaultURI, credential, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create azsecrets client for %s: %w", vaultURI, err)
	}
	return &keyVaultSecretsClient{client: client}, nil
}

// listSecrets lists all secrets in the vault using the azsecrets pager API.
func (c *keyVaultSecretsClient) listSecrets(ctx context.Context) ([]domain.KeyVaultSecret, error) {
	var secrets []domain.KeyVaultSecret

	pager := c.client.NewListSecretPropertiesPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		for _, sp := range page.Value {
			if sp == nil {
				continue
			}

			s := domain.KeyVaultSecret{
				ID:          derefID(sp.ID),
				Name:        secretNameFromID(derefID(sp.ID)),
				ContentType: derefString(sp.ContentType),
				Tags:        copyStringMap(sp.Tags),
			}

			if sp.Attributes != nil {
				s.Enabled = derefBool(sp.Attributes.Enabled)
				s.NotBefore = sp.Attributes.NotBefore
				s.Expires = sp.Attributes.Expires
			}

			secrets = append(secrets, s)
		}
	}

	return secrets, nil
}

// getSecret retrieves the current value of a named secret.
func (c *keyVaultSecretsClient) getSecret(ctx context.Context, name string) (string, error) {
	resp, err := c.client.GetSecret(ctx, name, "", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get secret %q: %w", name, err)
	}
	if resp.Value == nil {
		return "", nil
	}
	return *resp.Value, nil
}

// ----- helpers -----

// derefID converts a potentially nil *azsecrets.ID to a plain string.
func derefID(id *azsecrets.ID) string {
	if id == nil {
		return ""
	}
	return string(*id)
}

// secretNameFromID extracts the secret name from a Key Vault secret ID.
//
// IDs have the format:
//
//	https://<vault>.vault.azure.net/secrets/<name>/<version>
func secretNameFromID(id string) string {
	// Find "/secrets/" segment.
	idx := strings.Index(id, "/secrets/")
	if idx < 0 {
		return ""
	}
	rest := id[idx+len("/secrets/"):]
	// Take up to the next "/".
	if slash := strings.Index(rest, "/"); slash >= 0 {
		return rest[:slash]
	}
	return rest
}
