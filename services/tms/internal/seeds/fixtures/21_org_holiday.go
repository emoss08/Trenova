package fixtures

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
)

type OrgHolidaySeed struct {
}

// OrgHolidaySeed gives the development org a recurring US holiday calendar
// plus a peak-season blackout so PTO day counting and blackout validation have
// something to bite on.
//
// Depends on:
//   - Organization: the tenant the calendar belongs to
var OrgHoliday = Seeds.Add(seed.Seed{
	Name:        "org_holiday",
	Kind:        seed.Fixture,
	Description: "Seeds the organisation holiday calendar and a peak-season blackout",
	After:       []*seed.Seed{AdminAccount},
	Run:         (&OrgHolidaySeed{}).Run,
})

type seededHoliday struct {
	name        string
	month       time.Month
	day         int
	kind        worker.HolidayKind
	recurs      bool
	description string
}

func (s *OrgHolidaySeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	year := time.Now().UTC().Year()
	for _, h := range s.calendar() {
		entity := &worker.OrgHoliday{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			Name:           h.name,
			HolidayDate:    time.Date(year, h.month, h.day, 0, 0, 0, 0, time.UTC).Unix(),
			Kind:           h.kind,
			RecursAnnually: h.recurs,
			Description:    h.description,
		}
		if err = s.ensure(ctx, db, entity); err != nil {
			return fmt.Errorf("ensure holiday %q: %w", h.name, err)
		}
	}
	return nil
}

func (s *OrgHolidaySeed) calendar() []seededHoliday {
	return []seededHoliday{
		{name: "New Year's Day", month: time.January, day: 1, kind: worker.HolidayKindHoliday, recurs: true},
		{name: "Memorial Day", month: time.May, day: 25, kind: worker.HolidayKindHoliday, recurs: true},
		{name: "Independence Day", month: time.July, day: 4, kind: worker.HolidayKindHoliday, recurs: true},
		{name: "Labor Day", month: time.September, day: 7, kind: worker.HolidayKindHoliday, recurs: true},
		{name: "Thanksgiving", month: time.November, day: 26, kind: worker.HolidayKindHoliday, recurs: true},
		{name: "Christmas Day", month: time.December, day: 25, kind: worker.HolidayKindHoliday, recurs: true},
		{
			name:        "Peak season freeze",
			month:       time.December,
			day:         22,
			kind:        worker.HolidayKindBlackout,
			recurs:      false,
			description: "No new time off during the holiday freight surge",
		},
		{
			name:        "Peak season freeze",
			month:       time.December,
			day:         23,
			kind:        worker.HolidayKindBlackout,
			recurs:      false,
			description: "No new time off during the holiday freight surge",
		},
	}
}

func (s *OrgHolidaySeed) ensure(ctx context.Context, db *keel.DB, entity *worker.OrgHoliday) error {
	h := workerdb.OrgHolidays
	_, err := h.CreateOrSkip(
		ctx,
		db,
		[]*worker.OrgHoliday{entity},
		h.OrganizationID,
		h.BusinessUnitID,
		h.HolidayDate,
		h.Kind,
	)
	return err
}
