package recordlabelrepository

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry/journalentrydb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordnumberports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission/factoringsubmissiondb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor/tractordb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer/trailerdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement/carriersettlementdb"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement/driversettlementdb"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type numberSource struct {
	numbering recordnumberports.Numbering
	foldsCase bool
	query     func(numbers []string) projection
}

type numberRow struct {
	ID     pulid.ID
	Number string
}

var numberSources = map[recordkind.Kind]numberSource{
	recordkind.Kind(permission.ResourceShipment): {
		numbering: recordnumberports.Numbering{Field: "proNumber", Name: "PRO number"},
		query: func(numbers []string) projection {
			t := shipmentdb.Shipments
			return t.Pick(t.ID, t.ProNumber).Where(t.ProNumber.In(numbers...)).Limit(len(numbers))
		},
	},
	recordkind.Kind(permission.ResourceOrder): {
		numbering: recordnumberports.Numbering{Field: "orderNumber", Name: "order number"},
		foldsCase: true,
		query: func(numbers []string) projection {
			t := orderdb.Orders
			return t.Pick(t.ID, t.OrderNumber).
				Where(t.OrderNumber.Lower().In(numbers...)).
				Limit(len(numbers))
		},
	},
	recordkind.Kind(permission.ResourceInvoice): {
		numbering: recordnumberports.Numbering{Field: "number", Name: "invoice number"},
		query: func(numbers []string) projection {
			t := invoicedb.Invoices
			return t.Pick(t.ID, t.Number).Where(t.Number.In(numbers...)).Limit(len(numbers))
		},
	},
	recordkind.BillingQueueItem: {
		numbering: recordnumberports.Numbering{Field: "number", Name: "billing queue number"},
		query: func(numbers []string) projection {
			t := billingqueuedb.BillingQueueItems
			return t.Pick(t.ID, t.Number).Where(t.Number.In(numbers...)).Limit(len(numbers))
		},
	},
	recordkind.Kind(permission.ResourceFactoringSubmission): {
		numbering: recordnumberports.Numbering{Field: "number", Name: "submission number"},
		query: func(numbers []string) projection {
			t := factoringsubmissiondb.Submissions
			return t.Pick(t.ID, t.Number).Where(t.Number.In(numbers...)).Limit(len(numbers))
		},
	},
	recordkind.Kind(permission.ResourceServiceFailure): {
		numbering: recordnumberports.Numbering{Field: "number", Name: "service failure number"},
		query: func(numbers []string) projection {
			t := servicefailuredb.ServiceFailures
			return t.Pick(t.ID, t.Number).Where(t.Number.In(numbers...)).Limit(len(numbers))
		},
	},
	recordkind.Kind(permission.ResourceJournalEntry): {
		numbering: recordnumberports.Numbering{Field: "entryNumber", Name: "entry number"},
		query: func(numbers []string) projection {
			t := journalentrydb.JournalEntries
			return t.Pick(t.ID, t.EntryNumber).
				Where(t.EntryNumber.In(numbers...)).
				Limit(len(numbers))
		},
	},
	recordkind.Kind(permission.ResourceDriverSettlement): {
		numbering: recordnumberports.Numbering{
			Field: "settlementNumber",
			Name:  "settlement number",
		},
		query: func(numbers []string) projection {
			t := driversettlementdb.Settlements
			return t.Pick(t.ID, t.SettlementNumber).
				Where(t.SettlementNumber.In(numbers...)).
				Limit(len(numbers))
		},
	},
	recordkind.Kind(permission.ResourceCarrierSettlement): {
		numbering: recordnumberports.Numbering{
			Field: "settlementNumber",
			Name:  "settlement number",
		},
		query: func(numbers []string) projection {
			t := carriersettlementdb.CarrierSettlements
			return t.Pick(t.ID, t.SettlementNumber).
				Where(t.SettlementNumber.In(numbers...)).
				Limit(len(numbers))
		},
	},
	recordkind.Kind(permission.ResourceTractor): {
		numbering: recordnumberports.Numbering{Field: "code", Name: "unit number"},
		foldsCase: true,
		query: func(numbers []string) projection {
			t := tractordb.Tractors
			return t.Pick(t.ID, t.Code).Where(t.Code.Lower().In(numbers...)).Limit(len(numbers))
		},
	},
	recordkind.Kind(permission.ResourceTrailer): {
		numbering: recordnumberports.Numbering{Field: "code", Name: "unit number"},
		foldsCase: true,
		query: func(numbers []string) projection {
			t := trailerdb.Trailers
			return t.Pick(t.ID, t.Code).Where(t.Code.Lower().In(numbers...)).Limit(len(numbers))
		},
	},
}

func NewRecordNumbers(p Params) recordnumberports.RecordNumberReader {
	return &Labeler{db: p.DB}
}

func (l *Labeler) Numbering(kind recordkind.Kind) (recordnumberports.Numbering, bool) {
	src, ok := numberSources[kind]

	return src.numbering, ok
}

func (l *Labeler) FindNumbers(
	ctx context.Context,
	_ pagination.TenantInfo,
	numbers map[recordkind.Kind][]string,
) (recordnumberports.NumberedRecords, error) {
	kinds := make([]recordkind.Kind, 0, len(numbers))
	for kind := range numbers {
		if _, ok := numberSources[kind]; ok {
			kinds = append(kinds, kind)
		}
	}
	if len(kinds) == 0 {
		return recordnumberports.NumberedRecords{}, nil
	}
	slices.Sort(kinds)

	return dbtx.Read(ctx, l.db, func(ctx context.Context) (recordnumberports.NumberedRecords, error) {
		found := make(recordnumberports.NumberedRecords, len(kinds))
		for _, kind := range kinds {
			src := numberSources[kind]
			asked := distinctNumbers(numbers[kind], src.foldsCase)
			if len(asked) == 0 {
				continue
			}
			rows, err := keel.Into[numberRow](ctx, l.db.Keel(ctx), src.query(asked))
			if err != nil {
				return nil, fmt.Errorf("read %s numbers: %w", kind, err)
			}
			if matched := matchNumbers(numbers[kind], rows, src.foldsCase); len(matched) > 0 {
				found[kind] = matched
			}
		}

		return found, nil
	})
}

func distinctNumbers(numbers []string, foldsCase bool) []string {
	seen := make(map[string]struct{}, len(numbers))
	out := make([]string, 0, min(len(numbers), MaxIDsPerResource))
	for _, number := range numbers {
		number = strings.TrimSpace(number)
		if foldsCase {
			number = strings.ToLower(number)
		}
		if number == "" {
			continue
		}
		if _, dup := seen[number]; dup {
			continue
		}
		seen[number] = struct{}{}
		out = append(out, number)
		if len(out) == MaxIDsPerResource {
			break
		}
	}

	return out
}

func matchNumbers(
	asked []string,
	rows []numberRow,
	foldsCase bool,
) map[string][]recordnumberports.NumberedRecord {
	matched := make(map[string][]recordnumberports.NumberedRecord, len(asked))
	for _, number := range asked {
		if _, done := matched[number]; done {
			continue
		}
		for _, row := range rows {
			same := row.Number == number
			if foldsCase {
				same = strings.EqualFold(row.Number, number)
			}
			if same {
				matched[number] = append(matched[number], recordnumberports.NumberedRecord{
					ID:     row.ID,
					Number: row.Number,
				})
			}
		}
	}

	return matched
}
