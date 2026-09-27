package captureservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/capture"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// maxBulkFile bounds one "file all": a stack of a hundred documents is a long
// morning's scanning, and more than that in one request is somebody's loop.
const maxBulkFile = 100

type FileItemsInput struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Items      []FileItemInput       `json:"items"`
}

// FileItemFailure is one document that did not file, and why, in words the
// person can act on.
type FileItemFailure struct {
	ItemID  pulid.ID `json:"itemId"`
	Message string   `json:"message"`
}

type FileItemsResult struct {
	Filed    []*capture.CaptureItem `json:"filed"`
	Failures []FileItemFailure      `json:"failures"`
}

// FileItems files several documents at once, each exactly as FileItem would
// file it alone: checked, versioned and filed as the caller. One document that
// cannot be filed does not stop the rest; it comes back with its reason, so a
// person filing a whole stack fixes the one that failed rather than starting
// over.
func (s *Service) FileItems(ctx context.Context, in *FileItemsInput) (*FileItemsResult, error) {
	if _, err := s.require(
		ctx,
		in.TenantInfo,
		permission.ResourceCaptureBatch,
		permission.OpUpdate,
	); err != nil {
		return nil, err
	}
	if len(in.Items) == 0 || len(in.Items) > maxBulkFile {
		return nil, errortypes.NewValidationError("items", errortypes.ErrInvalid,
			"File between 1 and {0} documents at a time", maxBulkFile)
	}

	seen := make(map[pulid.ID]struct{}, len(in.Items))
	for i := range in.Items {
		if _, dup := seen[in.Items[i].ItemID]; dup {
			return nil, errortypes.NewValidationError("items", errortypes.ErrDuplicate,
				"Each document can be filed only once in a request")
		}
		seen[in.Items[i].ItemID] = struct{}{}
	}

	result := &FileItemsResult{
		Filed:    make([]*capture.CaptureItem, 0, len(in.Items)),
		Failures: make([]FileItemFailure, 0),
	}
	for i := range in.Items {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		entry := in.Items[i]
		entry.TenantInfo = in.TenantInfo
		entry.Automatic = false

		item, err := s.FileItem(ctx, &entry)
		if err != nil {
			result.Failures = append(result.Failures, FileItemFailure{
				ItemID:  entry.ItemID,
				Message: s.filingFailure(entry.ItemID, err),
			})

			continue
		}
		result.Filed = append(result.Filed, item)
	}

	return result, nil
}

// filingFailure is what a person is told about one document in a bulk filing.
// A reason the person can act on is passed through; anything else is logged
// and reported as a failure to try again, since its text is for operators.
func (s *Service) filingFailure(itemID pulid.ID, err error) string {
	var multiErr *errortypes.MultiError
	if errors.As(err, &multiErr) && multiErr.HasErrors() {
		messages := make([]string, 0, len(multiErr.Errors))
		for _, fieldErr := range multiErr.Errors {
			messages = append(messages, fieldErr.Error())
		}

		return strings.Join(messages, "; ")
	}

	switch {
	case errortypes.IsError(err),
		errortypes.IsBusinessError(err),
		errortypes.IsConflictError(err),
		errortypes.IsNotFoundError(err),
		errortypes.IsAuthorizationError(err):
		return err.Error()
	default:
		s.l.Error("capture bulk filing failed",
			zap.String("itemId", itemID.String()),
			zap.Error(err))

		return filingFailedMessage
	}
}
