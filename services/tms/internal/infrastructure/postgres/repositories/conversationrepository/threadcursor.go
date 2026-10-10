package conversationrepository

import (
	"encoding/base64"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const threadCursorSeparator = "."

type threadCursor struct {
	pinned        bool
	lastMessageAt int64
	createdAt     int64
	id            pulid.ID
}

func threadCursorAt(thread *conversation.Thread) threadCursor {
	return threadCursor{
		pinned:        thread.Pinned,
		lastMessageAt: thread.LastMessageAt,
		createdAt:     thread.CreatedAt,
		id:            thread.ID,
	}
}

func (c threadCursor) encode() string {
	pinned := "0"
	if c.pinned {
		pinned = "1"
	}

	return base64.RawURLEncoding.EncodeToString([]byte(strings.Join([]string{
		pinned,
		strconv.FormatInt(c.lastMessageAt, 10),
		strconv.FormatInt(c.createdAt, 10),
		c.id.String(),
	}, threadCursorSeparator)))
}

func decodeThreadCursor(raw string) (*threadCursor, error) {
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, errBadThreadCursor()
	}

	parts := strings.SplitN(string(decoded), threadCursorSeparator, 4)
	if len(parts) != 4 || (parts[0] != "0" && parts[0] != "1") {
		return nil, errBadThreadCursor()
	}

	lastMessageAt, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return nil, errBadThreadCursor()
	}

	createdAt, err := strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return nil, errBadThreadCursor()
	}

	id, err := pulid.Parse(parts[3])
	if err != nil {
		return nil, errBadThreadCursor()
	}

	return &threadCursor{
		pinned:        parts[0] == "1",
		lastMessageAt: lastMessageAt,
		createdAt:     createdAt,
		id:            id,
	}, nil
}

func errBadThreadCursor() error {
	return errortypes.NewValidationError(
		"cursor",
		errortypes.ErrInvalidFormat,
		"Cursor is not one a previous page returned",
	)
}
