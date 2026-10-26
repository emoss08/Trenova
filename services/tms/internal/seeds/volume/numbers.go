package volume

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant/tenantdb"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type sequenceStore struct {
	db *keel.DB
}

func newNumberer(db *keel.DB) *numberer {
	formats := &formatProvider{db: db}

	return &numberer{
		generator: seqgen.NewGenerator(seqgen.GeneratorParams{
			Store:    &sequenceStore{db: db},
			Provider: formats,
			Logger:   zap.NewNop(),
		}),
		formats: formats,
	}
}

func (s *sequenceStore) GetNextSequence(
	ctx context.Context,
	req *seqgen.SequenceRequest,
) (int64, error) {
	next, err := s.GetNextSequenceBatch(ctx, &seqgen.SequenceRequest{
		Type:  req.Type,
		OrgID: req.OrgID,
		BuID:  req.BuID,
		Year:  req.Year,
		Month: req.Month,
		Count: 1,
	})
	if err != nil {
		return 0, err
	}
	if len(next) == 0 {
		return 0, seqgen.ErrNoSequencesReturned
	}

	return next[0], nil
}

func (s *sequenceStore) GetNextSequenceBatch(
	ctx context.Context,
	req *seqgen.SequenceRequest,
) ([]int64, error) {
	if req.Count <= 0 {
		return []int64{}, nil
	}

	sq := tenantdb.Sequences
	current, err := keel.WriteValue(ctx, s.db, func(ctx context.Context) (int64, error) {
		row := &tenant.Sequence{
			ID:              pulid.MustNew("seq_"),
			SequenceType:    req.Type,
			OrganizationID:  req.OrgID,
			BusinessUnitID:  req.BuID,
			Year:            int16(req.Year),  //nolint:gosec // a calendar year
			Month:           int16(req.Month), //nolint:gosec // a calendar month
			CurrentSequence: int64(req.Count),
		}
		if _, err := sq.Upsert(ctx, s.db, []*tenant.Sequence{row}, keel.OnConflict(
			sq.SequenceType,
			sq.OrganizationID,
			sq.BusinessUnitID,
			sq.Year,
			sq.Month,
		).Set(sq.CurrentSequence.Increment(int64(req.Count)))); err != nil {
			return 0, err
		}

		stored, err := sq.Pick(sq.CurrentSequence).Where(
			sq.SequenceType.Eq(req.Type),
			sq.Year.Eq(row.Year),
			sq.Month.Eq(row.Month),
		).One(ctx, s.db)
		if err != nil {
			return 0, err
		}

		return stored.CurrentSequence, nil
	})
	if err != nil {
		return nil, fmt.Errorf("reserve %d %s numbers: %w", req.Count, req.Type, err)
	}

	start := current - int64(req.Count) + 1
	out := make([]int64, req.Count)
	for i := range out {
		out[i] = start + int64(i)
	}

	return out, nil
}

func (s *sequenceStore) UpdateLastGenerated(
	ctx context.Context,
	req *seqgen.LastGeneratedRequest,
) error {
	sq := tenantdb.Sequences
	if _, err := sq.Where(
		sq.SequenceType.Eq(req.Type),
		sq.Year.Eq(int16(req.Year)),   //nolint:gosec // a calendar year
		sq.Month.Eq(int16(req.Month)), //nolint:gosec // a calendar month
	).Update(ctx, s.db, sq.LastGenerated.To(req.Value)); err != nil {
		return fmt.Errorf("record the last %s number: %w", req.Type, err)
	}

	return nil
}

type formatProvider struct {
	db *keel.DB
}

func (p *formatProvider) GetFormat(
	ctx context.Context,
	sequenceType tenant.SequenceType,
	_, _ pulid.ID,
) (*tenant.SequenceFormat, error) {
	sc := tenantdb.SequenceConfigs
	cfg, err := sc.Where(sc.SequenceType.Eq(sequenceType)).First(ctx, p.db)
	if errors.Is(err, keel.ErrNotFound) {
		return tenant.DefaultSequenceFormat(sequenceType)
	}
	if err != nil {
		return nil, fmt.Errorf("find %s sequence config: %w", sequenceType, err)
	}

	return cfg.Format(), nil
}

type numberer struct {
	generator seqgen.Generator
	formats   seqgen.FormatProvider
	orgID     pulid.ID
	buID      pulid.ID
	buCode    string
}

type numberSlot struct {
	At           int64
	LocationCode string
}

type numberGroup struct {
	at      time.Time
	code    string
	indexes []int
}

// allocate issues one number per slot from the tenant's real sequence, in the
// order the slots happened. Slots are grouped by every date part and the
// location code the tenant's format prints, so each number reads exactly as
// the application would have issued it at that moment.
func (n *numberer) allocate(
	ctx context.Context,
	typ tenant.SequenceType,
	slots []numberSlot,
) ([]string, error) {
	if len(slots) == 0 {
		return nil, nil
	}

	format, err := n.formats.GetFormat(ctx, typ, n.orgID, n.buID)
	if err != nil {
		return nil, fmt.Errorf("get %s format: %w", typ, err)
	}
	byDay, byWeek, byLocation := formatParts(format)

	groups := make(map[string]*numberGroup)
	for i, slot := range slots {
		at := time.Unix(slot.At, 0).UTC()
		key := at.Format("2006-01")
		switch {
		case byDay:
			key = at.Format("2006-01-02")
		case byWeek:
			year, week := at.ISOWeek()
			key = fmt.Sprintf("%s-w%d-%d", key, year, week)
		}
		code := ""
		if byLocation {
			code = slot.LocationCode
			key += "|" + code
		}
		group, ok := groups[key]
		if !ok {
			group = &numberGroup{at: at, code: code}
			groups[key] = group
		}
		group.indexes = append(group.indexes, i)
	}

	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]string, len(slots))
	for _, key := range keys {
		group := groups[key]
		sort.SliceStable(group.indexes, func(a, b int) bool {
			return slots[group.indexes[a]].At < slots[group.indexes[b]].At
		})
		numbers, genErr := n.generator.GenerateBatch(ctx, &seqgen.GenerateRequest{
			Type:             typ,
			OrgID:            n.orgID,
			BuID:             n.buID,
			Count:            len(group.indexes),
			Time:             group.at,
			LocationCode:     group.code,
			BusinessUnitCode: n.buCode,
		})
		if genErr != nil {
			return nil, fmt.Errorf("generate %s numbers: %w", typ, genErr)
		}
		for i, idx := range group.indexes {
			out[idx] = numbers[i]
		}
	}

	return out, nil
}

func formatParts(format *tenant.SequenceFormat) (byDay, byWeek, byLocation bool) {
	if format.AllowCustomFormat && format.CustomFormat != "" {
		return strings.Contains(format.CustomFormat, "{D}"),
			strings.Contains(format.CustomFormat, "{W}"),
			strings.Contains(format.CustomFormat, "{L}")
	}

	return format.IncludeDay, format.IncludeWeekNumber, format.IncludeLocationCode
}
