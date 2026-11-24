package driversettlementboardreader

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/settlementboard"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/settlementreleaseports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const slot = "{0}"

var settlementKind = recordkind.Kind(permission.ResourceDriverSettlement)

type Params struct {
	fx.In

	Release settlementreleaseports.Reader
}

type Settlements struct{ release settlementreleaseports.Reader }

type NetPay struct{ release settlementreleaseports.Reader }

func NewSettlements(p Params) *Settlements { return &Settlements{release: p.Release} }

func NewNetPay(p Params) *NetPay { return &NetPay{release: p.Release} }

func (*Settlements) Kind() recordkind.Kind { return settlementKind }
func (*NetPay) Kind() recordkind.Kind      { return settlementKind }
func (*NetPay) Measure() string            { return settlementboard.MeasureNetPay }

func (r *Settlements) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	rows, err := r.release.Settlements(ctx, &settlementreleaseports.SettlementsRequest{
		IDs: req.IDs,
	})
	if err != nil {
		return nil, err
	}
	out := make(map[pulid.ID]boardreading.Summary, len(rows))
	for _, row := range rows {
		status, tone := settlementStatus(ctx, row.Status)
		out[row.ID] = boardreading.Summary{
			Label:      strings.TrimSpace(row.SettlementNumber),
			Subtitle:   row.Driver,
			Status:     status,
			StatusTone: tone,
		}
	}

	return out, nil
}

func (r *NetPay) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	rows, err := r.release.Settlements(ctx, &settlementreleaseports.SettlementsRequest{
		IDs:      req.IDs,
		Disputes: req.Place,
	})
	if err != nil {
		return nil, err
	}
	out := make(map[pulid.ID]boardreading.Reading, len(rows))
	for _, row := range rows {
		out[row.ID] = netPay(ctx, row, req.Location)
	}

	return out, nil
}

func netPay(
	ctx context.Context,
	row *settlementreleaseports.Settlement,
	loc *time.Location,
) boardreading.Reading {
	value := boardcatalog.Money(row.NetPayMinor, row.Currency)
	reading := boardreading.Reading{
		Value:    value,
		Standing: boardreading.Words{Text: slot, Value: &value},
	}
	if row.Status != driversettlement.StatusPaid && row.Status != driversettlement.StatusVoided {
		reading.Fact = &boardreading.Words{
			Text: i18n.T(ctx, "Pays {0}"),
			At:   settlementreleaseports.Day(row.PayDate, loc).Unix(),
			Day:  true,
		}
	}
	switch {
	case row.Has(driversettlement.ExceptionCodeMissingPayProfile):
		reading.Standing = boardreading.Words{Text: i18n.T(ctx, "Not priced")}
		reading.Tone = boardcatalog.ToneWarn
	case row.CarryForwardOutMinor < 0:
		carried := boardcatalog.Money(row.CarryForwardOutMinor, row.Currency)
		reading.Standing = boardreading.Words{
			Text:  i18n.T(ctx, "{0} carries forward"),
			Value: &carried,
		}
		reading.Tone = boardcatalog.ToneWarn
	}
	if row.Block == driversettlement.ReleaseBlockHeld {
		reading.Tone = boardcatalog.ToneWarn
		reading.Fact = &boardreading.Words{
			Text: slot,
			Value: &boardcatalog.Value{
				Kind: boardcatalog.ValueText,
				Text: settlementreleaseports.HoldWords(ctx, row.HoldReason, row.HoldNote),
			},
		}

		return reading
	}
	if exception, critical := row.CriticalException(); critical &&
		row.Block == driversettlement.ReleaseBlockCriticalException {
		message := strings.TrimSpace(exception.Message)
		reading.Tone = boardcatalog.ToneWarn
		reading.Fact = &boardreading.Words{Text: settlementreleaseports.BlockWords(ctx, row.Block)}
		if message != "" {
			reading.Fact = &boardreading.Words{
				Text: slot,
				Value: &boardcatalog.Value{
					Kind: boardcatalog.ValueText,
					Text: i18n.T(ctx, "Has a critical exception: {0}", message),
				},
			}
		}

		return reading
	}
	if row.Block == driversettlement.ReleaseBlockDisputeOpen {
		reading.Tone = boardcatalog.ToneWarn
		reading.Fact = &boardreading.Words{
			Text:  settlementreleaseports.BlockWords(ctx, row.Block),
			Place: settlementreleaseports.DisputeAbout(ctx, row.DisputeCategory),
		}
	}

	return reading
}

func settlementStatus(
	ctx context.Context,
	status driversettlement.Status,
) (string, boardcatalog.Tone) {
	switch status {
	case driversettlement.StatusDraft:
		return i18n.T(ctx, "Draft"), ""
	case driversettlement.StatusPendingApproval:
		return i18n.T(ctx, "Waiting on approval"), boardcatalog.ToneWarn
	case driversettlement.StatusApproved:
		return i18n.T(ctx, "Approved"), boardcatalog.ToneInfo
	case driversettlement.StatusPosted:
		return i18n.T(ctx, "Posted, not paid"), boardcatalog.ToneInfo
	case driversettlement.StatusPaid:
		return i18n.T(ctx, "Paid"), boardcatalog.ToneOK
	case driversettlement.StatusVoided:
		return i18n.T(ctx, "Voided"), ""
	}

	return string(status), ""
}
