package workerports

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/keelkinds"
)

type RotaAssignments struct {
	WorkerID   keelkinds.PulidIDCol
	CreatedAt  keel.Int64Col
	ArchivedAt keel.Int64Col
}
