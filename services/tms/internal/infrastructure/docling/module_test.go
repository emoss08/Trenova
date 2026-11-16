package docling

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/documentconversionconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

type silentConverter struct {
	kernelports.DocumentConverter
	asked   chan struct{}
	release chan struct{}
}

func (c *silentConverter) Enabled() bool { return true }

func (c *silentConverter) Healthy(context.Context) error {
	close(c.asked)
	<-c.release
	return kernelports.ErrDocumentConverterUnavailable
}

func TestAConverterThatDoesNotAnswerNeverHoldsStartup(t *testing.T) {
	t.Parallel()

	converter := &silentConverter{asked: make(chan struct{}), release: make(chan struct{})}
	t.Cleanup(func() { close(converter.release) })
	lc := fxtest.NewLifecycle(t)
	registerHealthProbe(lc, converter, &documentconversionconfig.Config{}, zap.NewNop())

	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	began := time.Now()
	require.NoError(t, lc.Start(ctx))
	require.Less(t, time.Since(began), 500*time.Millisecond,
		"startup waited for a converter that was not answering")

	select {
	case <-converter.asked:
	case <-time.After(time.Second):
		t.Fatal("the converter was never asked whether it is healthy")
	}
}
