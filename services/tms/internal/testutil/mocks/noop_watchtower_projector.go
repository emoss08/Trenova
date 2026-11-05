package mocks

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/pkg/pagination"
)

var _ watchtowerports.WatchtowerProjector = (*NoopWatchtowerProjector)(nil)

type NoopWatchtowerProjector struct{}

func (*NoopWatchtowerProjector) Upsert(context.Context, watchtowerports.WatchtowerItemInput) {}

func (*NoopWatchtowerProjector) Resolve(
	context.Context,
	pagination.TenantInfo,
	watchtower.SourceKind,
	string,
) {
}
