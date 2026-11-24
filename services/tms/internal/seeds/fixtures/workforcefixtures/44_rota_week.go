package workforcefixtures

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/timesheet/timesheetdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/weekhours"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule/workerscheduledb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	rotaSecondsPerDay   = int64(86400)
	rotaSecondsPerMin   = int64(60)
	rotaAssignedWeeks   = 8
	rotaOvertimeMinutes = int32(2400)
)

type RotaWeekSeed struct{}

type rotaTemplateDef struct {
	code     string
	name     string
	days     string
	start    int16
	duration int16
}

type rotaWorkerDef struct {
	firstName string
	lastName  string
	email     string
	license   string
	template  string
	gender    worker.Gender
}

func rotaTemplates() []rotaTemplateDef {
	return []rotaTemplateDef{
		{code: "DOCK-N", name: "Dock nights", days: "1101101", start: 1320, duration: 600},
		{code: "SHOP-D", name: "Shop days", days: "1111110", start: 360, duration: 450},
		{code: "SHOP-L", name: "Shop lates", days: "1110011", start: 840, duration: 510},
		{code: "YARD-D", name: "Yard days", days: "0111110", start: 420, duration: 480},
		{code: "YARD-P", name: "Yard part week", days: "0011000", start: 420, duration: 480},
		{code: "YARD-4", name: "Yard four days", days: "0111100", start: 420, duration: 480},
	}
}

func rotaWorkers() []rotaWorkerDef {
	return []rotaWorkerDef{
		{
			"Hector",
			"Alvarez",
			"hector.alvarez@example.com",
			"IL40112001",
			"DOCK-N",
			worker.GenderMale,
		},
		{"Janet", "Cole", "janet.cole@example.com", "IL40112002", "SHOP-D", worker.GenderFemale},
		{
			"Dmitri",
			"Volkov",
			"dmitri.volkov@example.com",
			"IL40112003",
			"SHOP-L",
			worker.GenderMale,
		},
		{"Sam", "Whitaker", "sam.whitaker@example.com", "IL40112004", "YARD-D", worker.GenderMale},
		{"Rosa", "Lind", "rosa.lind@example.com", "IL40112005", "YARD-P", worker.GenderFemale},
		{
			"Kwame",
			"Boateng",
			"kwame.boateng@example.com",
			"IL40112006",
			"YARD-4",
			worker.GenderMale,
		},
	}
}

func (s *RotaWeekSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	w := workerdb.Workers
	_, err = w.AllTenants().
		Where(w.OrganizationID.Eq(org.ID), w.Email.Eq(rotaWorkers()[0].email)).
		One(ctx, db)
	if err == nil {
		return nil
	}
	if !errors.Is(err, keel.ErrNotFound) {
		return fmt.Errorf("look for the rota crew: %w", err)
	}

	templates, err := s.createTemplates(ctx, db, org)
	if err != nil {
		return err
	}
	state, err := tenantfixtures.StateByAbbreviation(ctx, db, "IL")
	if err != nil {
		return fmt.Errorf("get state IL: %w", err)
	}

	now := timeutils.NowUnix()
	loc := timeutils.LoadLocation(org.Timezone)
	for _, def := range rotaWorkers() {
		template := templates[def.template]
		workerID, workerErr := s.createWorker(ctx, db, &rotaWorker{
			org: org, stateID: state.ID, def: def, now: now,
		})
		if workerErr != nil {
			return workerErr
		}
		if err = s.assign(ctx, db, org, workerID, template, now); err != nil {
			return err
		}
		if err = s.clockWeek(ctx, db, &rotaClock{
			org: org, workerID: workerID, template: template, now: now, loc: loc,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *RotaWeekSeed) createTemplates(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
) (map[string]*workerschedule.ShiftTemplate, error) {
	out := make(map[string]*workerschedule.ShiftTemplate, len(rotaTemplates()))
	for _, def := range rotaTemplates() {
		template := &workerschedule.ShiftTemplate{
			OrganizationID:  org.ID,
			BusinessUnitID:  org.BusinessUnitID,
			Status:          domaintypes.StatusActive,
			Code:            def.code,
			Name:            def.name,
			DaysOfWeek:      def.days,
			StartMinute:     def.start,
			DurationMinutes: def.duration,
			CycleWeeks:      1,
		}
		if err := workerscheduledb.ShiftTemplates.Create(ctx, db, template); err != nil {
			return nil, fmt.Errorf("insert shift template %s: %w", def.code, err)
		}
		out[def.code] = template
	}

	return out, nil
}

type rotaWorker struct {
	org     *tenant.Organization
	stateID pulid.ID
	def     rotaWorkerDef
	now     int64
}

func (s *RotaWeekSeed) createWorker(
	ctx context.Context,
	db *keel.DB,
	in *rotaWorker,
) (pulid.ID, error) {
	workerID := pulid.MustNew("wrk_")
	entity := &worker.Worker{
		ID:             workerID,
		BusinessUnitID: in.org.BusinessUnitID,
		OrganizationID: in.org.ID,
		StateID:        in.stateID,
		Status:         domaintypes.StatusActive,
		Type:           worker.WorkerTypeEmployee,
		DriverType:     worker.DriverTypeLocal,
		FirstName:      in.def.firstName,
		LastName:       in.def.lastName,
		AddressLine1:   "2200 S Ashland Ave",
		City:           "Chicago",
		PostalCode:     "60608",
		Email:          in.def.email,
		Gender:         in.def.gender,
		CreatedAt:      in.now,
		UpdatedAt:      in.now,
	}
	if err := workerdb.Workers.Create(ctx, db, entity); err != nil {
		return "", fmt.Errorf("insert worker %s %s: %w", in.def.firstName, in.def.lastName, err)
	}

	profile := &worker.WorkerProfile{
		ID:                 pulid.MustNew("wrkp_"),
		WorkerID:           workerID,
		BusinessUnitID:     in.org.BusinessUnitID,
		OrganizationID:     in.org.ID,
		LicenseStateID:     in.stateID,
		DOB:                in.now - 35*365*rotaSecondsPerDay,
		LicenseNumber:      in.def.license,
		CDLClass:           worker.CDLClassA,
		Endorsement:        worker.EndorsementTypeNone,
		LicenseExpiry:      in.now + 3*365*rotaSecondsPerDay,
		HireDate:           in.now - 3*365*rotaSecondsPerDay,
		ComplianceStatus:   worker.ComplianceStatusPending,
		IsQualified:        true,
		TrainingHealth:     worker.TrainingHealthCurrent,
		SafetyRating:       worker.SafetyRatingExcellent,
		SafetyScore:        100,
		DrugAlcoholStatus:  worker.DrugAlcoholUnknown,
		ReturnToDutyStatus: worker.ReturnToDutyNotRequired,
		CreatedAt:          in.now,
		UpdatedAt:          in.now,
	}
	if err := workerdb.WorkerProfiles.Create(ctx, db, profile); err != nil {
		return "", fmt.Errorf("insert worker profile for %s: %w", in.def.email, err)
	}

	return workerID, nil
}

func (s *RotaWeekSeed) assign(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
	workerID pulid.ID,
	template *workerschedule.ShiftTemplate,
	now int64,
) error {
	assignment := &workerschedule.WorkerShiftAssignment{
		OrganizationID:  org.ID,
		BusinessUnitID:  org.BusinessUnitID,
		WorkerID:        workerID,
		ShiftTemplateID: template.ID,
		EffectiveFrom:   now - rotaAssignedWeeks*7*rotaSecondsPerDay,
		Notes:           "Seeded rota",
	}
	if err := workerscheduledb.WorkerShiftAssignments.Create(ctx, db, assignment); err != nil {
		return fmt.Errorf("assign %s to %s: %w", workerID, template.Code, err)
	}

	return nil
}

type rotaClock struct {
	org      *tenant.Organization
	workerID pulid.ID
	template *workerschedule.ShiftTemplate
	now      int64
	loc      *time.Location
}

func (s *RotaWeekSeed) clockWeek(ctx context.Context, db *keel.DB, in *rotaClock) error {
	weekStart, _ := weekhours.Bounds(in.now, in.loc)
	period := weekhours.TimesheetPeriod(weekStart, in.loc)
	sheet := &timesheet.Timesheet{
		OrganizationID:           in.org.ID,
		BusinessUnitID:           in.org.BusinessUnitID,
		WorkerID:                 in.workerID,
		Status:                   timesheet.TimesheetOpen,
		PeriodStart:              period,
		PeriodEnd:                period + 7*rotaSecondsPerDay,
		OvertimeThresholdMinutes: rotaOvertimeMinutes,
	}

	start := time.Unix(weekStart, 0).In(in.loc)
	entries := make([]*timesheet.TimeClockEntry, 0, weekhours.DaysInWeek)
	var worked int32
	for offset := range weekhours.DaysInWeek {
		day := start.AddDate(0, 0, offset)
		if !in.template.WorksOnDate(day, 0) {
			continue
		}
		clockIn := time.Date(
			day.Year(), day.Month(), day.Day(), 0, int(in.template.StartMinute), 0, 0, in.loc,
		).Unix()
		clockOut := clockIn + int64(in.template.DurationMinutes)*rotaSecondsPerMin
		if clockOut > in.now {
			continue
		}
		entries = append(entries, &timesheet.TimeClockEntry{
			OrganizationID: in.org.ID,
			BusinessUnitID: in.org.BusinessUnitID,
			WorkerID:       in.workerID,
			Source:         timesheet.TimeEntrySourceClock,
			ClockedInAt:    clockIn,
			ClockedOutAt:   &clockOut,
		})
		worked += int32(in.template.DurationMinutes)
	}
	sheet.ApplyTotals(worked, 0, int32(len(entries))) //nolint:gosec // a week of shifts
	if err := timesheetdb.Timesheets.Create(ctx, db, sheet); err != nil {
		return fmt.Errorf("insert this week's timesheet for %s: %w", in.workerID, err)
	}
	if len(entries) == 0 {
		return nil
	}
	for _, entry := range entries {
		entry.TimesheetID = sheet.ID
	}
	if err := timesheetdb.TimeClockEntries.CreateMany(ctx, db, entries); err != nil {
		return fmt.Errorf("insert this week's clock entries for %s: %w", in.workerID, err)
	}

	return nil
}
