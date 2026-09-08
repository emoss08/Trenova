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

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	_ bun.BeforeAppendModelHook          = (*DOTRandomPool)(nil)
	_ validationframework.TenantedEntity = (*DOTRandomPool)(nil)
	_ domaintypes.PostgresSearchable     = (*DOTRandomPool)(nil)
)

type DOTRandomPool struct {
	bun.BaseModel             `bun:"table:dot_random_pools,alias:drpool" json:"-"`
	pagination.CursorValueSet `bun:",embed"                              json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Code        string             `json:"code"        bun:"code,type:VARCHAR(50),notnull"`
	Name        string             `json:"name"        bun:"name,type:VARCHAR(100),notnull"`
	Description string             `json:"description" bun:"description,type:TEXT,nullzero"`
	Status      domaintypes.Status `json:"status"      bun:"status,type:status_enum,notnull,default:'Active'"`

	Period              RandomPeriod `json:"period"              bun:"period,type:dot_random_period_enum,notnull,default:'Quarterly'"`
	DrugRatePercent     int16        `json:"drugRatePercent"     bun:"drug_rate_percent,type:SMALLINT,notnull"`
	AlcoholRatePercent  int16        `json:"alcoholRatePercent"  bun:"alcohol_rate_percent,type:SMALLINT,notnull"`
	IncludedDriverTypes []string     `json:"includedDriverTypes" bun:"included_driver_types,type:JSONB,notnull,default:'[]'"`
	IsDefault           bool         `json:"isDefault"           bun:"is_default,type:BOOLEAN,notnull"`

	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`
	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

func (p *DOTRandomPool) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("drpool_")
		}
		if p.Status == "" {
			p.Status = domaintypes.StatusActive
		}
		if p.Period == "" {
			p.Period = RandomPeriodQuarterly
		}
		if p.IncludedDriverTypes == nil {
			p.IncludedDriverTypes = []string{}
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		if p.IncludedDriverTypes == nil {
			p.IncludedDriverTypes = []string{}
		}
		p.UpdatedAt = now
	}

	return nil
}

var (
	_ bun.BeforeAppendModelHook          = (*DOTRandomDraw)(nil)
	_ validationframework.TenantedEntity = (*DOTRandomDraw)(nil)
)

type DOTRandomDraw struct {
	bun.BaseModel `bun:"table:dot_random_draws,alias:drdraw" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	PoolID         pulid.ID `json:"poolId"         bun:"pool_id,type:VARCHAR(100),notnull"`

	PeriodKey   string           `json:"periodKey"   bun:"period_key,type:VARCHAR(20),notnull"`
	PeriodStart int64            `json:"periodStart" bun:"period_start,type:BIGINT,notnull"`
	PeriodEnd   int64            `json:"periodEnd"   bun:"period_end,type:BIGINT,notnull"`
	Status      RandomDrawStatus `json:"status"      bun:"status,type:dot_random_draw_status_enum,notnull,default:'Draft'"`

	PoolSize        int32 `json:"poolSize"        bun:"pool_size,type:INTEGER,notnull"`
	DrugTarget      int32 `json:"drugTarget"      bun:"drug_target,type:INTEGER,notnull"`
	AlcoholTarget   int32 `json:"alcoholTarget"   bun:"alcohol_target,type:INTEGER,notnull"`
	DrugSelected    int32 `json:"drugSelected"    bun:"drug_selected,type:INTEGER,notnull"`
	AlcoholSelected int32 `json:"alcoholSelected" bun:"alcohol_selected,type:INTEGER,notnull"`

	Seed        string   `json:"seed"        bun:"seed,type:VARCHAR(64),notnull"`
	Method      string   `json:"method"      bun:"method,type:VARCHAR(60),notnull"`
	Notes       string   `json:"notes"       bun:"notes,type:TEXT,nullzero"`
	DrawnAt     int64    `json:"drawnAt"     bun:"drawn_at,type:BIGINT,notnull"`
	DrawnByID   pulid.ID `json:"drawnById"   bun:"drawn_by_id,type:VARCHAR(100),nullzero"`
	FinalizedAt *int64   `json:"finalizedAt" bun:"finalized_at,type:BIGINT,nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Pool    *DOTRandomPool        `json:"pool,omitempty"    bun:"rel:belongs-to,join:pool_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	DrawnBy *tenant.User          `json:"drawnBy,omitempty" bun:"rel:belongs-to,join:drawn_by_id=id"`
	Entries []*DOTRandomDrawEntry `json:"entries,omitempty" bun:"rel:has-many,join:id=draw_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (d *DOTRandomDraw) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("drdraw_")
		}
		if d.Status == "" {
			d.Status = RandomDrawStatusDraft
		}
		if d.Method == "" {
			d.Method = RandomSelectionMethod
		}
		if d.DrawnAt == 0 {
			d.DrawnAt = now
		}
		d.CreatedAt = now
		d.UpdatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}

var (
	_ bun.BeforeAppendModelHook          = (*DOTRandomDrawEntry)(nil)
	_ validationframework.TenantedEntity = (*DOTRandomDrawEntry)(nil)
)

type DOTRandomDrawEntry struct {
	bun.BaseModel `bun:"table:dot_random_draw_entries,alias:drde" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	DrawID         pulid.ID `json:"drawId"         bun:"draw_id,type:VARCHAR(100),notnull"`
	WorkerID       pulid.ID `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull"`

	Substance    DOTTestSubstance  `json:"substance"    bun:"substance,type:dot_test_substance_enum,notnull"`
	Rank         int32             `json:"rank"         bun:"rank,type:INTEGER,notnull"`
	Status       RandomEntryStatus `json:"status"       bun:"status,type:dot_random_entry_status_enum,notnull,default:'Selected'"`
	NotifiedAt   *int64            `json:"notifiedAt"   bun:"notified_at,type:BIGINT,nullzero"`
	CompletedAt  *int64            `json:"completedAt"  bun:"completed_at,type:BIGINT,nullzero"`
	TestID       pulid.ID          `json:"testId"       bun:"test_id,type:VARCHAR(100),nullzero"`
	ExcuseReason string            `json:"excuseReason" bun:"excuse_reason,type:VARCHAR(255),nullzero"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Draw   *DOTRandomDraw `json:"draw,omitempty"   bun:"rel:belongs-to,join:draw_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Worker *Worker        `json:"worker,omitempty" bun:"rel:belongs-to,join:worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
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

func (e *DOTRandomDrawEntry) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if e.ID.IsNil() {
			e.ID = pulid.MustNew("drde_")
		}
		if e.Status == "" {
			e.Status = RandomEntrySelected
		}
		e.CreatedAt = now
		e.UpdatedAt = now
	case *bun.UpdateQuery:
		e.UpdatedAt = now
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
