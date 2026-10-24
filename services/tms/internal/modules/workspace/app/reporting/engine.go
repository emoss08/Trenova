package reporting

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/keel/analytics"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/reportaccess"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

type Prepared struct {
	Context  context.Context
	Request  analytics.Request
	Access   *reportaccess.Access
	Location *time.Location
}

type PrepareParams struct {
	Tenant     pagination.TenantInfo
	Definition *ir.Definition
	Params     map[string]any
	Mode       reportaccess.Mode
	Key        string
	Timezone   string
}

func (s *Service) Prepare(ctx context.Context, p *PrepareParams) (*Prepared, error) {
	if p.Definition == nil {
		return nil, errortypes.NewValidationError(
			"definition", errortypes.ErrRequired, "Definition is required",
		)
	}
	access, err := s.access.For(ctx, p.Tenant, p.Mode)
	if err != nil {
		return nil, err
	}

	timezone := p.Timezone
	if timezone == "" {
		timezone = s.orgTimezone(ctx, p.Tenant)
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	key := ""
	if p.Key != "" {
		key = p.Key + ":" + p.Tenant.UserID.String()
	}

	return &Prepared{
		Context: dbscope.WithTenant(ctx, p.Tenant.DBTenant()),
		Request: analytics.Request{
			Definition: p.Definition,
			Params:     p.Params,
			Policy:     access,
			Location:   loc,
			Now:        timeutils.Now(),
			Key:        key,
		},
		Access:   access,
		Location: loc,
	}, nil
}

func (s *Service) Check(ctx context.Context, p *PrepareParams) error {
	prepared, err := s.Prepare(ctx, p)
	if err != nil {
		return err
	}
	if _, err = s.engine.Compile(prepared.Context, prepared.Request); err != nil {
		return EngineError(prepared.Context, prepared.Access, err)
	}
	return nil
}

func EngineError(ctx context.Context, access *reportaccess.Access, err error) error {
	var ae *analytics.Error
	if !errors.As(err, &ae) {
		return err
	}

	switch ae.Code {
	case analytics.CodeDenied:
		if access != nil {
			return access.Refusal(ctx, err)
		}
		return errortypes.NewAuthorizationError(
			"This report references data you do not have access to: {0}", ae.Message,
		)
	case analytics.CodeInvalid:
		return definitionErrors(ae.Details, ae.Message)
	case analytics.CodeParameter:
		return errortypes.NewValidationError(
			"params",
			errortypes.ErrInvalid,
			"This report's parameters are not valid: {0}",
			ae.Message,
		)
	case analytics.CodeCursor:
		return errortypes.NewValidationError(
			"cursor", errortypes.ErrInvalid, "This page cursor does not belong to this report",
		)
	case analytics.CodeNotAllowed, analytics.CodeTooComplex:
		return errortypes.NewBusinessError("This report cannot run: {0}", ae.Message)
	case analytics.CodeTooCostly:
		return errortypes.NewBusinessError(
			"This report would read too much data to run; narrow its filters or date range",
		)
	case analytics.CodeBusy:
		return errortypes.NewRateLimitError(
			"report",
			"Your organization is already running as many reports as it can at once; try again shortly",
		)
	case analytics.CodeSuperseded:
		return context.Canceled
	case analytics.CodeTimeout:
		return errortypes.NewBusinessError(
			"This report took too long to run; narrow its filters or run it in the background",
		)
	case analytics.CodeCatalog:
		return fmt.Errorf("report catalog: %w", err)
	default:
		return err
	}
}

func DecodeDefinition(value any) (*ir.Definition, error) {
	def, err := report.DecodeDefinition(value)
	if err == nil {
		err = ir.Validate(def, ir.Limits{})
	}
	if err == nil {
		return def, nil
	}
	var details ir.Errors
	if errors.As(err, &details) {
		return nil, definitionErrors(details, err.Error())
	}
	return nil, definitionErrors(nil, err.Error())
}

func definitionErrors(details ir.Errors, message string) error {
	multiErr := errortypes.NewMultiError()
	if len(details) == 0 {
		multiErr.Add(
			"definition",
			errortypes.ErrInvalid,
			"This report definition is not valid: {0}",
			message,
		)
		return multiErr
	}
	for _, detail := range details {
		if len(detail.Suggestions) > 0 {
			multiErr.Add(
				pointerField(detail.Pointer),
				errortypes.ErrInvalid,
				"This report definition is not valid: {0}. Did you mean {1}?",
				detail.Message,
				strings.Join(detail.Suggestions, ", "),
			)
			continue
		}
		multiErr.Add(
			pointerField(detail.Pointer),
			errortypes.ErrInvalid,
			"This report definition is not valid: {0}",
			detail.Message,
		)
	}
	return multiErr
}

func pointerField(pointer string) string {
	if pointer == "" || pointer == "/" {
		return "definition"
	}
	parts := strings.Split(strings.TrimPrefix(pointer, "/"), "/")
	var b strings.Builder
	b.WriteString("definition")
	for _, part := range parts {
		if part != "" && strings.Trim(part, "0123456789") == "" {
			b.WriteString("[" + part + "]")
			continue
		}
		b.WriteString("." + part)
	}
	return b.String()
}

func sortedNames(values map[string]any) []string {
	return slices.Sorted(maps.Keys(values))
}
