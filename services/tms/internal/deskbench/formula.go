package deskbench

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/pagedraft"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultFormulaSchema = "shipment"
	defaultFormulaType   = "FreightCharge"
	formulaPagePath      = "/formula-templates"
	formulaPageTitle     = "Formula studio"
)

type FormulaPage struct {
	Template string `yaml:"template" json:"template,omitempty"`
	Type     string `yaml:"type"     json:"type,omitempty"`
}

type formulaState struct {
	draft pagedraft.Formula
}

var errTemplateNotFound = errors.New("no formula template has that name")

func (s *Session) OpenFormula(ctx context.Context, label string, page *FormulaPage) (*Conversation, error) {
	ctx = s.Context(ctx)

	state := &formulaState{draft: pagedraft.Formula{
		SchemaID:     defaultFormulaSchema,
		TemplateType: defaultFormulaType,
		Variables:    []pagedraft.FormulaVariable{},
	}}
	if page.Type != "" {
		state.draft.TemplateType = page.Type
	}

	req := &serviceports.OpenPageThreadRequest{
		TenantInfo:  s.Tenant,
		Origin:      conversation.ThreadOriginFormula,
		SubjectType: agent.SubjectFormulaTemplate,
	}
	if strings.TrimSpace(page.Template) != "" {
		templateID, err := s.loadFormula(ctx, page.Template, &state.draft)
		if err != nil {
			return nil, fmt.Errorf("open formula template %q: %w", page.Template, err)
		}
		req.SubjectID = templateID
	}

	opened, err := s.bench.Pages.OpenPageThread(ctx, req, &s.Actor)
	if err != nil {
		return nil, fmt.Errorf("open the formula studio's assistant: %w", err)
	}

	return &Conversation{
		Label:   label,
		session: s,
		Thread:  opened.Thread,
		queue:   s.bench.watcher.watch(opened.Thread.ID),
		formula: state,
	}, nil
}

func (s *Session) loadFormula(ctx context.Context, name string, draft *pagedraft.Formula) (pulid.ID, error) {
	type row struct {
		id         string
		kind       string
		schema     string
		expression string
		variables  []byte
	}

	found, err := dbtx.Read(ctx, s.bench.DB, func(ctx context.Context) (*row, error) {
		r := &row{}
		scanErr := s.bench.DB.DBForContext(ctx).QueryRowContext(ctx,
			`SELECT id, type, schema_id, expression, COALESCE(variable_definitions, '[]'::jsonb)
			 FROM formula_templates WHERE name ILIKE ? ORDER BY updated_at DESC LIMIT 1`,
			name,
		).Scan(&r.id, &r.kind, &r.schema, &r.expression, &r.variables)
		if errors.Is(scanErr, sql.ErrNoRows) {
			return nil, errTemplateNotFound
		}

		return r, scanErr
	})
	if err != nil {
		return pulid.Nil, err
	}

	templateID, err := pulid.MustParse(found.id)
	if err != nil {
		return pulid.Nil, err
	}

	variables := []pagedraft.FormulaVariable{}
	if len(found.variables) > 0 {
		if err = sonic.Unmarshal(found.variables, &variables); err != nil {
			return pulid.Nil, fmt.Errorf("read its variables: %w", err)
		}
	}

	draft.TemplateID = templateID.String()
	draft.SchemaID = found.schema
	draft.TemplateType = found.kind
	draft.Expression = found.expression
	draft.Variables = variables

	return templateID, nil
}

func (f *formulaState) page() *agent.PageContext {
	draft := f.draft
	draft.Variables = append([]pagedraft.FormulaVariable(nil), f.draft.Variables...)

	return &agent.PageContext{
		Path:  formulaPagePath,
		Title: formulaPageTitle,
		Draft: &pagedraft.Draft{Surface: pagedraft.SurfaceFormula, Formula: &draft},
	}
}

func (f *formulaState) apply(artifacts []serviceports.AssistantArtifact) {
	for idx := range artifacts {
		artifact := &artifacts[idx]
		if artifact.Kind != assistantartifact.KindDraftEdit {
			continue
		}

		var edit pagedraft.Edit
		if err := jsonutils.Convert(artifact.Payload, &edit); err != nil || edit.Formula == nil {
			continue
		}
		if edit.Formula.SchemaID != "" {
			f.draft.SchemaID = edit.Formula.SchemaID
		}
		f.draft.Expression = edit.Formula.Expression
		f.draft.Variables = append([]pagedraft.FormulaVariable(nil), edit.Formula.Variables...)
	}
}

func (f *formulaState) expression() string {
	if f == nil {
		return ""
	}

	return f.draft.Expression
}

func (f *formulaState) describe() string {
	if f == nil {
		return ""
	}

	names := make([]string, 0, len(f.draft.Variables))
	for _, variable := range f.draft.Variables {
		names = append(names, variable.Name)
	}
	if len(names) == 0 {
		return f.draft.Expression
	}

	return f.draft.Expression + "  [variables: " + strings.Join(names, ", ") + "]"
}
