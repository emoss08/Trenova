package providers

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var _ ports.SecretProvider = (*KubernetesProvider)(nil)

type KubernetesProvider struct {
	client      kubernetes.Interface
	namespace   string
	mountedPath string
	useAPI      bool
}

func NewKubernetesProvider(
	_ context.Context,
	cfg map[string]string,
) (*KubernetesProvider, error) {
	namespace := cfg["namespace"]
	if namespace == "" {
		nsPath := "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
		if data, err := os.ReadFile(nsPath); err == nil {
			namespace = strings.TrimSpace(string(data))
		} else {
			namespace = "default"
		}
	}

	provider := &KubernetesProvider{
		namespace:   namespace,
		mountedPath: cfg["mounted_path"],
	}

	if provider.mountedPath == "" {
		provider.mountedPath = "/var/run/secrets"
	}

	useAPI := cfg["use_api"]
	if useAPI == "true" || useAPI == "yes" || //nolint:nestif // This is a valid check
		useAPI == "1" {
		provider.useAPI = true

		var config *rest.Config

		kubeconfig := cfg["kubeconfig"]
		if kubeconfig == "" {
			kubeconfig = os.Getenv("KUBECONFIG")
		}
		if kubeconfig == "" {
			home, _ := os.UserHomeDir()
			kubeconfig = filepath.Join(home, ".kube", "config")
		}

		if _, err := os.Stat(kubeconfig); err == nil {
			config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
			if err != nil {
				return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
			}
		} else {
			config, err = rest.InClusterConfig()
			if err != nil {
				provider.useAPI = false
			}
		}

		if provider.useAPI && config != nil {
			client, err := kubernetes.NewForConfig(config)
			if err != nil {
				return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
			}
			provider.client = client
		}
	}

	return provider, nil
}

func (p *KubernetesProvider) GetSecret(ctx context.Context, key string) (string, error) {
	if p.useAPI {
		return p.getSecretFromAPI(ctx, key)
	}
	return p.getSecretFromMount(key)
}

func (p *KubernetesProvider) getSecretFromAPI(ctx context.Context, key string) (string, error) {
	parts := strings.SplitN(key, "/", 2)
	secretName := parts[0]
	var dataKey string
	if len(parts) > 1 {
		dataKey = parts[1]
	}

	secret, err := p.client.CoreV1().Secrets(p.namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}

	if dataKey == "" {
		if len(secret.Data) == 1 {
			for _, value := range secret.Data {
				return string(value), nil
			}
		}
		return "", fmt.Errorf("secret %s contains multiple keys, specify one", secretName)
	}

	value, exists := secret.Data[dataKey]
	if !exists {
		return "", fmt.Errorf("key %s not found in secret %s", dataKey, secretName)
	}

	return string(value), nil
}

func (p *KubernetesProvider) getSecretFromMount(key string) (string, error) {
	secretPath := filepath.Join(p.mountedPath, key)

	data, err := os.ReadFile(secretPath)
	if err != nil { //nolint:nestif // This is a valid check
		if os.IsNotExist(err) {
			if p.namespace != "" {
				namespacedPath := filepath.Join(p.mountedPath, p.namespace, key)
				data, err = os.ReadFile(namespacedPath)
				if err != nil {
					return "", fmt.Errorf("secret not found: %s", key)
				}
			} else {
				return "", fmt.Errorf("secret not found: %s", key)
			}
		} else {
			return "", fmt.Errorf("failed to read secret %s: %w", key, err)
		}
	}

	return strings.TrimSpace(string(data)), nil
}

func (p *KubernetesProvider) GetSecrets(
	ctx context.Context,
	keys []string,
) (map[string]string, error) {
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

// GetBinarySecret retrieves a binary secret from Kubernetes
func (p *KubernetesProvider) GetBinarySecret(ctx context.Context, key string) ([]byte, error) {
	if p.useAPI {
		return p.getBinarySecretFromAPI(ctx, key)
	}
	return p.getBinarySecretFromMount(key)
}

// getBinarySecretFromAPI retrieves a binary secret using the Kubernetes API
func (p *KubernetesProvider) getBinarySecretFromAPI(
	ctx context.Context,
	key string,
) ([]byte, error) {
	// Parse the key format
	parts := strings.SplitN(key, "/", 2)
	secretName := parts[0]
	var dataKey string
	if len(parts) > 1 {
		dataKey = parts[1]
	}

	// Get the secret from Kubernetes
	secret, err := p.client.CoreV1().Secrets(p.namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret %s: %w", secretName, err)
	}

	// If no specific key was requested, return error
	if dataKey == "" {
		if len(secret.Data) == 1 {
			for _, value := range secret.Data {
				return value, nil
			}
		}
		return nil, fmt.Errorf("secret %s contains multiple keys, specify one", secretName)
	}

	// Get the specific key from the secret
	value, exists := secret.Data[dataKey]
	if !exists {
		return nil, fmt.Errorf("key %s not found in secret %s", dataKey, secretName)
	}

	return value, nil
}

// getBinarySecretFromMount retrieves a binary secret from mounted volume
func (p *KubernetesProvider) getBinarySecretFromMount(key string) ([]byte, error) {
	secretPath := filepath.Join(p.mountedPath, key)

	data, err := os.ReadFile(secretPath)
	if err != nil {
		if os.IsNotExist(err) && p.namespace != "" {
			// Try with namespace prefix
			namespacedPath := filepath.Join(p.mountedPath, p.namespace, key)
			data, err = os.ReadFile(namespacedPath)
			if err != nil {
				return nil, fmt.Errorf("secret not found: %s", key)
			}
		} else {
			return nil, fmt.Errorf("failed to read secret %s: %w", key, err)
		}
	}

	return data, nil
}

// Close does nothing for Kubernetes provider
func (p *KubernetesProvider) Close() error {
	return nil
}

// ListSecrets lists available secrets
func (p *KubernetesProvider) ListSecrets(ctx context.Context) ([]string, error) {
	if p.useAPI {
		return p.listSecretsFromAPI(ctx)
	}
	return p.listSecretsFromMount()
}

// listSecretsFromAPI lists secrets using the Kubernetes API
func (p *KubernetesProvider) listSecretsFromAPI(ctx context.Context) ([]string, error) {
	secretList, err := p.client.CoreV1().Secrets(p.namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	var secrets []string
	for i := range secretList.Items {
		secret := &secretList.Items[i]
		for key := range secret.Data {
			secrets = append(secrets, fmt.Sprintf("%s/%s", secret.Name, key))
		}
	}

	return secrets, nil
}

// listSecretsFromMount lists secrets from mounted volumes
func (p *KubernetesProvider) listSecretsFromMount() ([]string, error) {
	var secrets []string

	// Check base path
	entries, err := os.ReadDir(p.mountedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			secrets = append(secrets, entry.Name())
		}
	}

	// Check namespace path if it exists
	if p.namespace != "" {
		namespacedPath := filepath.Join(p.mountedPath, p.namespace)
		if ent, entErr := os.ReadDir(namespacedPath); entErr == nil {
			for _, entry := range ent {
				if !entry.IsDir() {
					secrets = append(secrets, fmt.Sprintf("%s/%s", p.namespace, entry.Name()))
				}
			}
		}
	}

	return secrets, nil
}

func (p *KubernetesProvider) CreateSecret(
	ctx context.Context,
	name string,
	data map[string][]byte,
) error {
	if !p.useAPI {
		return ErrCreateSecretsRequiresAPI
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: p.namespace,
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}

	_, err := p.client.CoreV1().Secrets(p.namespace).Create(ctx, secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create secret %s: %w", name, err)
	}

	return nil
}

func (p *KubernetesProvider) UpdateSecret(
	ctx context.Context,
	name string,
	data map[string][]byte,
) error {
	if !p.useAPI {
		return ErrUpdateSecretsRequiresAPI
	}

	secret, err := p.client.CoreV1().Secrets(p.namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get secret %s: %w", name, err)
	}

	secret.Data = data

	_, err = p.client.CoreV1().Secrets(p.namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update secret %s: %w", name, err)
	}

	return nil
}

func (p *KubernetesProvider) DeleteSecret(ctx context.Context, name string) error {
	if !p.useAPI {
		return ErrDeleteSecretsRequiresAPI
	}

	err := p.client.CoreV1().Secrets(p.namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete secret %s: %w", name, err)
	}

	return nil
}

func (p *KubernetesProvider) CreateStringSecret(
	ctx context.Context,
	name string,
	data map[string]string,
) error {
	byteData := make(map[string][]byte)
	for k, v := range data {
		byteData[k] = []byte(v)
	}
	return p.CreateSecret(ctx, name, byteData)
}

func (p *KubernetesProvider) CreateBase64Secret(
	ctx context.Context,
	name string,
	data map[string]string,
) error {
	byteData := make(map[string][]byte)
	for k, v := range data {
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return fmt.Errorf("failed to decode base64 for key %s: %w", k, err)
		}
		byteData[k] = decoded
	}
	return p.CreateSecret(ctx, name, byteData)
}
