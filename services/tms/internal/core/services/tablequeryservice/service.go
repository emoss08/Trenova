// Package tablequeryservice turns a sentence into a table's filters.
//
// "Shipments in transit for Acme that haven't shipped yet" is a question every
// dispatcher can ask and almost nobody can express in a filter builder, because
// expressing it means knowing that the column is billingTransferStatus, that
// the value is spelled InTransit, and that "haven't shipped yet" is a null
// check on actualShipDate rather than a date in the future.
//
// The model's job here is only translation: it reads the prompt and names
// fields, operators and values from a catalogue it is shown. It never writes
// SQL and it never invents a column, because what it produces is compiled by
// filtercatalog — the same compiler the list tools go through — and anything
// the catalogue refuses is reported back as unresolved rather than run. A
// wrong guess costs the asker a line of explanation, not a wrong answer.
package tablequeryservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// maxPromptRunes bounds what reaches the model. A filter is a sentence; a
	// page pasted into the box is not a filter, and paying to classify one is
	// the kind of cost nobody notices until the invoice.
	maxPromptRunes  = 500
	maxOutputTokens = 900
	// maxComposedFilters matches what the list tools allow, so a composed view
	// and a tool call cannot disagree about what is too much to ask at once.
	maxComposedFilters = 6
)

type ServiceParams struct {
	fx.In

	Logger        *zap.Logger
	Completion    serviceports.StructuredCompleter
	Permissions   serviceports.PermissionEngine
	Catalog       *filtercatalog.Catalog
	Organizations repositories.OrganizationRepository
}

type Service struct {
	logger        *zap.Logger
	completion    serviceports.StructuredCompleter
	permissions   serviceports.PermissionEngine
	catalog       *filtercatalog.Catalog
	organizations repositories.OrganizationRepository
}

func NewService(p ServiceParams) *Service {
	return &Service{
		logger:        p.Logger.Named("tablequery-service"),
		completion:    p.Completion,
		permissions:   p.Permissions,
		catalog:       p.Catalog,
		organizations: p.Organizations,
	}
}

// CurrentView is the state the table is already in, so a follow-up narrows
// rather than replaces. "And only Acme's" means nothing without it.
type CurrentView struct {
	Query        string                    `json:"query"`
	FieldFilters []domaintypes.FieldFilter `json:"fieldFilters"`
	Sort         []domaintypes.SortField   `json:"sort"`
}

type ComposeRequest struct {
	TenantInfo pagination.TenantInfo
	Actor      *serviceports.RequestActor
	Resource   permission.Resource
	Prompt     string
	Current    CurrentView
	// Timezone overrides the organization's zone. It is for tests and for a
	// caller that already knows it; ordinarily it is left empty and looked up,
	// because "the next 7 days" measured in the wrong zone is a window nobody
	// asked for.
	Timezone string
}

// Unresolved is something the prompt asked for that the catalogue has no
// answer to.
//
// It is reported rather than dropped because a filter that silently did not
// apply is the worst outcome available: the table comes back looking answered,
// and the one condition the person cared about is the one that went missing.
type Unresolved struct {
	Phrase string `json:"phrase"`
	Reason string `json:"reason"`
}

type ComposeResult struct {
	Query        string                    `json:"query"`
	FieldFilters []domaintypes.FieldFilter `json:"fieldFilters"`
	Sort         []domaintypes.SortField   `json:"sort"`
	// Explanation is the compiled view in words, which is what the chips'
	// popover shows. It is built from what compiled, never from the prompt, so
	// it cannot describe a filter that is not there.
	Explanation string       `json:"explanation"`
	Terms       []string     `json:"terms"`
	Unresolved  []Unresolved `json:"unresolved"`
}

// Compose reads the prompt against one resource's catalogue.
func (s *Service) Compose(ctx context.Context, req *ComposeRequest) (*ComposeResult, error) {
	resource, err := s.authorize(ctx, req)
	if err != nil {
		return nil, err
	}

	prompt := strings.TrimSpace(req.Prompt)
	if prompt == "" {
		return nil, errortypes.NewValidationError(
			"prompt", errortypes.ErrRequired, "Say what to narrow the table to.",
		)
	}
	if len([]rune(prompt)) > maxPromptRunes {
		return nil, errortypes.NewValidationError(
			"prompt", errortypes.ErrInvalid,
			fmt.Sprintf("Keep it under %d characters.", maxPromptRunes),
		)
	}

	completion, err := s.completion.CompleteStructured(
		ctx,
		&serviceports.StructuredCompletionRequest{
			TenantInfo:   req.TenantInfo,
			Task:         aiprovider.TaskQueryCompose,
			System:       systemPrompt,
			Context:      buildContext(resource, prompt, req.Current),
			OutputSchema: outputSchema(resource),
			SchemaName:   "table_query",
			MaxTokens:    maxOutputTokens,
			Attribution:  serviceports.AIUsageAttribution{UserID: req.Actor.UserID},
		},
	)
	if err != nil {
		return nil, err
	}

	draft, err := parseDraft(completion.Text)
	if err != nil {
		s.logger.Warn("a composed table query could not be read",
			zap.String("model", completion.ModelIdentifier),
			zap.String("resource", string(req.Resource)),
			zap.Error(err),
		)

		return nil, errortypes.NewBusinessError(
			"The question could not be turned into filters. Try wording it differently.",
		)
	}

	return s.compile(resource, draft, s.timezone(ctx, req)), nil
}

// timezone is the organization's, because a relative window is whole days in
// the zone whose days are meant. A lookup that fails leaves it empty, which
// the clock reads as UTC — wrong for some organizations, but never wrong in a
// way that silently shifts an explicit calendar date.
func (s *Service) timezone(ctx context.Context, req *ComposeRequest) string {
	if req.Timezone != "" || s.organizations == nil {
		return req.Timezone
	}

	org, err := s.organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		s.logger.Warn("a composed table query could not read the organization's zone",
			zap.String("organization", req.TenantInfo.OrgID.String()),
			zap.Error(err),
		)

		return ""
	}

	return org.Timezone
}

// authorize refuses before the model call, not after it.
//
// A composed filter is a read of the resource by another name, so somebody who
// cannot open the table cannot pay to have one described either — and the
// refusal has to come first, because a model call that ends in 403 has already
// cost the organization money and already been shown the catalogue.
func (s *Service) authorize(
	ctx context.Context,
	req *ComposeRequest,
) (filtercatalog.Resource, error) {
	if req.Actor == nil {
		return filtercatalog.Resource{}, errortypes.NewAuthorizationError(
			"This request has no actor.",
		)
	}

	resource, ok := s.catalog.For(req.Resource)
	if !ok {
		return filtercatalog.Resource{}, errortypes.NewValidationError(
			"resource", errortypes.ErrInvalid,
			fmt.Sprintf("%s cannot be narrowed by description yet.", req.Resource),
		)
	}

	result, err := s.permissions.Check(ctx, &serviceports.PermissionCheckRequest{
		PrincipalType:  req.Actor.PrincipalType,
		PrincipalID:    req.Actor.PrincipalID,
		UserID:         req.Actor.UserID,
		APIKeyID:       req.Actor.APIKeyID,
		BusinessUnitID: req.Actor.BusinessUnitID,
		OrganizationID: req.Actor.OrganizationID,
		Resource:       resource.Resource.String(),
		Operation:      permission.OpRead,
	})
	if err != nil {
		return filtercatalog.Resource{}, err
	}
	if !result.Allowed {
		return filtercatalog.Resource{}, errortypes.NewAuthorizationError(
			fmt.Sprintf("You do not have permission to read %s.", resource.Entity),
		)
	}

	return resource, nil
}

// compile turns the model's draft into filters, one condition at a time.
//
// Each is compiled on its own so that one bad guess costs its own condition
// rather than the whole request. A person who asked for three things and got
// two, with a line saying why the third did not apply, has been served; one
// who got an error has not.
func (s *Service) compile(
	resource filtercatalog.Resource,
	draft *draft,
	timezone string,
) *ComposeResult {
	criteria := filtercatalog.NewCriteria(resource.Entity).
		At(filtercatalog.NewClock(timezone))

	result := &ComposeResult{
		Query:        draft.Query,
		FieldFilters: make([]domaintypes.FieldFilter, 0, len(draft.Filters)),
		Unresolved:   make([]Unresolved, 0, len(draft.Unresolved)),
	}
	criteria.Text(draft.Query)

	for index, condition := range draft.Filters {
		if index >= maxComposedFilters {
			result.Unresolved = append(result.Unresolved, Unresolved{
				Phrase: condition.Field,
				Reason: fmt.Sprintf("Only %d filters can be applied at once.", maxComposedFilters),
			})

			continue
		}

		filters, err := resource.Compile(
			[]filtercatalog.Condition{condition.toCondition()},
			criteria,
		)
		if err != nil {
			result.Unresolved = append(result.Unresolved, Unresolved{
				Phrase: condition.describe(),
				Reason: err.Error(),
			})

			continue
		}
		result.FieldFilters = append(result.FieldFilters, filters...)
	}

	result.Sort = s.sort(resource, draft, result)
	result.Terms = criteria.Terms()
	result.Explanation = explain(resource, criteria)

	for _, unresolved := range draft.Unresolved {
		if strings.TrimSpace(unresolved.Phrase) == "" {
			continue
		}
		result.Unresolved = append(result.Unresolved, unresolved)
	}

	return result
}

func (s *Service) sort(
	resource filtercatalog.Resource,
	draft *draft,
	result *ComposeResult,
) []domaintypes.SortField {
	sort, err := resource.Sort(draft.SortBy, draft.SortDirection)
	if err != nil {
		result.Unresolved = append(result.Unresolved, Unresolved{
			Phrase: "sorted by " + draft.SortBy,
			Reason: err.Error(),
		})

		return nil
	}

	if len(sort) > 0 {
		direction := "newest first"
		if sort[0].Direction == dbtype.SortDirectionAsc {
			direction = "oldest first"
		}
		result.Terms = append(result.Terms, "sorted by "+sort[0].Field+", "+direction)
	}

	return sort
}

// explain reads the compiled view back as one sentence.
func explain(resource filtercatalog.Resource, criteria *filtercatalog.Criteria) string {
	terms := criteria.Terms()
	if len(terms) == 0 {
		return "Every " + resource.Entity + ", unfiltered."
	}

	return resource.Entity + " where " + strings.Join(terms, ", and ")
}

// draft is what the model returns, before anything has been checked.
type draft struct {
	Query         string        `json:"query"`
	Filters       []draftFilter `json:"filters"`
	SortBy        string        `json:"sortBy"`
	SortDirection string        `json:"sortDirection"`
	Unresolved    []Unresolved  `json:"unresolved"`
}

type draftFilter struct {
	Field    string   `json:"field"`
	Operator string   `json:"operator"`
	Value    string   `json:"value"`
	Values   []string `json:"values"`
	Days     int      `json:"days"`
}

func (f draftFilter) toCondition() filtercatalog.Condition {
	return filtercatalog.Condition{
		Field:    f.Field,
		Operator: dbtype.Operator(f.Operator),
		Value:    f.Value,
		Values:   f.Values,
		Days:     f.Days,
	}
}

// describe names the condition in the words the model used, so an unresolved
// entry says what was asked for rather than which array index failed.
func (f draftFilter) describe() string {
	parts := []string{f.Field, f.Operator}
	switch {
	case f.Value != "":
		parts = append(parts, f.Value)
	case len(f.Values) > 0:
		parts = append(parts, strings.Join(f.Values, ", "))
	case f.Days > 0:
		parts = append(parts, fmt.Sprintf("%d days", f.Days))
	}

	return strings.TrimSpace(strings.Join(parts, " "))
}

func parseDraft(text string) (*draft, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, fmt.Errorf("the model returned nothing")
	}

	var parsed draft
	if err := sonic.UnmarshalString(trimmed, &parsed); err != nil {
		return nil, fmt.Errorf("the reply was not the requested object: %w", err)
	}

	return &parsed, nil
}
