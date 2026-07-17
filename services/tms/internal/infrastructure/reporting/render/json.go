package render

import (
	"bufio"
	"context"
	"errors"
	"io"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/shopspring/decimal"
)

var _ services.ReportRenderer = (*JSONRenderer)(nil)

type JSONRenderer struct{}

func NewJSON() *JSONRenderer { return &JSONRenderer{} }

func (r *JSONRenderer) Format() report.Format { return report.FormatJSON }

type jsonSchemaColumn struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	Type   string `json:"type"`
	Format string `json:"format,omitempty"`
}

type jsonMeta struct {
	Title       string         `json:"title"`
	Description string         `json:"description,omitempty"`
	GeneratedAt int64          `json:"generatedAt"`
	Timezone    string         `json:"timezone,omitempty"`
	RequestedBy string         `json:"requestedBy,omitempty"`
	Params      map[string]any `json:"params,omitempty"`
}

func (r *JSONRenderer) writeEnvelopeHead(
	out *bufio.Writer,
	meta *services.ReportRunMeta,
	schema []services.ReportResultColumn,
) error {
	metaJSON, err := sonic.Marshal(jsonMeta{
		Title:       meta.Title,
		Description: meta.Description,
		GeneratedAt: meta.GeneratedAtUnix,
		Timezone:    meta.Timezone,
		RequestedBy: meta.RequestedBy,
		Params:      meta.Params,
	})
	if err != nil {
		return err
	}

	schemaColumns := make([]jsonSchemaColumn, len(schema))
	for i := range schema {
		schemaColumns[i] = jsonSchemaColumn{
			ID:     schema[i].ID,
			Label:  schema[i].Label,
			Type:   string(schema[i].Type),
			Format: string(schema[i].Format),
		}
	}
	schemaJSON, err := sonic.Marshal(schemaColumns)
	if err != nil {
		return err
	}

	if _, err = out.WriteString(`{"meta":`); err != nil {
		return err
	}
	if _, err = out.Write(metaJSON); err != nil {
		return err
	}
	if _, err = out.WriteString(`,"schema":`); err != nil {
		return err
	}
	if _, err = out.Write(schemaJSON); err != nil {
		return err
	}
	_, err = out.WriteString(`,"rows":[`)
	return err
}

func (r *JSONRenderer) Render(
	ctx context.Context,
	req *services.ReportRenderRequest,
) (*services.ReportRenderStats, error) {
	out := bufio.NewWriterSize(req.Sink, 64*1024)
	schema := req.Dataset.Schema()

	var err error
	if err = r.writeEnvelopeHead(out, &req.Meta, schema); err != nil {
		return nil, err
	}

	first := true
	encoded := make([]any, len(schema))
	for {
		row, nextErr := req.Dataset.Next(ctx)
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nil, nextErr
		}

		for i := range schema {
			encoded[i] = jsonValue(row[i])
		}
		rowJSON, rowErr := sonic.Marshal(encoded)
		if rowErr != nil {
			return nil, rowErr
		}

		if !first {
			if _, err = out.WriteString(","); err != nil {
				return nil, err
			}
		}
		first = false
		if _, err = out.Write(rowJSON); err != nil {
			return nil, err
		}
	}

	truncated := req.Dataset.Truncated()
	tail, err := sonic.Marshal(map[string]any{
		"rowCount":  req.Dataset.RowCount(),
		"truncated": truncated,
	})
	if err != nil {
		return nil, err
	}

	if _, err = out.WriteString(`],"summary":`); err != nil {
		return nil, err
	}
	if _, err = out.Write(tail); err != nil {
		return nil, err
	}
	if _, err = out.WriteString("}"); err != nil {
		return nil, err
	}
	if err = out.Flush(); err != nil {
		return nil, err
	}

	return &services.ReportRenderStats{
		Rows:      req.Dataset.RowCount(),
		Truncated: truncated,
	}, nil
}

func jsonValue(value any) any {
	if value == nil {
		return nil
	}
	switch v := value.(type) {
	case decimal.Decimal:
		return v.String()
	case time.Time:
		return v.Unix()
	default:
		return v
	}
}
