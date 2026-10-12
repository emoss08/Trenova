package securityaudittest

import (
	"context"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/stretchr/testify/require"
)

type Recorder struct {
	mu      sync.Mutex
	changes []permissionports.SecurityChange
}

func (r *Recorder) RecordChange(_ context.Context, change *permissionports.SecurityChange) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.changes = append(r.changes, *change)
}

func (r *Recorder) Changes() []permissionports.SecurityChange {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]permissionports.SecurityChange(nil), r.changes...)
}

func (r *Recorder) Only(t *testing.T) permissionports.SecurityChange {
	t.Helper()
	changes := r.Changes()
	require.Len(t, changes, 1)
	return changes[0]
}
