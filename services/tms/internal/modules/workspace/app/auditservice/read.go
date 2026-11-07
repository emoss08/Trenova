package auditservice

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/identity/domain/operationbits"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"go.uber.org/zap"
)

var clientBitToOperation = map[int64]permission.Operation{
	int64(operationbits.Read):      permission.OpRead,
	int64(operationbits.Create):    permission.OpCreate,
	int64(operationbits.Update):    permission.OpUpdate,
	int64(operationbits.Delete):    permission.OpDelete,
	int64(operationbits.Export):    permission.OpExport,
	int64(operationbits.Import):    permission.OpImport,
	int64(operationbits.Approve):   permission.OpApprove,
	int64(operationbits.Reject):    permission.OpReject,
	int64(operationbits.Assign):    permission.OpAssign,
	int64(operationbits.Unassign):  permission.OpUnassign,
	int64(operationbits.Archive):   permission.OpArchive,
	int64(operationbits.Restore):   permission.OpRestore,
	int64(operationbits.Submit):    permission.OpSubmit,
	int64(operationbits.Cancel):    permission.OpCancel,
	int64(operationbits.Duplicate): permission.OpDuplicate,
	int64(operationbits.Close):     permission.OpClose,
	int64(operationbits.Lock):      permission.OpLock,
	int64(operationbits.Unlock):    permission.OpUnlock,
	int64(operationbits.Activate):  permission.OpActivate,
	int64(operationbits.Reopen):    permission.OpReopen,
}

func (s *service) ListConnection(
	ctx context.Context,
	req *auditports.ListAuditEntriesRequest,
) (*pagination.CursorListResult[*audit.Entry], error) {
	normalizeOperationFilters(req.Filter)

	entities, err := s.repo.ListConnection(ctx, req)
	if err != nil {
		s.logger.Error("failed to list audit entries", zap.Error(err))
		return nil, fmt.Errorf("failed to list audit entries: %w", err)
	}

	return entities, nil
}

func (s *service) GetByID(
	ctx context.Context,
	req *auditports.GetAuditEntryByIDOptions,
) (*audit.Entry, error) {
	entity, err := s.repo.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}

	return entity, nil
}

func normalizeOperationFilters(opts *pagination.QueryOptions) {
	if opts == nil {
		return
	}

	normalizeOperationFieldFilters(opts.FieldFilters)
	for idx := range opts.FilterGroups {
		normalizeOperationFieldFilters(opts.FilterGroups[idx].Filters)
	}
}

func normalizeOperationFieldFilters(filters []domaintypes.FieldFilter) {
	for idx := range filters {
		if filters[idx].Field != "operation" {
			continue
		}

		filters[idx].Value = normalizeOperationFilterValue(filters[idx].Value)
	}
}

func normalizeOperationFilterValue(value any) any {
	switch v := value.(type) {
	case []string:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if normalized, ok := toOperationString(item); ok {
				out = append(out, normalized)
			}
		}
		return out
	case []int:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if normalized, ok := toOperationString(item); ok {
				out = append(out, normalized)
			}
		}
		return out
	case []int64:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if normalized, ok := toOperationString(item); ok {
				out = append(out, normalized)
			}
		}
		return out
	case []float64:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if normalized, ok := toOperationString(item); ok {
				out = append(out, normalized)
			}
		}
		return out
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if normalized, ok := toOperationString(item); ok {
				out = append(out, normalized)
			}
		}
		return out
	default:
		if normalized, ok := toOperationString(value); ok {
			return normalized
		}
		return value
	}
}

//nolint:cyclop // existing business workflow intentionally has several domain branches
func toOperationString(value any) (string, bool) {
	switch v := value.(type) {
	case permission.Operation:
		return string(v), true
	case string:
		normalized := strings.ToLower(strings.TrimSpace(v))
		if normalized == "" {
			return "", false
		}

		if numeric, err := strconv.ParseInt(normalized, 10, 64); err == nil {
			if op, ok := clientBitToOperation[numeric]; ok {
				return string(op), true
			}
			return strconv.FormatInt(numeric, 10), true
		}

		return normalized, true
	case int:
		return normalizeOperationFromInt64(int64(v))
	case int8:
		return normalizeOperationFromInt64(int64(v))
	case int16:
		return normalizeOperationFromInt64(int64(v))
	case int32:
		return normalizeOperationFromInt64(int64(v))
	case int64:
		return normalizeOperationFromInt64(v)
	case uint:
		return normalizeOperationFromUint64(uint64(v))
	case uint8:
		return normalizeOperationFromUint64(uint64(v))
	case uint16:
		return normalizeOperationFromUint64(uint64(v))
	case uint32:
		return normalizeOperationFromUint64(uint64(v))
	case uint64:
		return normalizeOperationFromUint64(v)
	case float32:
		if math.Trunc(float64(v)) != float64(v) {
			return strconv.FormatFloat(float64(v), 'f', -1, 32), true
		}
		return normalizeOperationFromInt64(int64(v))
	case float64:
		if math.Trunc(v) != v {
			return strconv.FormatFloat(v, 'f', -1, 64), true
		}
		return normalizeOperationFromInt64(int64(v))
	default:
		return "", false
	}
}

func normalizeOperationFromInt64(value int64) (string, bool) {
	if op, ok := clientBitToOperation[value]; ok {
		return string(op), true
	}

	return strconv.FormatInt(value, 10), true
}

func normalizeOperationFromUint64(value uint64) (string, bool) {
	n, err := intutils.SafeUint64ToInt64(value)
	if err != nil {
		return strconv.FormatUint(value, 10), true
	}

	return normalizeOperationFromInt64(n)
}
