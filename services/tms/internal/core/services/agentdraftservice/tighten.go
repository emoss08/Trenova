package agentdraftservice

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type tightenReply struct {
	Instructions string `json:"instructions"`
}

func parseTightenReply(text string) (string, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return "", errEmptyReply
	}

	var reply tightenReply
	if err := sonic.UnmarshalString(trimmed, &reply); err != nil {
		return "", fmt.Errorf("the reply was not the requested object: %w", err)
	}

	return reply.Instructions, nil
}

func checkTightened(
	original, rewritten string,
) (*serviceports.TightenedAgentInstructions, error) {
	tightened := strings.TrimSpace(rewritten)
	if tightened == "" {
		return nil, errortypes.NewBusinessError(
			"The model returned no instructions. Your instructions are unchanged.",
		)
	}
	if utf8.RuneCountInString(tightened) > agentdefinition.MaxInstructionsRunes {
		return nil, errortypes.NewBusinessError(
			"The model returned instructions longer than {0} characters. Your instructions are unchanged.",
			agentdefinition.MaxInstructionsRunes,
		)
	}

	missing, added := agentdefinition.ChangedInstructionVariables(original, tightened)
	if len(missing) > 0 {
		return nil, errortypes.NewBusinessError(
			"The tightened version dropped {0}. Your instructions are unchanged.",
			strings.Join(missing, ", "),
		)
	}
	if len(added) > 0 {
		return nil, errortypes.NewBusinessError(
			"The tightened version added {0}. Your instructions are unchanged.",
			strings.Join(added, ", "),
		)
	}

	original = strings.TrimSpace(original)

	return &serviceports.TightenedAgentInstructions{
		Instructions: tightened,
		Changed:      tightened != original,
	}, nil
}
