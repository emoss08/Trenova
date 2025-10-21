package providers

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports"
)

var _ ports.SecretProvider = (*AWSProvider)(nil)

type AWSProvider struct {
	client *secretsmanager.Client
	region string
}

func NewAWSProvider(ctx context.Context, cfg map[string]string) (*AWSProvider, error) {
	region := cfg["region"]
	if region == "" {
		region = "us-east-1"
	}

	awsCfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	if endpoint := cfg["endpoint"]; endpoint != "" {
		awsCfg.BaseEndpoint = aws.String(endpoint)
	}

	client := secretsmanager.NewFromConfig(awsCfg)

	return &AWSProvider{
		client: client,
		region: region,
	}, nil
}

func (p *AWSProvider) GetSecret(ctx context.Context, key string) (string, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(key),
	}

	result, err := p.client.GetSecretValue(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s from AWS Secrets Manager: %w", key, err)
	}

	if result.SecretString != nil {
		return *result.SecretString, nil
	}

	if len(result.SecretBinary) > 0 {
		return base64.StdEncoding.EncodeToString(result.SecretBinary), nil
	}

	return "", fmt.Errorf("secret %s has no value", key)
}

func (p *AWSProvider) GetSecrets(ctx context.Context, keys []string) (map[string]string, error) {
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

func (p *AWSProvider) GetBinarySecret(ctx context.Context, key string) ([]byte, error) {
	input := &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(key),
	}

	result, err := p.client.GetSecretValue(ctx, input)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to get binary secret %s from AWS Secrets Manager: %w",
			key,
			err,
		)
	}

	if len(result.SecretBinary) > 0 {
		return result.SecretBinary, nil
	}

	if result.SecretString != nil {
		return []byte(*result.SecretString), nil
	}

	return nil, fmt.Errorf("secret %s has no value", key)
}

func (p *AWSProvider) Close() error {
	return nil
}

func (p *AWSProvider) GetJSONSecret(ctx context.Context, key string) (map[string]string, error) {
	secretString, err := p.GetSecret(ctx, key)
	if err != nil {
		return nil, err
	}

	var result map[string]string
	if err = sonic.Unmarshal([]byte(secretString), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON secret %s: %w", key, err)
	}

	return result, nil
}

func (p *AWSProvider) CreateSecret(
	ctx context.Context,
	key, value string,
	description string,
) error {
	input := &secretsmanager.CreateSecretInput{
		Name:         aws.String(key),
		SecretString: aws.String(value),
	}

	if description != "" {
		input.Description = aws.String(description)
	}

	_, err := p.client.CreateSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create secret %s: %w", key, err)
	}

	return nil
}

func (p *AWSProvider) UpdateSecret(ctx context.Context, key, value string) error {
	input := &secretsmanager.UpdateSecretInput{
		SecretId:     aws.String(key),
		SecretString: aws.String(value),
	}

	_, err := p.client.UpdateSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to update secret %s: %w", key, err)
	}

	return nil
}

func (p *AWSProvider) DeleteSecret(ctx context.Context, key string, forceDelete bool) error {
	input := &secretsmanager.DeleteSecretInput{
		SecretId: aws.String(key),
	}

	if forceDelete {
		input.ForceDeleteWithoutRecovery = aws.Bool(true)
	}

	_, err := p.client.DeleteSecret(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete secret %s: %w", key, err)
	}

	return nil
}
