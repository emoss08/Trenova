package reporting

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/app/reporting/compiler"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/pkg/reportrows"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type RunCard struct {
	Run        *report.ReportRun
	Report     RunCardReport
	Parameters []RunCardParameter
	Timezone   string
	Sample     *reportrows.Sample
	Withheld   RunCardWithheld
	Previous   *RunCardPrevious
	Schedule   *report.ReportSchedule
	Chart      *report.ChartSpec
	Sort       []report.SortSpec
}

type RunCardWithheld string

const (
	RunCardWithheldAccess  = RunCardWithheld("access")
	RunCardWithheldChanged = RunCardWithheld("changed")
)

type RunCardReport struct {
	Name         string
	Description  string
	Category     string
	DefinitionID pulid.ID
	CannedKey    string
}

type RunCardParameter struct {
	Name        string
	Label       string
	Type        reportcatalog.FieldType
	Value       any
	IsDefault   bool
	RefEntity   string
	WindowStart int64
	WindowEnd   int64
}

type RunCardPrevious struct {
	RunID       pulid.ID
	CompletedAt int64
	RowCount    int64
	Schema      []reportrows.Column
	Totals      []any
}

func (s *Service) GetRunCard(ctx context.Context, req *GetRunRequest) (*RunCard, error) {
	run, err := s.GetRun(ctx, req)
	if err != nil {
		return nil, err
	}

	timezone := s.orgTimezone(ctx, req.TenantInfo)
	card := &RunCard{
		Run:        run,
		Timezone:   timezone,
		Parameters: []RunCardParameter{},
		Sort:       []report.SortSpec{},
	}

	definition, err := s.runCardReport(ctx, req, card)
	if err != nil {
		return nil, err
	}
	if definition != nil {
		card.Parameters = runCardParameters(definition, run, timezone)
		card.Sort = append(card.Sort, definition.Sort...)
		if len(definition.Charts) > 0 {
			chart := definition.Charts[0]
			card.Chart = &chart
		}
	}

	if !run.ScheduleID.IsNil() {
		card.Schedule = s.runCardSchedule(ctx, req, run)
	}

	if run.Sample == nil {
		return card, nil
	}
	if err = s.authorizeStoredRun(ctx, req, run); err != nil {
		withheld, ok := withheldReason(err)
		if !ok {
			return nil, err
		}
		card.Withheld = withheld
		return card, nil
	}
	card.Sample = run.Sample
	card.Previous = s.runCardPrevious(ctx, req, run)

	return card, nil
}

func (s *Service) runCardReport(
	ctx context.Context,
	req *GetRunRequest,
	card *RunCard,
) (definition *report.Definition, err error) {
	run := card.Run
	switch {
	case !run.DefinitionID.IsNil():
		card.Report.DefinitionID = run.DefinitionID
		return s.savedRunCardReport(ctx, req, card)
	case run.CannedKey != "":
		card.Report.CannedKey = run.CannedKey
		entry, ok := s.canned.Get(run.CannedKey)
		if ok {
			card.Report.Name = entry.Name
			card.Report.Description = entry.Description
			card.Report.Category = entry.Category
			definition = entry.Definition
		}
	}

	return definition, nil
}

func (s *Service) savedRunCardReport(
	ctx context.Context,
	req *GetRunRequest,
	card *RunCard,
) (definition *report.Definition, err error) {
	run := card.Run
	entity, err := s.defRepo.GetByID(ctx, &reportports.GetReportDefinitionRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: run.DefinitionID,
	})
	switch {
	case err == nil:
		card.Report.Name = entity.Name
		card.Report.Description = entity.Description
		card.Report.Category = entity.Category
	case !errortypes.IsNotFoundError(err):
		return nil, err
	}
	if run.RevisionID.IsNil() {
		return definition, nil
	}

	revision, err := s.defRepo.GetRevision(ctx, &reportports.GetReportRevisionRequest{
		TenantInfo: req.TenantInfo,
		RevisionID: run.RevisionID,
	})
	switch {
	case err == nil:
		definition = revision.Definition
	case !errortypes.IsNotFoundError(err):
		return nil, err
	}

	return definition, nil
}

func (s *Service) runCardSchedule(
	ctx context.Context,
	req *GetRunRequest,
	run *report.ReportRun,
) *report.ReportSchedule {
	schedule, err := s.scheduleRepo.GetByID(ctx, &reportports.GetReportScheduleRequest{
		TenantInfo: req.TenantInfo,
		ScheduleID: run.ScheduleID,
	})
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			s.l.Warn("failed to read the schedule a report run came from",
				zap.String("runId", run.ID.String()), zap.Error(err))
		}
		return nil
	}

	return schedule
}

func (s *Service) runCardPrevious(
	ctx context.Context,
	req *GetRunRequest,
	run *report.ReportRun,
) *RunCardPrevious {
	if run.Status != report.RunStatusSucceeded || run.CompletedAt == 0 {
		return nil
	}

	previous, err := s.runRepo.LatestComparable(ctx, &reportports.GetComparableReportRunRequest{
		TenantInfo:      req.TenantInfo,
		RequestedBy:     run.RequestedByID,
		DefinitionID:    run.DefinitionID,
		CannedKey:       run.CannedKey,
		Params:          run.Params,
		ExcludeRunID:    run.ID,
		CompletedBefore: run.CompletedAt,
	})
	if err != nil {
		if !errortypes.IsNotFoundError(err) {
			s.l.Warn("failed to read the run a report run is compared with",
				zap.String("runId", run.ID.String()), zap.Error(err))
		}
		return nil
	}
	if previous.Sample == nil || len(previous.Sample.Totals) == 0 {
		return nil
	}

	return &RunCardPrevious{
		RunID:       previous.ID,
		CompletedAt: previous.CompletedAt,
		RowCount:    previous.RowCount,
		Schema:      previous.Sample.Schema,
		Totals:      previous.Sample.Totals,
	}
}

func withheldReason(err error) (RunCardWithheld, bool) {
	if _, ok := errors.AsType[*errortypes.AuthorizationError](err); ok {
		return RunCardWithheldAccess, true
	}
	if _, ok := errors.AsType[*errortypes.BusinessError](err); ok {
		return RunCardWithheldChanged, true
	}
	if errortypes.IsNotFoundError(err) {
		return RunCardWithheldChanged, true
	}

	return "", false
}

func runCardParameters(
	definition *report.Definition,
	run *report.ReportRun,
	timezone string,
) []RunCardParameter {
	if len(definition.Parameters) == 0 {
		return []RunCardParameter{}
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}
	ranAt := run.StartedAt
	if ranAt == 0 {
		ranAt = run.QueuedAt
	}
	now := time.Unix(ranAt, 0)

	parameters := make([]RunCardParameter, 0, len(definition.Parameters))
	for i := range definition.Parameters {
		def := &definition.Parameters[i]
		value, given := run.Params[def.Name]
		isDefault := !given || value == nil
		if isDefault {
			value = def.Default
		}
		parameter := RunCardParameter{
			Name:      def.Name,
			Label:     def.Label,
			Type:      def.Type,
			Value:     value,
			IsDefault: isDefault,
			RefEntity: def.RefEntity,
		}
		if op, ok := parameterOperator(definition, def.Name); ok {
			start, end, windowed := compiler.ParameterWindow(op, value, loc, now)
			if windowed {
				parameter.WindowStart = start
				parameter.WindowEnd = end
			}
		}
		parameters = append(parameters, parameter)
	}

	return parameters
}

func parameterOperator(
	definition *report.Definition,
	name string,
) (op dbtype.Operator, ok bool) {
	find := func(filter *report.FieldFilter) error {
		if !ok && filter.Param == name {
			op, ok = filter.Operator, true
		}
		return nil
	}
	_ = definition.Filters.Walk(find)
	_ = definition.Having.Walk(find)

	return op, ok
}
