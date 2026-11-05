package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type NoopWatchtowerProjector struct{}

func (p *NoopWatchtowerProjector) Upsert(
	_ context.Context,
	_ watchtowerports.WatchtowerItemInput,
) {
}

func (p *NoopWatchtowerProjector) Resolve(
	_ context.Context,
	_ pagination.TenantInfo,
	_ watchtower.SourceKind,
	_ string,
) {
}
