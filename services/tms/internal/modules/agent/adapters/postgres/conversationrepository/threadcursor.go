package conversationrepository

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	threadCursorSeparator = "."
	threadCursorResource  = "agent_threads:"
)

type threadCursor struct {
	pinned    bool
	touchedAt int64
	id        pulid.ID
}

func threadCursorAt(thread *conversation.Thread) threadCursor {
	return threadCursor{
		pinned:    thread.Pinned,
		touchedAt: max(thread.LastMessageAt, thread.CreatedAt),
		id:        thread.ID,
	}
}

func threadCursorBinding(req *conversationports.ListThreadsRequest) pagination.CursorBinding {
	return pagination.BindCursor(req.TenantInfo, threadCursorResource+req.UserID.String(), nil)
}

func (c threadCursor) encode(binding pagination.CursorBinding) string {
	pinned := "0"
	if c.pinned {
		pinned = "1"
	}

	return pagination.SealCursor(binding, []byte(strings.Join([]string{
		pinned,
		strconv.FormatInt(c.touchedAt, 10),
		c.id.String(),
	}, threadCursorSeparator)))
}

func decodeThreadCursor(binding pagination.CursorBinding, raw string) (*threadCursor, error) {
	decoded, err := pagination.OpenCursor(binding, raw)
	if err != nil {
		return nil, errBadThreadCursor()
	}

	parts := strings.Split(string(decoded), threadCursorSeparator)
	if len(parts) != 3 || (parts[0] != "0" && parts[0] != "1") {
		return nil, errBadThreadCursor()
	}

	touchedAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, errBadThreadCursor()
	}

	id, err := pulid.Parse(parts[2])
	if err != nil {
		return nil, errBadThreadCursor()
	}

	return &threadCursor{
		pinned:    parts[0] == "1",
		touchedAt: touchedAt,
		id:        id,
	}, nil
}

func errBadThreadCursor() error {
	return errortypes.NewValidationError(
		"cursor",
		errortypes.ErrInvalidFormat,
		"Cursor is not one a previous page returned",
	)
}
