// Package extractionaccuracyports reports how well production document extraction reads
// documents, measured from the corrections people make when they confirm a draft.
package extractionaccuracyports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ExtractionAccuracyRequest struct {
	TenantInfo pagination.TenantInfo
	WindowDays int
}

type ExtractionAccuracyGroup struct {
	Key         string
	Corrections int
	Scored      int
	Correct     int
	Accuracy    float64
}

type ExtractionAccuracy struct {
	WindowDays  int
	Since       int64
	Corrections int
	Sampled     bool
	Scored      int
	Correct     int
	Corrected   int
	Missed      int
	Unconfirmed int
	Accuracy    float64
	ByModel     []ExtractionAccuracyGroup
	ByKind      []ExtractionAccuracyGroup
	Fields      []aicorrection.FieldAccuracy
}

const ExtractionAccuracyDriftEvent = "ai.extraction_accuracy_drift"

type ExtractionProviderTrend struct {
	ProviderID      pulid.ID
	ProviderName    string
	Model           string
	ProviderRemoved bool
	Weeks           []aicorrection.WeekAccuracy
	Checked         aicorrection.WeekAccuracy
	Baseline        aicorrection.WeekAccuracy
	DropPoints      float64
	Comparable      bool
	Drifting        bool
}

type ExtractionProviderTrends struct {
	Weeks             []int64
	CheckedWeek       int64
	BaselineStart     int64
	Providers         []ExtractionProviderTrend
	DriftPoints       int
	MinWeekFields     int
	MinBaselineFields int
}

type ExtractionAccuracyService interface {
	ListCorrections(
		ctx context.Context,
		req *aicorrectionports.ListAICorrectionConnectionRequest,
	) (*pagination.CursorListResult[*aicorrection.Correction], error)
	GetCorrection(
		ctx context.Context,
		req aicorrectionports.GetAICorrectionRequest,
	) (*aicorrection.Correction, error)
	Accuracy(ctx context.Context, req *ExtractionAccuracyRequest) (*ExtractionAccuracy, error)
	ProviderTrends(
		ctx context.Context,
		tenant pagination.TenantInfo,
	) (*ExtractionProviderTrends, error)
}

type ExtractionDriftCheckRequest struct {
	TenantInfo pagination.TenantInfo
	Now        int64
}

// ExtractionDriftChecker tells an organization when a provider read last week's documents
// worse than it did over the weeks before.
type ExtractionDriftChecker interface {
	CheckDrift(ctx context.Context, req *ExtractionDriftCheckRequest) (int, error)
}
