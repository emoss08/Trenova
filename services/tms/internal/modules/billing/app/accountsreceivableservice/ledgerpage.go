package accountsreceivableservice

import (
	"context"
	"encoding/binary"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	DefaultLedgerPageSize = 100
	MaxLedgerPageSize     = 500

	ledgerCursorResource = "ar.customer-ledger"
)

type ledgerCursor struct {
	through  int64
	balance  int64
	position customerledgerports.ARLedgerPosition
}

func (s *Service) ListCustomerLedger(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	req customerledgerports.ARLedgerPageRequest,
) (*customerledgerports.ARLedgerPage, error) {
	if err := checkLedgerWindow(req); err != nil {
		return nil, err
	}
	binding := ledgerCursorBinding(tenantInfo, req)

	window := customerledgerports.ListCustomerLedgerWindowRequest{
		TenantInfo: tenantInfo,
		CustomerID: req.CustomerID,
		From:       req.StartDate,
		Through:    req.EndDate,
		Limit:      ledgerPageSize(req.First) + 1,
	}
	if window.Through == 0 {
		window.Through = timeutils.NowUnix()
	}

	opening := int64(0)
	if req.After != "" {
		cursor, err := openLedgerCursor(binding, req.After)
		if err != nil {
			return nil, err
		}
		window.Through = cursor.through
		window.After = &cursor.position
		opening = cursor.balance
	} else if req.StartDate > 0 {
		balance, err := s.repo.GetCustomerLedgerBalance(
			ctx,
			customerledgerports.GetCustomerLedgerBalanceRequest{
				TenantInfo: tenantInfo,
				CustomerID: req.CustomerID,
				Before:     req.StartDate,
			},
		)
		if err != nil {
			return nil, err
		}
		opening = balance
	}

	entries, err := s.repo.ListCustomerLedgerWindow(ctx, window)
	if err != nil {
		return nil, err
	}

	page := &customerledgerports.ARLedgerPage{OpeningBalanceMinor: opening}
	if size := window.Limit - 1; len(entries) > size {
		entries = entries[:size]
		page.HasNextPage = true
	}

	running := opening
	for _, entry := range entries {
		running += entry.AmountMinor
		entry.RunningBalanceMinor = running
	}
	page.Entries = entries

	if page.HasNextPage {
		last := entries[len(entries)-1]
		page.EndCursor = sealLedgerCursor(binding, ledgerCursor{
			through:  window.Through,
			balance:  running,
			position: last.Position,
		})
	}
	return page, nil
}

func checkLedgerWindow(req customerledgerports.ARLedgerPageRequest) error {
	switch {
	case req.CustomerID.IsNil():
		return errortypes.NewValidationError(
			"customerId", errortypes.ErrRequired, "Customer is required",
		)
	case req.StartDate < 0 || req.EndDate < 0:
		return errortypes.NewValidationError(
			"startDate", errortypes.ErrInvalid, "Ledger dates cannot be negative",
		)
	case req.EndDate > 0 && req.StartDate > req.EndDate:
		return errortypes.NewValidationError(
			"startDate", errortypes.ErrInvalid, "Start the ledger on or before its end date",
		)
	case req.First < 0:
		return errortypes.NewValidationError(
			"first", errortypes.ErrInvalid, "Page size cannot be negative",
		)
	}
	return nil
}

func ledgerPageSize(first int) int {
	if first == 0 {
		return DefaultLedgerPageSize
	}
	return min(first, MaxLedgerPageSize)
}

func ledgerCursorBinding(
	tenantInfo pagination.TenantInfo,
	req customerledgerports.ARLedgerPageRequest,
) pagination.CursorBinding {
	var query strings.Builder
	query.WriteString(ledgerCursorResource)
	for _, part := range []string{
		req.CustomerID.String(),
		strconv.FormatInt(req.StartDate, 10),
		strconv.FormatInt(req.EndDate, 10),
	} {
		query.WriteByte(0)
		query.WriteString(part)
	}
	return pagination.CursorBinding{
		OrgID: tenantInfo.OrgID,
		BuID:  tenantInfo.BuID,
		Query: query.String(),
	}
}

func sealLedgerCursor(binding pagination.CursorBinding, cursor ledgerCursor) string {
	payload := binary.AppendVarint(nil, cursor.through)
	payload = binary.AppendVarint(payload, cursor.balance)
	payload = binary.AppendVarint(payload, cursor.position.TransactionDate)
	payload = binary.AppendVarint(payload, cursor.position.LineNumber)
	payload = append(payload, cursor.position.ID...)
	return pagination.SealCursor(binding, payload)
}

func openLedgerCursor(binding pagination.CursorBinding, encoded string) (ledgerCursor, error) {
	raw, err := pagination.OpenCursor(binding, encoded)
	if err != nil {
		return ledgerCursor{}, invalidLedgerCursor()
	}

	values := make([]int64, 4)
	for i := range values {
		value, read := binary.Varint(raw)
		if read <= 0 {
			return ledgerCursor{}, invalidLedgerCursor()
		}
		values[i] = value
		raw = raw[read:]
	}
	id := pulid.ID(raw)
	if id.IsNil() {
		return ledgerCursor{}, invalidLedgerCursor()
	}

	return ledgerCursor{
		through: values[0],
		balance: values[1],
		position: customerledgerports.ARLedgerPosition{
			TransactionDate: values[2],
			LineNumber:      values[3],
			ID:              id,
		},
	}, nil
}

func invalidLedgerCursor() error {
	return errortypes.NewValidationError("after", errortypes.ErrInvalidFormat, "Cursor is invalid")
}
