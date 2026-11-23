package employment

import (
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/shopspring/decimal"
)

type EffectKind string

const (
	EffectDispatchRemoved         = EffectKind("DispatchRemoved")
	EffectDispatchRestored        = EffectKind("DispatchRestored")
	EffectPTOPolicyEnded          = EffectKind("PtoPolicyEnded")
	EffectPTOPaidOut              = EffectKind("PtoPaidOut")
	EffectPTOForfeited            = EffectKind("PtoForfeited")
	EffectTimeOffCancelled        = EffectKind("TimeOffCancelled")
	EffectPayAssignmentEnded      = EffectKind("PayAssignmentEnded")
	EffectPortalAccessRevoked     = EffectKind("PortalAccessRevoked")
	EffectChecklistStarted        = EffectKind("ChecklistStarted")
	EffectChecklistClosed         = EffectKind("ChecklistClosed")
	EffectTrainingAssigned        = EffectKind("TrainingAssigned")
	EffectDefaultPTOPolicyApplied = EffectKind("DefaultPtoPolicyApplied")
)

func (k EffectKind) String() string { return string(k) }

type Effect struct {
	Kind      EffectKind
	Name      *string
	Count     *int
	Days      *decimal.Decimal
	PTOType   *worker.PTOType
	StartDate *int64
	EndDate   *int64
	Failed    bool
}
