package boardcardservice

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/analysis"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/zap"
)

const percentPoints = 100.0

type tableViewPayload struct {
	Columns  []assistantartifact.DisplayColumn `json:"columns"`
	Rows     []map[string]any                  `json:"rows"`
	RowCount int                               `json:"rowCount"`
}

func (s *Service) artifact(
	ctx context.Context,
	req *agentports.BoardFillRequest,
	id pulid.ID,
) (*assistantartifact.Artifact, error) {
	artifact, err := s.artifacts.GetByID(ctx, assistantartifactports.GetArtifactRequest{
		ID:         id,
		TenantInfo: req.Actor.TenantInfo(),
	})
	if err != nil {
		return nil, err
	}
	if artifact.ThreadID != req.ThreadID {
		return nil, errArtifactGone
	}

	return artifact, nil
}

func (s *Service) readArtifacts(
	ctx context.Context,
	req *agentports.BoardFillRequest,
	reads *board.Reads,
) {
	for _, id := range req.Board.Artifacts() {
		artifact, err := s.artifact(ctx, req, id)
		if err != nil {
			if !errors.Is(err, errArtifactGone) {
				s.l.Warn("a board's artifact could not be read", zap.Error(err))
			}
			continue
		}
		switch {
		case artifact.Kind == assistantartifact.KindAnalysis:
			if read, ok := analysisOf(artifact); ok {
				reads.Analyses[id] = read
			}
		case tabular(artifact):
			if read, ok := tableOf(artifact); ok {
				reads.Tables[id] = read
			}
		}
	}
}

func tabular(artifact *assistantartifact.Artifact) bool {
	return artifact.Kind == assistantartifact.KindTableView &&
		assistantartifact.FamilyOf(artifact.Kind, artifact.Payload) == assistantartifact.FamilyTable
}

func analysisOf(artifact *assistantartifact.Artifact) (*board.AnalysisRead, bool) {
	var display analysis.Display
	if err := jsonutils.Convert(artifact.Payload, &display); err != nil {
		return nil, false
	}

	return &board.AnalysisRead{Title: artifact.Title, Display: display}, true
}

func tableOf(artifact *assistantartifact.Artifact) (*board.TableRead, bool) {
	var payload tableViewPayload
	if err := jsonutils.Convert(artifact.Payload, &payload); err != nil ||
		len(payload.Columns) == 0 {
		return nil, false
	}
	table := analysis.Table{
		Columns:   make([]analysis.TableColumn, 0, len(payload.Columns)),
		Rows:      make([][]any, 0, len(payload.Rows)),
		TotalRows: max(payload.RowCount, len(payload.Rows)),
	}
	for _, column := range payload.Columns {
		table.Columns = append(table.Columns, analysis.TableColumn{
			Name: column.Label,
			Kind: kindOf(column.Type),
		})
	}
	for _, row := range payload.Rows {
		cells := make([]any, 0, len(payload.Columns))
		for _, column := range payload.Columns {
			cells = append(cells, cellOf(column.Type, row[column.Key]))
		}
		table.Rows = append(table.Rows, cells)
	}

	return &board.TableRead{Title: artifact.Title, Table: table}, true
}

func kindOf(display assistantartifact.DisplayType) analysis.ValueKind {
	switch display {
	case assistantartifact.DisplayMoney:
		return analysis.ValueMoney
	case assistantartifact.DisplayNumber:
		return analysis.ValueNumber
	case assistantartifact.DisplayPercent:
		return analysis.ValuePercent
	case assistantartifact.DisplayDate:
		return analysis.ValueDate
	case assistantartifact.DisplayDateTime:
		return analysis.ValueDateTime
	case assistantartifact.DisplayText, assistantartifact.DisplayLongText,
		assistantartifact.DisplayEnum, assistantartifact.DisplayStatus,
		assistantartifact.DisplayBoolean, assistantartifact.DisplayFlag,
		assistantartifact.DisplayMetrics, assistantartifact.DisplayLinks:
	}

	return analysis.ValueText
}

func cellOf(display assistantartifact.DisplayType, value any) any {
	switch typed := value.(type) {
	case float64:
		if display == assistantartifact.DisplayPercent {
			return typed / percentPoints
		}
		return typed
	case string:
		return typed
	case bool:
		return typed
	case nil:
		return nil
	default:
		return typeutils.StringOfTrimmed(typed)
	}
}

func (s *Service) checkArtifacts(ctx context.Context, req *agentports.BoardFillRequest) error {
	for idx := range req.Board.Blocks {
		block := &req.Board.Blocks[idx]
		at := idx + 1
		if block.Artifact.IsNotNil() {
			artifact, err := s.artifact(ctx, req, block.Artifact)
			if err != nil || !tabular(artifact) {
				return mistake("block %d names a table that is not one this conversation kept; "+
					"name it by the artifactId its result gave you", at)
			}
		}
		if err := s.checkFigures(ctx, req, block, at); err != nil {
			return err
		}
		if block.Analysis.IsNil() {
			continue
		}
		display, err := s.analysisFor(ctx, req, block.Analysis, at)
		if err != nil {
			return err
		}
		if err = checkAnalysisBlock(block, display, at); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) checkFigures(
	ctx context.Context,
	req *agentports.BoardFillRequest,
	block *board.Block,
	at int,
) error {
	for figure := range block.Figures {
		source := &block.Figures[figure]
		if source.Analysis.IsNil() {
			continue
		}
		display, err := s.analysisFor(ctx, req, source.Analysis, at)
		if err != nil {
			return err
		}
		if source.Index >= len(display.Figures) {
			return mistake("block %d asks for figure %d of an analysis that showed %d",
				at, source.Index+1, len(display.Figures))
		}
	}

	return nil
}

func checkAnalysisBlock(block *board.Block, display *analysis.Display, at int) error {
	switch block.Type {
	case board.BlockChart:
		if block.Index >= len(display.Charts) {
			return mistake("block %d asks for chart %d of an analysis that drew %d",
				at, block.Index+1, len(display.Charts))
		}
	case board.BlockTable, board.BlockCompare:
		if block.Index >= len(display.Tables) {
			return mistake("block %d asks for table %d of an analysis that showed %d",
				at, block.Index+1, len(display.Tables))
		}
		if block.Type == board.BlockCompare {
			return checkCompare(block, &display.Tables[block.Index], at)
		}
	case board.BlockFigures, board.BlockRecords, board.BlockTimeline, board.BlockChecks,
		board.BlockMeter, board.BlockProse, board.BlockNote:
	}

	return nil
}

func (s *Service) analysisFor(
	ctx context.Context,
	req *agentports.BoardFillRequest,
	id pulid.ID,
	at int,
) (*analysis.Display, error) {
	artifact, err := s.artifact(ctx, req, id)
	if err != nil || artifact.Kind != assistantartifact.KindAnalysis {
		return nil, mistake("block %d names an analysis that is not one an analysis_run result "+
			"kept in this conversation; name it by its artifactId", at)
	}
	read, ok := analysisOf(artifact)
	if !ok {
		return nil, mistake("block %d names an analysis that could not be read", at)
	}

	return &read.Display, nil
}

func checkCompare(block *board.Block, table *analysis.Table, at int) error {
	names := make([]string, 0, len(table.Columns))
	for idx := range table.Columns {
		names = append(names, table.Columns[idx].Name)
	}
	has := func(name string) bool {
		for _, column := range names {
			if strings.EqualFold(strings.TrimSpace(column), strings.TrimSpace(name)) {
				return true
			}
		}
		return false
	}
	if !has(block.Key) {
		return mistake("block %d: the table has no %q column; its columns are %s",
			at, block.Key, strings.Join(names, ", "))
	}
	for _, row := range block.Rows {
		if !has(row.Column) {
			return mistake("block %d: the table has no %q column; its columns are %s",
				at, row.Column, strings.Join(names, ", "))
		}
	}

	return nil
}
