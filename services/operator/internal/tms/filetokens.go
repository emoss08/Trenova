package tms

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync"
)

type FileTokens struct {
	path string

	mu    sync.Mutex
	token string
}

func NewFileTokens(path string) *FileTokens {
	return &FileTokens{path: path}
}

func (f *FileTokens) Token(ctx context.Context) (string, error) {
	f.mu.Lock()
	token := f.token
	f.mu.Unlock()
	if token != "" {
		return token, nil
	}
	return f.Refresh(ctx)
}

func (f *FileTokens) Refresh(context.Context) (string, error) {
	body, err := os.ReadFile(f.path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", fmt.Errorf("%w: %s does not exist", ErrNoToken, f.path)
	}
	if err != nil {
		return "", fmt.Errorf("read the managed token: %w", err)
	}
	token := strings.TrimSpace(string(body))
	if token == "" {
		return "", fmt.Errorf("%w: %s is empty", ErrNoToken, f.path)
	}
	f.mu.Lock()
	f.token = token
	f.mu.Unlock()
	return token, nil
}
