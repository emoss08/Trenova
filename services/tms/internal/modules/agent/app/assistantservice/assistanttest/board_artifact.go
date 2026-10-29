package assistanttest

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/domain/boardparse"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type StubBoards struct {
	Checked []*assistantports.BoardFillRequest
	Filled  []*assistantports.BoardFillRequest
	Refuse  error
}

func (s *StubBoards) Check(_ context.Context, req *assistantports.BoardFillRequest) error {
	s.Checked = append(s.Checked, req)

	return s.Refuse
}

func (s *StubBoards) Fill(
	_ context.Context,
	req *assistantports.BoardFillRequest,
) (*board.Filled, error) {
	s.Filled = append(s.Filled, req)

	return &board.Filled{
		Title:  req.Board.Title,
		ReadAt: 1_793_000_000,
		Blocks: []board.FilledBlock{{
			Type: board.BlockFigures, Numeral: 1, State: board.StateReady,
			Figures: []board.FilledFigure{{
				Label: "Late now",
				Value: &board.Value{Kind: board.ValueCount, Number: decimal.NewFromInt(6)},
			}},
		}},
	}, nil
}

func (s *StubBoards) Card(
	context.Context,
	*assistantports.BoardCardRequest,
) (*assistantports.BoardCard, error) {
	return nil, nil
}

func (s *StubBoards) Location(context.Context, pagination.TenantInfo) *time.Location {
	return time.UTC
}

func ShownBoard(t *testing.T, revises pulid.ID) agentruntimeports.ToolObservation {
	t.Helper()
	parsed, problems := boardparse.Parse(map[string]any{
		"title":   "Six late shipments",
		"verdict": "Four are [held][1].",
		"blocks": []any{map[string]any{"type": "figures", "figures": []any{map[string]any{
			"label":   "Late now",
			"records": []any{pulid.MustNew("shp_").String()},
			"measure": "lateness",
		}}}},
	})
	require.Empty(t, problems)

	return agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{ID: "call_board", Name: "show_board"},
		Data: agentruntimeports.BoardRequest{Board: *parsed, ArtifactID: revises},
	}
}
