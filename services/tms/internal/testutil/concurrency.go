package testutil

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/require"
)

type TxLockHandle struct {
	locked  chan struct{}
	release chan struct{}
	done    chan error
}

func HoldTxLock(
	t *testing.T,
	conn *postgres.Connection,
	opts kernelports.TxOptions,
	fn func(context.Context) error,
) *TxLockHandle {
	t.Helper()

	handle := &TxLockHandle{
		locked:  make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan error, 1),
	}

	go func() {
		err := conn.Transact(context.Background(), opts, func(ctx context.Context) error {
			if err := fn(ctx); err != nil {
				return err
			}

			close(handle.locked)
			<-handle.release
			return nil
		})
		handle.done <- err
		close(handle.done)
	}()

	return handle
}

func (h *TxLockHandle) WaitLocked(t *testing.T) {
	t.Helper()

	select {
	case <-h.locked:
	case err := <-h.done:
		require.NoError(t, err)
		t.Fatal("transaction finished before lock was acquired")
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for transaction lock")
	}
}

func (h *TxLockHandle) Release() {
	close(h.release)
}

func (h *TxLockHandle) Wait(t *testing.T) {
	t.Helper()

	select {
	case err := <-h.done:
		require.NoError(t, err)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for transaction completion")
	}
}
