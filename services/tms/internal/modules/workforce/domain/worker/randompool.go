package worker

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	ErrInvalidRandomPeriod     = errors.New("invalid random testing period")
	ErrInvalidRandomDrawStatus = errors.New("invalid random draw status")
	ErrInvalidRandomEntryState = errors.New("invalid random draw entry status")
)

// DOTMinimumDrugRatePercent and DOTMinimumAlcoholRatePercent are the annual
// percentages of the average driver population FMCSA requires be tested
// (49 CFR 382.305). The administrator can raise a pool above these; a pool set
// below one of them is not a DOT pool.
const (
	DOTMinimumDrugRatePercent    = 50
	DOTMinimumAlcoholRatePercent = 10
)

// RandomSelectionMethod names the algorithm the draw used, recorded on every
// round so a selection can be re-checked years later. The substance is salted
// into the key so the drug and alcohol draws are two independent orders rather
// than the same list read twice.
const RandomSelectionMethod = "HMAC-SHA256(seed:substance, workerId)"

type RandomPeriod string

const (
	RandomPeriodMonthly    = RandomPeriod("Monthly")
	RandomPeriodQuarterly  = RandomPeriod("Quarterly")
	RandomPeriodSemiAnnual = RandomPeriod("SemiAnnual")
	RandomPeriodAnnual     = RandomPeriod("Annual")
)

func (p RandomPeriod) String() string { return string(p) }

func (p RandomPeriod) IsValid() bool {
	switch p {
	case RandomPeriodMonthly, RandomPeriodQuarterly, RandomPeriodSemiAnnual, RandomPeriodAnnual:
		return true
	default:
		return false
	}
}

// PerYear is how many draws a year this period produces, which is what turns an
// annual rate into a per-round target.
func (p RandomPeriod) PerYear() int {
	switch p {
	case RandomPeriodMonthly:
		return 12
	case RandomPeriodQuarterly:
		return 4
	case RandomPeriodSemiAnnual:
		return 2
	case RandomPeriodAnnual:
		return 1
	default:
		return 0
	}
}

// Months is the length of one period.
func (p RandomPeriod) Months() int {
	perYear := p.PerYear()
	if perYear == 0 {
		return 0
	}
	return 12 / perYear
}

type RandomDrawStatus string

const (
	RandomDrawStatusDraft     = RandomDrawStatus("Draft")
	RandomDrawStatusFinal     = RandomDrawStatus("Final")
	RandomDrawStatusCancelled = RandomDrawStatus("Cancelled")
)

func (s RandomDrawStatus) String() string { return string(s) }

func (s RandomDrawStatus) IsValid() bool {
	switch s {
	case RandomDrawStatusDraft, RandomDrawStatusFinal, RandomDrawStatusCancelled:
		return true
	default:
		return false
	}
}

type RandomEntryStatus string

const (
	RandomEntrySelected  = RandomEntryStatus("Selected")
	RandomEntryNotified  = RandomEntryStatus("Notified")
	RandomEntryCompleted = RandomEntryStatus("Completed")
	RandomEntryExcused   = RandomEntryStatus("Excused")
	RandomEntryMissed    = RandomEntryStatus("Missed")
)

func (s RandomEntryStatus) String() string { return string(s) }

func (s RandomEntryStatus) IsValid() bool {
	switch s {
	case RandomEntrySelected, RandomEntryNotified, RandomEntryCompleted, RandomEntryExcused,
		RandomEntryMissed:
		return true
	default:
		return false
	}
}

// IsOutstanding reports whether the office still owes this selection a
// collection.
func (s RandomEntryStatus) IsOutstanding() bool {
	return s == RandomEntrySelected || s == RandomEntryNotified
}

var (
	_ validationframework.TenantedEntity = (*DOTRandomPool)(nil)
	_ domaintypes.PostgresSearchable     = (*DOTRandomPool)(nil)
)

type DOTRandomPool struct {
	keel.Table     `keel:"dot_random_pools"`
	ID             pulid.ID `keel:"id,pk,autoid:drpool_"       json:"id"`
	BusinessUnitID pulid.ID `keel:"business_unit_id,pk,tenant" json:"businessUnitId"`
	OrganizationID pulid.ID `keel:"organization_id,pk,tenant"  json:"organizationId"`

	Code        string             `json:"code"        keel:"code"`
	Name        string             `json:"name"        keel:"name"`
	Description string             `json:"description" keel:"description,nullzero"`
	Status      domaintypes.Status `json:"status"      keel:"status"`

	Period              RandomPeriod `json:"period"              keel:"period"`
	DrugRatePercent     int16        `json:"drugRatePercent"     keel:"drug_rate_percent"`
	AlcoholRatePercent  int16        `json:"alcoholRatePercent"  keel:"alcohol_rate_percent"`
	IncludedDriverTypes []string     `json:"includedDriverTypes" keel:"included_driver_types,jsonb"`
	IsDefault           bool         `json:"isDefault"           keel:"is_default"`

	SearchVector string `json:"-"         keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-"         keel:"rank,computed"`
	Version      int64  `json:"version"   keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (p *DOTRandomPool) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		p,
		validation.Field(
			&p.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 50).Error("Code cannot exceed 50 characters"),
		),
		validation.Field(
			&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name cannot exceed 100 characters"),
		),
		validation.Field(
			&p.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status must be either Active or Inactive"),
		),
		validation.Field(
			&p.Period,
			validation.Required.Error("Draw period is required"),
			domainvalidation.ValidEnum[RandomPeriod]("Draw period is not valid"),
		),
		validation.Field(
			&p.DrugRatePercent,
			validation.Min(int16(0)).Error("Drug rate cannot be negative"),
			validation.Max(int16(100)).Error("Drug rate cannot exceed 100%"),
		),
		validation.Field(
			&p.AlcoholRatePercent,
			validation.Min(int16(0)).Error("Alcohol rate cannot be negative"),
			validation.Max(int16(100)).Error("Alcohol rate cannot exceed 100%"),
		),
	))
}

// MeetsDOTMinimums reports whether the configured rates satisfy 382.305. A pool
// below either minimum is still usable — some carriers run a non-DOT pool for
// non-safety-sensitive staff — but it cannot be shown as evidence of DOT
// compliance.
func (p *DOTRandomPool) MeetsDOTMinimums() bool {
	return p.DrugRatePercent >= DOTMinimumDrugRatePercent &&
		p.AlcoholRatePercent >= DOTMinimumAlcoholRatePercent
}

// Includes reports whether a driver type belongs in this pool. An empty list
// means every safety-sensitive driver, which is the ordinary case.
func (p *DOTRandomPool) Includes(driverType string) bool {
	if len(p.IncludedDriverTypes) == 0 {
		return true
	}
	for _, candidate := range p.IncludedDriverTypes {
		if candidate == driverType {
			return true
		}
	}
	return false
}

// TargetsFor turns the annual rates into the number of names this round must
// draw. The rate is annual, so it is divided across the year's rounds and
// rounded up: rounding down would let a carrier test under the minimum every
// round and still call the year compliant.
func (p *DOTRandomPool) TargetsFor(poolSize int) (drug int, alcohol int) {
	perYear := p.Period.PerYear()
	if poolSize <= 0 || perYear == 0 {
		return 0, 0
	}
	return roundTarget(poolSize, int(p.DrugRatePercent), perYear),
		roundTarget(poolSize, int(p.AlcoholRatePercent), perYear)
}

func roundTarget(poolSize, ratePercent, perYear int) int {
	if ratePercent <= 0 {
		return 0
	}
	// Multiply before dividing so the per-round target is computed from the
	// annual obligation rather than from an already-rounded annual figure.
	numerator := poolSize*ratePercent + (100*perYear - 1)
	target := numerator / (100 * perYear)
	if target > poolSize {
		return poolSize
	}
	return target
}

func (p *DOTRandomPool) GetID() pulid.ID { return p.ID }

func (p *DOTRandomPool) GetCreatedAt() int64 { return p.CreatedAt }

func (p *DOTRandomPool) GetOrganizationID() pulid.ID { return p.OrganizationID }

func (p *DOTRandomPool) GetBusinessUnitID() pulid.ID { return p.BusinessUnitID }

func (p *DOTRandomPool) GetTableName() string { return "dot_random_pools" }

func (p *DOTRandomPool) GetResourceType() string { return "dot_random_pool" }

func (p *DOTRandomPool) GetResourceID() string { return p.ID.String() }

func (p *DOTRandomPool) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "drpool",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "code", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (p *DOTRandomPool) BeforeCreate(_ context.Context) error {
	if p.Status == "" {
		p.Status = domaintypes.StatusActive
	}
	if p.Period == "" {
		p.Period = RandomPeriodQuarterly
	}
	if p.IncludedDriverTypes == nil {
		p.IncludedDriverTypes = []string{}
	}

	return nil
}

func (p *DOTRandomPool) BeforeUpdate(_ context.Context) error {
	if p.IncludedDriverTypes == nil {
		p.IncludedDriverTypes = []string{}
	}

	return nil
}

var (
	_ validationframework.TenantedEntity = (*DOTRandomDraw)(nil)
)

type DOTRandomDraw struct {
	keel.Table `keel:"dot_random_draws"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:drdraw_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	PoolID         pulid.ID `json:"poolId"         keel:"pool_id"`

	PeriodKey   string           `json:"periodKey"   keel:"period_key"`
	PeriodStart int64            `json:"periodStart" keel:"period_start"`
	PeriodEnd   int64            `json:"periodEnd"   keel:"period_end"`
	Status      RandomDrawStatus `json:"status"      keel:"status"`

	PoolSize        int32 `json:"poolSize"        keel:"pool_size"`
	DrugTarget      int32 `json:"drugTarget"      keel:"drug_target"`
	AlcoholTarget   int32 `json:"alcoholTarget"   keel:"alcohol_target"`
	DrugSelected    int32 `json:"drugSelected"    keel:"drug_selected"`
	AlcoholSelected int32 `json:"alcoholSelected" keel:"alcohol_selected"`

	Seed        string   `json:"seed"        keel:"seed"`
	Method      string   `json:"method"      keel:"method"`
	Notes       string   `json:"notes"       keel:"notes,nullzero"`
	DrawnAt     int64    `json:"drawnAt"     keel:"drawn_at"`
	DrawnByID   pulid.ID `json:"drawnById"   keel:"drawn_by_id,nullzero"`
	FinalizedAt *int64   `json:"finalizedAt" keel:"finalized_at"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Pool    *DOTRandomPool        `json:"pool,omitempty"    keel:"belongs:pool_id"`
	DrawnBy *tenant.User          `json:"drawnBy,omitempty" keel:"belongs:drawn_by_id"`
	Entries []*DOTRandomDrawEntry `json:"entries,omitempty" keel:"many:draw_id"`
}

func (d *DOTRandomDraw) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		d,
		validation.Field(&d.PoolID, validation.Required.Error("Pool is required")),
		validation.Field(
			&d.PeriodKey,
			validation.Required.Error("Period is required"),
			validation.Length(1, 20).Error("Period cannot exceed 20 characters"),
		),
		validation.Field(
			&d.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[RandomDrawStatus]("Status is not valid"),
		),
		validation.Field(&d.Seed, validation.Required.Error("Seed is required")),
	))

	if d.PeriodEnd <= d.PeriodStart {
		multiErr.Add("periodEnd", errortypes.ErrInvalid, "The period must end after it starts")
	}
}

// ShortOfTarget reports how many names the round still owes. A draw can fall
// short only when the pool is smaller than the target, which is worth surfacing
// rather than hiding.
func (d *DOTRandomDraw) ShortOfTarget() (drug int32, alcohol int32) {
	drug = d.DrugTarget - d.DrugSelected
	if drug < 0 {
		drug = 0
	}
	alcohol = d.AlcoholTarget - d.AlcoholSelected
	if alcohol < 0 {
		alcohol = 0
	}
	return drug, alcohol
}

func (d *DOTRandomDraw) IsFinal() bool { return d.Status == RandomDrawStatusFinal }

func (d *DOTRandomDraw) GetID() pulid.ID { return d.ID }

func (d *DOTRandomDraw) GetCreatedAt() int64 { return d.CreatedAt }

func (d *DOTRandomDraw) GetOrganizationID() pulid.ID { return d.OrganizationID }

func (d *DOTRandomDraw) GetBusinessUnitID() pulid.ID { return d.BusinessUnitID }

func (d *DOTRandomDraw) GetTableName() string { return "dot_random_draws" }

func (d *DOTRandomDraw) GetResourceType() string { return "dot_random_draw" }

func (d *DOTRandomDraw) GetResourceID() string { return d.ID.String() }

func (d *DOTRandomDraw) BeforeCreate(_ context.Context) error {
	now := timeutils.NowUnix()
	if d.Status == "" {
		d.Status = RandomDrawStatusDraft
	}
	if d.Method == "" {
		d.Method = RandomSelectionMethod
	}
	if d.DrawnAt == 0 {
		d.DrawnAt = now
	}

	return nil
}

var (
	_ validationframework.TenantedEntity = (*DOTRandomDrawEntry)(nil)
)

type DOTRandomDrawEntry struct {
	keel.Table `keel:"dot_random_draw_entries"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:drde_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	DrawID         pulid.ID `json:"drawId"         keel:"draw_id"`
	WorkerID       pulid.ID `json:"workerId"       keel:"worker_id"`

	Substance    DOTTestSubstance  `json:"substance"    keel:"substance"`
	Rank         int32             `json:"rank"         keel:"rank"`
	Status       RandomEntryStatus `json:"status"       keel:"status"`
	NotifiedAt   *int64            `json:"notifiedAt"   keel:"notified_at"`
	CompletedAt  *int64            `json:"completedAt"  keel:"completed_at"`
	TestID       pulid.ID          `json:"testId"       keel:"test_id,nullzero"`
	ExcuseReason string            `json:"excuseReason" keel:"excuse_reason,nullzero"`

	Version   int64 `json:"version"   keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	Draw   *DOTRandomDraw `json:"draw,omitempty"   keel:"belongs:draw_id"`
	Worker *Worker        `json:"worker,omitempty" keel:"belongs:worker_id"`
}

func (e *DOTRandomDrawEntry) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		e,
		validation.Field(&e.DrawID, validation.Required.Error("Draw is required")),
		validation.Field(&e.WorkerID, validation.Required.Error("Worker is required")),
		validation.Field(
			&e.Substance,
			validation.Required.Error("Substance is required"),
			domainvalidation.ValidEnum[DOTTestSubstance]("Substance must be Drug or Alcohol"),
		),
		validation.Field(
			&e.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[RandomEntryStatus]("Status is not valid"),
		),
		validation.Field(
			&e.ExcuseReason,
			validation.Length(0, 255).Error("Reason cannot exceed 255 characters"),
		),
	))

	if e.Status == RandomEntryExcused && e.ExcuseReason == "" {
		multiErr.Add(
			"excuseReason",
			errortypes.ErrRequired,
			"Excusing a selection needs a reason on the record",
		)
	}
}

func (e *DOTRandomDrawEntry) GetID() pulid.ID { return e.ID }

func (e *DOTRandomDrawEntry) GetCreatedAt() int64 { return e.CreatedAt }

func (e *DOTRandomDrawEntry) GetOrganizationID() pulid.ID { return e.OrganizationID }

func (e *DOTRandomDrawEntry) GetBusinessUnitID() pulid.ID { return e.BusinessUnitID }

func (e *DOTRandomDrawEntry) GetTableName() string { return "dot_random_draw_entries" }

func (e *DOTRandomDrawEntry) GetResourceType() string { return "dot_random_draw_entry" }

func (e *DOTRandomDrawEntry) GetResourceID() string { return e.ID.String() }

func (e *DOTRandomDrawEntry) BeforeCreate(_ context.Context) error {
	if e.Status == "" {
		e.Status = RandomEntrySelected
	}

	return nil
}

// SelectRandom picks the first n candidates by the order the seed imposes. The
// order is a keyed hash of each worker id, so it is unpredictable before the
// seed exists and reproducible forever after: an auditor with the seed and the
// roster gets the same names in the same order. Ties are broken by id so the
// result never depends on the order the roster arrived in.
func SelectRandom(candidates []pulid.ID, seed string, n int) []pulid.ID {
	if n <= 0 || len(candidates) == 0 {
		return nil
	}

	type scored struct {
		id    pulid.ID
		score uint64
	}

	ranked := make([]scored, 0, len(candidates))
	for _, id := range candidates {
		ranked = append(ranked, scored{id: id, score: selectionScore(seed, id)})
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score != ranked[j].score {
			return ranked[i].score < ranked[j].score
		}
		return ranked[i].id < ranked[j].id
	})

	if n > len(ranked) {
		n = len(ranked)
	}
	picked := make([]pulid.ID, 0, n)
	for _, entry := range ranked[:n] {
		picked = append(picked, entry.id)
	}
	return picked
}

func selectionScore(seed string, id pulid.ID) uint64 {
	mac := hmac.New(sha256.New, []byte(seed))
	mac.Write([]byte(id.String()))
	return binary.BigEndian.Uint64(mac.Sum(nil)[:8])
}

// PeriodKeyFor names the round a moment belongs to, in the form an auditor
// reads back: "2026-Q1", "2026-M03", "2026-H1" or "2026".
func PeriodKeyFor(period RandomPeriod, at time.Time) string {
	at = at.UTC()
	switch period {
	case RandomPeriodMonthly:
		return fmt.Sprintf("%d-M%02d", at.Year(), int(at.Month()))
	case RandomPeriodQuarterly:
		return fmt.Sprintf("%d-Q%d", at.Year(), (int(at.Month())-1)/3+1)
	case RandomPeriodSemiAnnual:
		return fmt.Sprintf("%d-H%d", at.Year(), (int(at.Month())-1)/6+1)
	case RandomPeriodAnnual:
		return fmt.Sprintf("%d", at.Year())
	default:
		return ""
	}
}

// PeriodBoundsFor returns the half-open window a round covers, so a draw
// carries the dates the selections apply to rather than only a label.
func PeriodBoundsFor(period RandomPeriod, at time.Time) (start int64, end int64) {
	at = at.UTC()
	months := period.Months()
	if months == 0 {
		return 0, 0
	}
	startMonth := ((int(at.Month())-1)/months)*months + 1
	from := time.Date(at.Year(), time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
	return from.Unix(), from.AddDate(0, months, 0).Unix()
}
