package artifactrecorder

import (
	"errors"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
)

var (
	ErrUnknownBoard error = &agentruntimeports.PublishMistake{
		Reason: "there is no board with that boardId in this conversation; leave boardId out " +
			"to show a new one",
	}
	ErrNoBoards   = errors.New("boards cannot be shown here")
	errBoardStore = errors.New("the board could not be saved")
)
