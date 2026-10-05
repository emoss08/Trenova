package aitraining

import (
	"github.com/emoss08/trenova/internal/cloud/aitraining/aitrainingjobs"
	"github.com/emoss08/trenova/internal/cloud/aitraining/aitrainingrepository"
	"github.com/emoss08/trenova/internal/cloud/aitraining/aitrainingservice"
	"github.com/emoss08/trenova/internal/cloud/aitraining/retrainingalert"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
)

var Module = fx.Options(
	fx.Provide(
		aitrainingrepository.NewExports,
		aitrainingrepository.NewRecords,
		aitrainingrepository.NewCycles,
	),
	aitrainingservice.Module,
	aitrainingjobs.Module,
	retrainingalert.Module,
	fx.Decorate(DecorateHistory),
)

type HistoryParams struct {
	fx.In

	Default services.AITrainingHistoryService
	History aitrainingservice.HistoryParams
}

func DecorateHistory(p HistoryParams) services.AITrainingHistoryService {
	return aitrainingservice.NewHistory(p.History)
}
