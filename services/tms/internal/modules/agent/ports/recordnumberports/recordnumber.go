package recordnumberports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type Numbering struct {
	Field string
	Name  string
}

type NumberedRecord struct {
	ID     pulid.ID
	Number string
}

type NumberedRecords map[recordkind.Kind]map[string][]NumberedRecord

type RecordNumberReader interface {
	Numbering(kind recordkind.Kind) (Numbering, bool)
	FindNumbers(
		ctx context.Context,
		tenant pagination.TenantInfo,
		numbers map[recordkind.Kind][]string,
	) (NumberedRecords, error)
}
