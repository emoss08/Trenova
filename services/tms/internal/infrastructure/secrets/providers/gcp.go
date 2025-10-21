package providers

import (
	"context"
	"errors"
	"fmt"
	"strings"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
	"github.com/emoss08/trenova/internal/core/ports"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var _ ports.SecretProvider = (*GCPProvider)(nil)

type GCPProvider struct {
	client    *secretmanager.Client
	projectID string
}

func NewGCPProvider(ctx context.Context, cfg map[string]string) (*GCPProvider, error) {
	projectID := cfg["project_id"]
	if projectID == "" {
		return nil, ErrProjectIDRequiredGCP
	}

	var opts []option.ClientOption

	if keyFile := cfg["credentials_file"]; keyFile != "" {
		opts = append(opts, option.WithCredentialsFile(keyFile))
	} else if credentials := cfg["credentials"]; credentials != "" {
		opts = append(opts, option.WithCredentialsJSON([]byte(credentials)))
	}

	if endpoint := cfg["endpoint"]; endpoint != "" {
		opts = append(opts, option.WithEndpoint(endpoint))
	}

	client, err := secretmanager.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCP Secret Manager client: %w", err)
	}

	return &GCPProvider{
		client:    client,
		projectID: projectID,
	}, nil
}

func (p *GCPProvider) GetSecret(ctx context.Context, key string) (string, error) {
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", p.projectID, key)

	result, err := p.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return "", fmt.Errorf("failed to access secret %s: %w", key, err)
	}

	return string(result.GetPayload().GetData()), nil
}

func (p *GCPProvider) GetSecrets(ctx context.Context, keys []string) (map[string]string, error) {
	secrets := make(map[string]string)
	var errs []string

	for _, key := range keys {
		value, err := p.GetSecret(ctx, key)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", key, err))
			continue
		}
		secrets[key] = value
	}

	if len(errs) > 0 {
		return secrets, fmt.Errorf("failed to get some secrets: %s", strings.Join(errs, "; "))
	}

	return secrets, nil
}

func (p *GCPProvider) GetBinarySecret(ctx context.Context, key string) ([]byte, error) {
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/latest", p.projectID, key)

	result, err := p.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to access binary secret %s: %w", key, err)
	}

	return result.GetPayload().GetData(), nil
}

func (p *GCPProvider) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

func (p *GCPProvider) GetSecretVersion(
	ctx context.Context,
	key string,
	version string,
) (string, error) {
	name := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", p.projectID, key, version)

	result, err := p.client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: name,
	})
	if err != nil {
		return "", fmt.Errorf("failed to access secret %s version %s: %w", key, version, err)
	}

	return string(result.GetPayload().GetData()), nil
}

func (p *GCPProvider) ListSecrets(ctx context.Context) ([]string, error) {
	var secretNames []string

	req := &secretmanagerpb.ListSecretsRequest{
		Parent: fmt.Sprintf("projects/%s", p.projectID),
	}

	it := p.client.ListSecrets(ctx, req)
	for {
		secret, err := it.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list secrets: %w", err)
		}

		parts := strings.Split(secret.GetName(), "/")
		if len(parts) >= 4 {
			secretNames = append(secretNames, parts[3])
		}
	}

	return secretNames, nil
}

func (p *GCPProvider) CreateSecret(ctx context.Context, key, value string) error {
	createReq := &secretmanagerpb.CreateSecretRequest{
		Parent:   fmt.Sprintf("projects/%s", p.projectID),
		SecretId: key,
		Secret: &secretmanagerpb.Secret{
			Replication: &secretmanagerpb.Replication{
				Replication: &secretmanagerpb.Replication_Automatic_{
					Automatic: &secretmanagerpb.Replication_Automatic{},
				},
			},
		},
	}

	secret, err := p.client.CreateSecret(ctx, createReq)
	if err != nil {
		return fmt.Errorf("failed to create secret %s: %w", key, err)
	}

	addReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: secret.GetName(),
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(value),
		},
	}

	_, err = p.client.AddSecretVersion(ctx, addReq)
	if err != nil {
		return fmt.Errorf("failed to add secret version for %s: %w", key, err)
	}

	return nil
}

func (p *GCPProvider) UpdateSecret(ctx context.Context, key, value string) error {
	secretName := fmt.Sprintf("projects/%s/secrets/%s", p.projectID, key)

	addReq := &secretmanagerpb.AddSecretVersionRequest{
		Parent: secretName,
		Payload: &secretmanagerpb.SecretPayload{
			Data: []byte(value),
		},
	}

	_, err := p.client.AddSecretVersion(ctx, addReq)
	if err != nil {
		return fmt.Errorf("failed to update secret %s: %w", key, err)
	}

	return nil
}

func (p *GCPProvider) DeleteSecret(ctx context.Context, key string) error {
	secretName := fmt.Sprintf("projects/%s/secrets/%s", p.projectID, key)

	req := &secretmanagerpb.DeleteSecretRequest{
		Name: secretName,
	}

	err := p.client.DeleteSecret(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to delete secret %s: %w", key, err)
	}

	return nil
}

func (p *GCPProvider) DisableSecretVersion(ctx context.Context, key, version string) error {
	versionName := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", p.projectID, key, version)

	req := &secretmanagerpb.DisableSecretVersionRequest{
		Name: versionName,
	}

	_, err := p.client.DisableSecretVersion(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to disable secret %s version %s: %w", key, version, err)
	}

	return nil
}

func (p *GCPProvider) EnableSecretVersion(ctx context.Context, key, version string) error {
	versionName := fmt.Sprintf("projects/%s/secrets/%s/versions/%s", p.projectID, key, version)

	req := &secretmanagerpb.EnableSecretVersionRequest{
		Name: versionName,
	}

	_, err := p.client.EnableSecretVersion(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to enable secret %s version %s: %w", key, version, err)
	}

	return nil
}
