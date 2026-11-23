package desktranscript

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type ConversationFilter struct {
	Timeframe         Timeframe                   `json:"timeframe"`
	UserID            pulid.ID                    `json:"userId,omitempty"`
	AgentDefinitionID pulid.ID                    `json:"agentDefinitionId,omitempty"`
	Origins           []conversation.ThreadOrigin `json:"origins,omitempty"`
}

func (f *ConversationFilter) Normalize() {
	if f.Timeframe.Preset == "" {
		f.Timeframe = AllTime()
	}
	origins := slices.Clone(f.Origins)
	slices.Sort(origins)
	f.Origins = slices.Compact(origins)
}

func (f *ConversationFilter) Validate(field string, multiErr *errortypes.MultiError) {
	f.Timeframe.Validate(field+".timeframe", multiErr)
	for _, origin := range f.Origins {
		if !origin.IsValid() {
			multiErr.Add(
				field+".origins",
				errortypes.ErrInvalid,
				"Where a conversation began is invalid",
			)
			break
		}
	}
}

func (f *ConversationFilter) Key() string {
	origins := make([]string, len(f.Origins))
	for idx, origin := range f.Origins {
		origins[idx] = string(origin)
	}

	return strings.Join([]string{
		string(f.Timeframe.Preset),
		f.Timeframe.FromDate,
		f.Timeframe.ToDate,
		f.UserID.String(),
		f.AgentDefinitionID.String(),
		strings.Join(origins, ","),
	}, "|")
}
