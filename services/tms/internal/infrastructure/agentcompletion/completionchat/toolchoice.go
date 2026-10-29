package completionchat

import (
	"errors"
	"net/http"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/agentcompletion/modeladapter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
)

const toolChoiceField = "tool_choice"

func refusedToolChoice(
	provider *aiprovider.Provider,
	req *modeladapter.Request,
	err error,
) bool {
	if err == nil || !req.ToolChoiceNone || len(req.Tools) == 0 ||
		provider.Kind == aiprovider.KindAnthropicMessages {
		return false
	}

	var transport *modeladapter.TransportError
	if !errors.As(err, &transport) {
		return false
	}
	if transport.StatusCode != http.StatusBadRequest &&
		transport.StatusCode != http.StatusUnprocessableEntity {
		return false
	}

	return strings.Contains(strings.ToLower(transport.Message), toolChoiceField)
}
