package development

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

type OrgHolidaySeed struct {
	seedhelpers.BaseSeed
}

// OrgHolidaySeed gives the development org a recurring US holiday calendar
// plus a peak-season blackout so PTO day counting and blackout validation have
// something to bite on.
//
// Depends on:
//   - Organization: the tenant the calendar belongs to
func NewOrgHolidaySeed() *OrgHolidaySeed {
	seed := &OrgHolidaySeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"OrgHoliday",
		"1.0.0",
		"Seeds the organisation holiday calendar and a peak-season blackout",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedTestOrganizations)
	return seed
}

type seededHoliday struct {
	name        string
	month       time.Month
	day         int
	kind        worker.HolidayKind
	recurs      bool
	description string
}

func (s *OrgHolidaySeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
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
				if err = s.ensure(ctx, tx, entity); err != nil {
					return fmt.Errorf("ensure holiday %q: %w", h.name, err)
				}
			}
			return nil
		},
	)
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

func (s *OrgHolidaySeed) ensure(ctx context.Context, tx bun.Tx, entity *worker.OrgHoliday) error {
	cols := buncolgen.OrgHolidayColumns
	existing := new(worker.OrgHoliday)
	err := tx.NewSelect().
		Model(existing).
		Where(cols.OrganizationID.Eq(), entity.OrganizationID).
		Where(cols.BusinessUnitID.Eq(), entity.BusinessUnitID).
		Where(cols.HolidayDate.Eq(), entity.HolidayDate).
		Where(cols.Kind.Eq(), entity.Kind).
		Limit(1).
		Scan(ctx)
	if err == nil {
		return nil
	}

	entity.ID = pulid.MustNew("ohol_")
	_, err = tx.NewInsert().Model(entity).Exec(ctx)
	return err
}
