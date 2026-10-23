package deskbench

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	formulaPagePath  = "/billing/configuration-files/formula-templates"
	libraryPagePath  = "/billing/configuration-files/formula-libraries"
	formulaPageTitle = "Formula studio"
	libraryPageTitle = "Formula library"
)

type FormulaPage struct {
	Template string `yaml:"template" json:"template,omitempty"`
	Library  string `yaml:"library"  json:"library,omitempty"`
}

func (p *FormulaPage) editsLibrary() bool {
	return strings.TrimSpace(p.Library) != ""
}

type formulaState struct {
	library bool
	id      string
	name    string
	about   string
	source  string
}

var (
	errTemplateNotFound = errors.New("no formula template has that name")
	errLibraryNotFound  = errors.New("no formula library has that name")
)

func (s *Session) OpenFormula(
	ctx context.Context,
	label string,
	page *FormulaPage,
) (*Conversation, error) {
	ctx = s.Context(ctx)

	state := &formulaState{library: page.editsLibrary()}
	req := &agentports.OpenPageThreadRequest{
		TenantInfo:  s.Tenant,
		Origin:      conversation.ThreadOriginFormula,
		SubjectType: agent.SubjectFormulaTemplate,
	}
	switch {
	case state.library:
		req.SubjectType = agent.SubjectFormulaLibrary
		id, err := s.loadSource(ctx, librarySource, page.Library, state)
		if err != nil {
			return nil, fmt.Errorf("open formula library %q: %w", page.Library, err)
		}
		req.SubjectID = id
	case strings.TrimSpace(page.Template) != "":
		id, err := s.loadSource(ctx, templateSource, page.Template, state)
		if err != nil {
			return nil, fmt.Errorf("open formula template %q: %w", page.Template, err)
		}
		req.SubjectID = id
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

type sourceQuery struct {
	query    string
	notFound error
}

var (
	templateSource = sourceQuery{
		query: `SELECT id, name, COALESCE(description, ''), source
			FROM formula_templates WHERE name ILIKE ? ORDER BY updated_at DESC LIMIT 1`,
		notFound: errTemplateNotFound,
	}
	librarySource = sourceQuery{
		query: `SELECT id, name, COALESCE(description, ''), source
			FROM formula_libraries WHERE lower(name) = lower(?) LIMIT 1`,
		notFound: errLibraryNotFound,
	}
)

func (s *Session) loadSource(
	ctx context.Context,
	from sourceQuery,
	name string,
	state *formulaState,
) (pulid.ID, error) {
	name = strings.TrimPrefix(strings.TrimSpace(name), "org.")
	found, err := dbtx.Read(ctx, s.bench.DB, func(ctx context.Context) (*formulaState, error) {
		row := &formulaState{library: state.library}
		scanErr := s.bench.DB.DBForContext(ctx).QueryRowContext(ctx, from.query, name).
			Scan(&row.id, &row.name, &row.about, &row.source)
		if errors.Is(scanErr, sql.ErrNoRows) {
			return nil, from.notFound
		}

		return row, scanErr
	})
	if err != nil {
		return pulid.Nil, err
	}

	id, err := pulid.MustParse(found.id)
	if err != nil {
		return pulid.Nil, err
	}
	*state = *found

	return id, nil
}

func (f *formulaState) page() *agent.PageContext {
	if f.library {
		return &agent.PageContext{
			Path:  libraryPagePath,
			Title: libraryPageTitle,
			Draft: &pagedraft.Draft{
				Surface: pagedraft.SurfaceFormulaLibrary,
				FormulaLibrary: &pagedraft.FormulaLibrary{
					LibraryID:   f.id,
					Name:        f.name,
					Description: f.about,
					Source:      f.source,
				},
			},
		}
	}

	return &agent.PageContext{
		Path:  formulaPagePath,
		Title: formulaPageTitle,
		Draft: &pagedraft.Draft{
			Surface: pagedraft.SurfaceFormula,
			Formula: &pagedraft.Formula{
				TemplateID:  f.id,
				Name:        f.name,
				Description: f.about,
				Source:      f.source,
			},
		},
	}
}

func (f *formulaState) apply(artifacts []agentports.AssistantArtifact) {
	for idx := range artifacts {
		artifact := &artifacts[idx]
		if artifact.Kind != assistantartifact.KindDraftEdit {
			continue
		}

		var edit pagedraft.Edit
		if err := jsonutils.Convert(artifact.Payload, &edit); err != nil {
			continue
		}
		proposal := edit.Formula
		if f.library {
			proposal = edit.Library
		}
		if proposal == nil {
			continue
		}
		f.source = proposal.Source
		if proposal.Name != "" {
			f.name = proposal.Name
		}
		if proposal.Description != "" {
			f.about = proposal.Description
		}
	}
}

func (f *formulaState) expression() string {
	if f == nil {
		return ""
	}

	return f.source
}

func (f *formulaState) describe() string {
	if f == nil {
		return ""
	}

	return f.source
}
