package securityaudittest

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/stretchr/testify/require"
)

type Recorder struct {
	mu      sync.Mutex
	changes []services.SecurityChange
}

func (r *Recorder) RecordChange(_ context.Context, change services.SecurityChange) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.changes = append(r.changes, change)
}

func (r *Recorder) Changes() []services.SecurityChange {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]services.SecurityChange(nil), r.changes...)
}

func (r *Recorder) Only(t *testing.T) services.SecurityChange {
	t.Helper()
	changes := r.Changes()
	require.Len(t, changes, 1)
	return changes[0]
}
