package providers

import "errors"

var (
	ErrAzureVaultNameRequired = errors.New(
		"vault name is required for Azure provider",
	)
	ErrAzureTenantIDClientIDClientSecretRequired = errors.New(
		"tenant id, client id, and client secret are required for client secret auth",
	)
	ErrProjectIDRequiredGCP = errors.New(
		"project id is required for GCP provider",
	)
	ErrFileProviderBasePathRequired = errors.New(
		"base path is required for file provider",
	)
	ErrJSONProviderFilePathRequired = errors.New(
		"file path is required for JSON provider",
	)
	ErrUpdateSecretsRequiresAPI = errors.New(
		"updating secrets requires API access",
	)
	ErrDeleteSecretsRequiresAPI = errors.New(
		"deleting secrets requires API access",
	)
	ErrCreateSecretsRequiresAPI = errors.New(
		"creating secrets requires API access",
	)
	ErrVaultTokenRequired = errors.New(
		"vault token is required",
	)
	ErrVaultSealed = errors.New(
		"vault is sealed",
	)
)
