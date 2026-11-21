package archiveimport

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
)

var errImportOnly = errors.New(
	"trenova db import-archive only loads an archive; it never takes one, so it verifies no trail to attest",
)

type importOnly struct{}

func (importOnly) VerifyThrough(
	context.Context,
	pagination.TenantInfo,
	int64,
) (*auditports.TrailCheck, error) {
	return nil, errImportOnly
}

type importOnlyAI struct{}

func (importOnlyAI) VerifyThrough(
	context.Context,
	pagination.TenantInfo,
	int64,
) (*aiauditports.TrailCheck, error) {
	return nil, errImportOnly
}
