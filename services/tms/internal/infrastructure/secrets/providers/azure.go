package providers

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/keyvault/azsecrets"
	"github.com/emoss08/trenova/internal/core/ports"
)

var _ ports.SecretProvider = (*AzureProvider)(nil)

type AzureProvider struct {
	client   *azsecrets.Client
	vaultURL string
}

func NewAzureProvider(_ context.Context, cfg map[string]string) (*AzureProvider, error) {
	vaultName := cfg["vault_name"]
	if vaultName == "" {
		return nil, ErrAzureVaultNameRequired
	}

	vaultURL := fmt.Sprintf("https://%s.vault.azure.net", vaultName)
	if customURL := cfg["vault_url"]; customURL != "" {
		vaultURL = customURL
	}

	var cred azcore.TokenCredential
	var err error

	authMethod := cfg["auth_method"]
	if authMethod == "" {
		authMethod = "default"
	}

	switch authMethod {
	case "client_secret":
		tenantID := cfg["tenant_id"]
		clientID := cfg["client_id"]
		clientSecret := cfg["client_secret"]

		if tenantID == "" || clientID == "" || clientSecret == "" {
			return nil, ErrAzureTenantIDClientIDClientSecretRequired
		}

		cred, err = azidentity.NewClientSecretCredential(tenantID, clientID, clientSecret, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create client secret credential: %w", err)
		}

	case "managed_identity":
		clientID := cfg["client_id"]

		opts := &azidentity.ManagedIdentityCredentialOptions{}
		if clientID != "" {
			opts.ID = azidentity.ClientID(clientID)
		}

		cred, err = azidentity.NewManagedIdentityCredential(opts)
		if err != nil {
			return nil, fmt.Errorf("failed to create managed identity credential: %w", err)
		}

	case "cli":
		cred, err = azidentity.NewAzureCLICredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create Azure CLI credential: %w", err)
		}

	default:
		cred, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create default Azure credential: %w", err)
		}
	}

	client, err := azsecrets.NewClient(vaultURL, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Azure Key Vault client: %w", err)
	}

	return &AzureProvider{
		client:   client,
		vaultURL: vaultURL,
	}, nil
}

func (p *AzureProvider) GetSecret(ctx context.Context, key string) (string, error) {
	secretName := p.normalizeSecretName(key)

	resp, err := p.client.GetSecret(ctx, secretName, "", nil)
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s from Azure Key Vault: %w", key, err)
	}

	if resp.Value == nil {
		return "", fmt.Errorf("secret %s has no value", key)
	}

	return *resp.Value, nil
}

func (p *AzureProvider) GetSecrets(ctx context.Context, keys []string) (map[string]string, error) {
	secrets := make(map[string]string)
	var errors []string

	for _, key := range keys {
		value, err := p.GetSecret(ctx, key)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", key, err))
			continue
		}
		secrets[key] = value
	}

	if len(errors) > 0 {
		return secrets, fmt.Errorf("failed to get some secrets: %s", strings.Join(errors, "; "))
	}

	return secrets, nil
}

func (p *AzureProvider) GetBinarySecret(ctx context.Context, key string) ([]byte, error) {
	value, err := p.GetSecret(ctx, key)
	if err != nil {
		return nil, err
	}

	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return []byte(value), err
	}

	return decoded, nil
}

func (p *AzureProvider) Close() error {
	return nil
}

func (p *AzureProvider) ListSecrets(ctx context.Context) ([]string, error) {
	var secretNames []string

	pager := p.client.NewListSecretsPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		for _, secret := range page.Value {
			if secret.ID != nil {
				parts := strings.Split(string(*secret.ID), "/")
				if len(parts) > 0 {
					secretNames = append(secretNames, parts[len(parts)-1])
				}
			}
		}
	}

	return secretNames, nil
}

func (p *AzureProvider) SetSecret(ctx context.Context, key, value string) error {
	secretName := p.normalizeSecretName(key)

	params := azsecrets.SetSecretParameters{
		Value: &value,
	}

	_, err := p.client.SetSecret(ctx, secretName, params, nil)
	if err != nil {
		return fmt.Errorf("failed to set secret %s: %w", key, err)
	}

	return nil
}

func (p *AzureProvider) DeleteSecret(ctx context.Context, key string) error {
	secretName := p.normalizeSecretName(key)

	_, err := p.client.DeleteSecret(ctx, secretName, nil)
	if err != nil {
		return fmt.Errorf("failed to delete secret %s: %w", key, err)
	}

	return nil
}

func (p *AzureProvider) normalizeSecretName(name string) string {
	normalized := strings.ReplaceAll(name, "_", "-")
	normalized = strings.ReplaceAll(normalized, ".", "-")
	normalized = strings.ReplaceAll(normalized, "/", "-")

	var result strings.Builder
	for _, r := range normalized {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}

	return result.String()
}

func (p *AzureProvider) GetSecretVersion(ctx context.Context, key, version string) (string, error) {
	secretName := p.normalizeSecretName(key)

	resp, err := p.client.GetSecret(ctx, secretName, version, nil)
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s version %s: %w", key, version, err)
	}

	if resp.Value == nil {
		return "", fmt.Errorf("secret %s version %s has no value", key, version)
	}

	return *resp.Value, nil
}
