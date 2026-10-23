package tariff

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
)

const cellPasses = 4

var ErrUnresolvedNeeds = errors.New(
	"tariff: the module still needs table cells after loading whole tables",
)

type cellLoad struct {
	contexts  tariffports.ContextSource
	tenant    pagination.TenantInfo
	tariffCtx *rules.TariffContext
}

func (l *cellLoad) until(ctx context.Context, call func() ([]rules.TariffNeed, error)) error {
	for pass := 0; ; pass++ {
		needs, err := call()
		if err != nil {
			return err
		}
		if len(needs) == 0 {
			return nil
		}
		if err = l.load(ctx, pass, needs); err != nil {
			return err
		}
	}
}

func (l *cellLoad) load(ctx context.Context, pass int, needs []rules.TariffNeed) error {
	switch {
	case pass < cellPasses-1:
		return l.contexts.LoadCells(ctx, l.tenant, l.tariffCtx, needs)
	case pass == cellPasses-1:
		return l.contexts.LoadTables(ctx, l.tenant, l.tariffCtx, neededTables(needs))
	default:
		return ErrUnresolvedNeeds
	}
}

func neededTables(needs []rules.TariffNeed) []string {
	seen := make(map[string]struct{}, len(needs))
	tables := make([]string, 0, len(needs))
	for _, need := range needs {
		if _, ok := seen[need.Table]; ok {
			continue
		}
		seen[need.Table] = struct{}{}
		tables = append(tables, need.Table)
	}
	return tables
}

type TableCellsRequest struct {
	TenantInfo pagination.TenantInfo
	Needs      []rules.TariffNeed
	Whole      bool
}

type TableCellsTable struct {
	Code   string             `json:"code"`
	Cells  []rules.TariffCell `json:"cells"`
	Loaded [][]string         `json:"loaded"`
}

type TableCells struct {
	Stamp  string            `json:"stamp"`
	Tables []TableCellsTable `json:"tables"`
}

func (c *Checker) TableCells(ctx context.Context, req *TableCellsRequest) (*TableCells, error) {
	stamp, err := c.context.TableStamp(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	tariffCtx, err := c.contextFor(ctx, req.TenantInfo, nil)
	if err != nil {
		return nil, err
	}
	codes := neededTables(req.Needs)
	if req.Whole {
		err = c.context.LoadTables(ctx, req.TenantInfo, tariffCtx, codes)
	} else {
		err = c.context.LoadCells(ctx, req.TenantInfo, tariffCtx, req.Needs)
	}
	if err != nil {
		return nil, err
	}

	out := &TableCells{Stamp: stamp, Tables: make([]TableCellsTable, 0, len(codes))}
	for _, code := range codes {
		at := tableAt(tariffCtx, code)
		if at < 0 {
			continue
		}
		table := &tariffCtx.Tables[at]
		cells := table.Cells
		if cells == nil {
			cells = []rules.TariffCell{}
		}
		out.Tables = append(out.Tables, TableCellsTable{
			Code:   table.Code,
			Cells:  cells,
			Loaded: table.Loaded,
		})
	}
	return out, nil
}
