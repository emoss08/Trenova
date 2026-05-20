package editransferrepository

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/edirepository"
)

type Params = edirepository.Params

func New(p Params) repositories.EDILoadTenderTransferRepository {
	return edirepository.NewTransferRepository(p)
}
