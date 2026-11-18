package documentmeter_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/documentmeter"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/datausageports"
	"github.com/stretchr/testify/assert"
)

func TestTheDataUsageFigureIsTheStorageMeterItself(t *testing.T) {
	t.Parallel()

	got := documentmeter.New(documentmeter.Params{})

	assert.Same(t, got.Reader, got.Source, "one reader, so the meter and the figure never disagree")
	assert.Equal(t, platformcatalog.MeterDocumentStorageBytes, got.Reader.Meter())
	assert.Equal(t, datausageports.FigureDocuments, got.Source.Figure())
}
