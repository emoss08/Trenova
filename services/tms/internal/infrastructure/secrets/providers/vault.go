package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports"
	vault "github.com/hashicorp/vault-client-go"
	"github.com/hashicorp/vault-client-go/schema"
)

var _ ports.SecretProvider = (*VaultProvider)(nil)

type VaultProvider struct {
	client    *vault.Client
	mountPath string
	version   string // KV v1 or v2
}

func NewHashiCorpVaultProvider(ctx context.Context, cfg map[string]string) (*VaultProvider, error) {
	address := cfg["address"]
	if address == "" {
		address = "http://localhost:8200"
	}

	token := cfg["token"]
	if token == "" {
		return nil, ErrVaultTokenRequired
	}

	mountPath := cfg["mount_path"]
	if mountPath == "" {
		mountPath = "secret"
	}

	version := cfg["version"]
	if version == "" {
		version = "v2"
	}

	client, err := vault.New(
		vault.WithAddress(address),
		vault.WithRequestTimeout(30*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Vault client: %w", err)
	}

	if err = client.SetToken(token); err != nil {
		return nil, fmt.Errorf("failed to set Vault token: %w", err)
	}

	healthResp, err := client.System.ReadHealthStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Vault: %w", err)
	}

	if healthResp.Data != nil {
		if sealed, ok := healthResp.Data["sealed"].(bool); ok && sealed {
			return nil, ErrVaultSealed
		}
	}

	return &VaultProvider{
		client:    client,
		mountPath: mountPath,
		version:   version,
	}, nil
}

func (p *VaultProvider) GetSecret(ctx context.Context, key string) (string, error) {
	if p.version == "v2" {
		return p.getSecretV2(ctx, key)
	}
	return p.getSecretV1(ctx, key)
}

func (p *VaultProvider) getSecretV2(ctx context.Context, key string) (string, error) {
	response, err := p.client.Secrets.KvV2Read(ctx, key, vault.WithMountPath(p.mountPath))
	if err != nil {
		return "", fmt.Errorf("failed to read secret %s from Vault: %w", key, err)
	}

	if response == nil {
		return "", fmt.Errorf("secret %s not found", key)
	}

	if response.Data.Data == nil {
		return "", fmt.Errorf("secret %s has no data", key)
	}

	return p.extractSecretValue(response.Data.Data)
}

func (p *VaultProvider) getSecretV1(ctx context.Context, key string) (string, error) {
	path := fmt.Sprintf("%s/%s", p.mountPath, key)
	response, err := p.client.Read(ctx, path)
	if err != nil {
		return "", fmt.Errorf("failed to read secret %s from Vault: %w", key, err)
	}

	if response == nil || response.Data == nil {
		return "", fmt.Errorf("secret %s not found", key)
	}

	return p.extractSecretValue(response.Data)
}

func (p *VaultProvider) extractSecretValue(data map[string]any) (string, error) {
	if value, ok := data["value"]; ok {
		return fmt.Sprintf("%v", value), nil
	}

	jsonBytes, err := sonic.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal secret data: %w", err)
	}

	return string(jsonBytes), nil
}

func (p *VaultProvider) GetSecrets(ctx context.Context, keys []string) (map[string]string, error) {
	secrets := make(map[string]string, len(keys))

	for _, key := range keys {
		value, err := p.GetSecret(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to get secret %s: %w", key, err)
		}
		secrets[key] = value
	}

	return secrets, nil
}

func (p *VaultProvider) GetBinarySecret(ctx context.Context, key string) ([]byte, error) {
	value, err := p.GetSecret(ctx, key)
	if err != nil {
		return nil, err
	}
	return []byte(value), nil
}

func (p *VaultProvider) Close() error {
	return nil
}

func (p *VaultProvider) WriteSecret(ctx context.Context, key string, data map[string]any) error {
	if p.version == "v2" {
		_, err := p.client.Secrets.KvV2Write(ctx, key, schema.KvV2WriteRequest{
			Data: data,
		}, vault.WithMountPath(p.mountPath))
		if err != nil {
			return fmt.Errorf("failed to write secret %s to Vault: %w", key, err)
		}
	} else {
		path := fmt.Sprintf("%s/%s", p.mountPath, key)
		_, err := p.client.Write(ctx, path, data)
		if err != nil {
			return fmt.Errorf("failed to write secret %s to Vault: %w", key, err)
		}
	}

	return nil
}

func (p *VaultProvider) DeleteSecret(ctx context.Context, key string) error {
	if p.version == "v2" {
		_, err := p.client.Secrets.KvV2Delete(ctx, key, vault.WithMountPath(p.mountPath))
		if err != nil {
			return fmt.Errorf("failed to delete secret %s from Vault: %w", key, err)
		}
	} else {
		path := fmt.Sprintf("%s/%s", p.mountPath, key)
		_, err := p.client.Delete(ctx, path)
		if err != nil {
			return fmt.Errorf("failed to delete secret %s from Vault: %w", key, err)
		}
	}

	return nil
}

func (p *VaultProvider) ListSecrets(ctx context.Context, path string) ([]string, error) {
	var keys []string

	if p.version == "v2" { //nolint:nestif // This is a valid check
		response, err := p.client.Secrets.KvV2List(ctx, path, vault.WithMountPath(p.mountPath))
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets at %s: %w", path, err)
		}

		if response != nil {
			return response.Data.Keys, nil
		}
	} else {
		fullPath := fmt.Sprintf("%s/%s", p.mountPath, path)
		response, err := p.client.List(ctx, fullPath)
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets at %s: %w", path, err)
		}

		if response != nil && response.Data != nil {
			if keysList, ok := response.Data["keys"].([]any); ok {
				keys = make([]string, 0, len(keysList))
				for _, k := range keysList {
					if keyStr, keyOk := k.(string); keyOk {
						keys = append(keys, keyStr)
					}
				}
			}
		}
	}

	return keys, nil
}

func (p *VaultProvider) RenewToken(ctx context.Context) error {
	_, err := p.client.Auth.TokenRenewSelf(ctx, schema.TokenRenewSelfRequest{})
	if err != nil {
		return fmt.Errorf("failed to renew Vault token: %w", err)
	}
	return nil
}
