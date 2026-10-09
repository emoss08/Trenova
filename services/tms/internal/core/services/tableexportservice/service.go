package tableexportservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/services/reporting"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/reportcatalog"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const Category = "Table exports"

type Params struct {
	fx.In

	Logger    *zap.Logger
	Reporting *reporting.Service
}

type Service struct {
	l         *zap.Logger
	reporting *reporting.Service
}

func New(p Params) *Service {
	return &Service{l: p.Logger.Named("service.tableexport"), reporting: p.Reporting}
}

type ExportRequest struct {
	reporting.Request

	Name   string
	View   View
	Format report.Format
}

type ScheduleRequest struct {
	reporting.Request

	Name            string
	View            View
	CronExpression  string
	Timezone        string
	Formats         []string
	EmailRecipients []string
	EmailAttach     bool
	EmailInline     bool
}

type Result struct {
	DefinitionID pulid.ID
	Run          *report.ReportRun
	Schedule     *report.ReportSchedule
	Skipped      []string
}

func (s *Service) Export(ctx context.Context, req *ExportRequest) (*Result, error) {
	if !req.Format.IsValid() {
		return nil, errortypes.NewValidationError(
			"format",
			errortypes.ErrInvalid,
			"Choose a file format",
		)
	}

	definition, built, err := s.saveDefinition(ctx, &req.Request, req.Name, &req.View, req.Format)
	if err != nil {
		return nil, err
	}

	run, err := s.reporting.RunReport(ctx, &reporting.RunReportRequest{
		Request:      req.Request,
		DefinitionID: definition.ID,
		Format:       req.Format,
		Trigger:      report.RunTriggerManual,
	})
	if err != nil {
		return nil, err
	}

	return &Result{DefinitionID: definition.ID, Run: run, Skipped: built.Skipped}, nil
}

func (s *Service) Schedule(ctx context.Context, req *ScheduleRequest) (*Result, error) {
	if len(req.Formats) == 0 {
		return nil, errortypes.NewValidationError(
			"formats",
			errortypes.ErrRequired,
			"Choose at least one file format",
		)
	}

	definition, built, err := s.saveDefinition(
		ctx,
		&req.Request,
		req.Name,
		&req.View,
		report.Format(req.Formats[0]),
	)
	if err != nil {
		return nil, err
	}

	schedule, err := s.reporting.CreateSchedule(ctx, &reporting.SaveScheduleRequest{
		Request:         req.Request,
		DefinitionID:    definition.ID,
		CronExpression:  req.CronExpression,
		Timezone:        req.Timezone,
		Formats:         req.Formats,
		EmailRecipients: req.EmailRecipients,
		EmailAttach:     req.EmailAttach,
		EmailInline:     req.EmailInline,
		Enabled:         true,
	})
	if err != nil {
		return nil, err
	}

	return &Result{DefinitionID: definition.ID, Schedule: schedule, Skipped: built.Skipped}, nil
}

func (s *Service) saveDefinition(
	ctx context.Context,
	request *reporting.Request,
	name string,
	view *View,
	format report.Format,
) (*report.ReportDefinition, *Built, error) {
	built, err := Build(&reportcatalog.Default, view)
	if err != nil {
		return nil, nil, err
	}

	definition, err := s.reporting.CreateDefinition(ctx, &reporting.SaveDefinitionRequest{
		Request:       *request,
		Name:          name,
		Description:   fmt.Sprintf("Exported from the %s table", view.Resource),
		Category:      Category,
		Visibility:    report.VisibilityPrivate,
		Status:        report.DefinitionStatusActive,
		DefaultFormat: format,
		Definition:    built.Definition,
	})
	if err != nil {
		return nil, nil, err
	}

	return definition, built, nil
}
