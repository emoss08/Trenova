package providers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports"
)

var _ ports.SecretProvider = (*FileProvider)(nil)

type FileProvider struct {
	basePath string
}

func NewFileProvider(config map[string]string) *FileProvider {
	basePath := config["base_path"]
	if basePath == "" {
		basePath = "/run/secrets"
	}
	return &FileProvider{
		basePath: basePath,
	}
}

func (p *FileProvider) GetSecret(_ context.Context, key string) (string, error) {
	filePath := p.getFilePath(key)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("secret file not found: %s", filePath)
		}
		return "", fmt.Errorf("failed to read secret file %s: %w", filePath, err)
	}

	secret := strings.TrimSpace(string(data))
	if secret == "" {
		return "", fmt.Errorf("secret file %s is empty", filePath)
	}

	return secret, nil
}

func (p *FileProvider) GetSecrets(ctx context.Context, keys []string) (map[string]string, error) {
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

func (p *FileProvider) GetBinarySecret(_ context.Context, key string) ([]byte, error) {
	filePath := p.getFilePath(key)

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("secret file not found: %s", filePath)
		}
		return nil, fmt.Errorf("failed to read secret file %s: %w", filePath, err)
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("secret file %s is empty", filePath)
	}

	return data, nil
}

func (p *FileProvider) Close() error {
	return nil
}

func (p *FileProvider) getFilePath(key string) string {
	safeName := strings.ReplaceAll(key, "..", "")
	safeName = strings.ReplaceAll(safeName, "/", "_")
	safeName = strings.ReplaceAll(safeName, "\\", "_")

	return filepath.Join(p.basePath, safeName)
}
