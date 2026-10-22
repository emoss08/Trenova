package formulalibraryservice

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
)

type TestSourceRequest struct {
	TenantInfo pagination.TenantInfo
	Source     string
}

type TestSourceResult struct {
	Check   *rules.TariffCheck   `json:"check"`
	Imports []tarifftypes.Import `json:"imports"`
	Tests   *rules.TariffTestRun `json:"tests"`
}

func (s *Service) TestSource(
	ctx context.Context,
	req *TestSourceRequest,
) (*TestSourceResult, error) {
	if len(req.Source) > tarifftypes.MaxSourceLength {
		return nil, errortypes.NewValidationError(
			fieldSource,
			errortypes.ErrInvalid,
			"Source cannot exceed 64 KiB",
		)
	}

	content, err := s.checkSource(ctx, req.TenantInfo, req.Source)
	if err != nil {
		return nil, err
	}

	return &TestSourceResult{
		Check:   content.checked.Check,
		Imports: orEmpty(content.checked.Imports),
		Tests:   content.tests,
	}, nil
}
