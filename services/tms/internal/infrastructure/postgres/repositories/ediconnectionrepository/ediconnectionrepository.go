package ediconnectionrepository

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/edirepository"
)

type Params = edirepository.Params

func New(p Params) repositories.EDIConnectionRepository {
	return edirepository.NewConnectionRepository(p)
}
