package tms

import (
	"context"
	"fmt"
	"strings"
	"sync"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const TokenKey = "managedToken"

type SecretTokens struct {
	client    kubernetes.Interface
	namespace string
	name      string

	mu    sync.Mutex
	token string
}

type SecretTokensParams struct {
	Client    kubernetes.Interface
	Namespace string
	Name      string
}

func NewSecretTokens(p SecretTokensParams) *SecretTokens {
	return &SecretTokens{client: p.Client, namespace: p.Namespace, name: p.Name}
}

func (s *SecretTokens) Token(ctx context.Context) (string, error) {
	s.mu.Lock()
	token := s.token
	s.mu.Unlock()
	if token != "" {
		return token, nil
	}
	return s.Refresh(ctx)
}

func (s *SecretTokens) Refresh(ctx context.Context) (string, error) {
	sec, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, s.name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return "", fmt.Errorf("%w: Secret %s/%s does not exist", ErrNoToken, s.namespace, s.name)
	}
	if err != nil {
		return "", fmt.Errorf("read the Secret %s/%s: %w", s.namespace, s.name, err)
	}
	token := strings.TrimSpace(string(sec.Data[TokenKey]))
	if token == "" {
		return "", fmt.Errorf("%w: Secret %s/%s has no %s", ErrNoToken, s.namespace, s.name, TokenKey)
	}
	s.mu.Lock()
	s.token = token
	s.mu.Unlock()
	return token, nil
}
