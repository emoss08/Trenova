package sim

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

var (
	ErrUnknownPartner = errors.New("no such partner")
	ErrInvalid        = errors.New("invalid request")
	ErrConflict       = errors.New("conflict")
)

type SendRequest struct {
	Set       string            `json:"set"`
	Seed      uint64            `json:"seed"`
	With      []uint64          `json:"with,omitempty"`
	Variation string            `json:"variation,omitempty"`
	Sparse    bool              `json:"sparse,omitempty"`
	Delta     DeltaRequest      `json:"delta"`
	Load      *partner.LoadNeed `json:"load,omitempty"`
	Payload   string            `json:"payload,omitempty"`
}

type DeltaRequest struct {
	RatePercent float64      `json:"ratePercent,omitempty"`
	AddStop     bool         `json:"addStop,omitempty"`
	Move        *MoveRequest `json:"move,omitempty"`
}

type MoveRequest struct {
	Stop int    `json:"stop"`
	By   string `json:"by"`
}

func (r *SendRequest) variation() (partner.Variation, error) {
	kind := partner.Kind(strings.TrimSpace(r.Variation))
	if kind == "" {
		kind = partner.KindOriginal
	}
	if !slices.Contains(partner.Kinds, kind) {
		return partner.Variation{}, fmt.Errorf("%w: variation %q is not one of %v",
			ErrInvalid, r.Variation, partner.Kinds)
	}
	variation := partner.Variation{
		Kind:   kind,
		Sparse: r.Sparse,
		Delta:  partner.Delta{RatePercent: r.Delta.RatePercent, AddStop: r.Delta.AddStop},
		Need:   r.Load,
	}
	if problems := r.Load.Problems(); len(problems) > 0 {
		return partner.Variation{}, fmt.Errorf("%w: load: %s", ErrInvalid,
			strings.Join(problems, "; "))
	}
	if r.Delta.RatePercent <= -100 {
		return partner.Variation{}, fmt.Errorf("%w: delta.ratePercent %v leaves no rate",
			ErrInvalid, r.Delta.RatePercent)
	}
	if move := r.Delta.Move; move != nil {
		by, err := time.ParseDuration(move.By)
		if err != nil || move.Stop < 0 {
			return partner.Variation{}, fmt.Errorf(
				"%w: delta.move needs a stop (0 is the last) and a duration such as 24h",
				ErrInvalid,
			)
		}
		variation.Delta.Move = &partner.Move{Stop: move.Stop, By: by}
	}
	return variation, nil
}

func (s *Server) Send(
	ctx context.Context,
	name string,
	req *SendRequest,
) ([]journal.Message, error) {
	p, ok := s.peer(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownPartner, name)
	}
	if strings.TrimSpace(req.Payload) != "" {
		m, err := s.prepare(p, &outgoing{purpose: journal.PurposeDocument,
			raw: []byte(req.Payload), variation: "raw"})
		if err != nil {
			return nil, fmt.Errorf("%w: %w", ErrInvalid, err)
		}
		return []journal.Message{s.sendNow(ctx, p, m)}, nil
	}
	if req.Set == "" || req.Seed == 0 {
		return nil, fmt.Errorf("%w: set and seed are required (or a raw payload)", ErrInvalid)
	}
	variation, err := req.variation()
	if err != nil {
		return nil, err
	}
	seeds := append([]uint64{req.Seed}, req.With...)
	if variation.Kind == partner.KindDuplicate && p.def.Resends(req.Set) {
		return s.resend(ctx, p, req.Set, seeds)
	}
	sends := make([]partner.Send, 0, len(seeds))
	for _, seed := range seeds {
		sends = append(sends, partner.Send{Seed: seed, Variation: variation})
	}
	writtens, _, err := p.def.Batch(req.Set, sends, p.controls)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalid, err)
	}
	s.changed()
	out := make([]journal.Message, 0, len(writtens))
	next := 0
	for _, written := range writtens {
		count := transactionsIn(written)
		batch := seeds[next : next+count]
		next += count
		if variation.Kind == partner.KindOriginal {
			for _, seed := range batch {
				p.originals.remember(originalKey(req.Set, seed), string(written.Bytes))
			}
		}
		m, prepareErr := s.prepare(p, &outgoing{
			purpose:   journal.PurposeDocument,
			raw:       written.Bytes,
			seed:      batch[0],
			variation: variation.String(),
		})
		if prepareErr != nil {
			return out, prepareErr
		}
		out = append(out, s.sendNow(ctx, p, m))
	}
	return out, nil
}

func (s *Server) resend(
	ctx context.Context,
	p *peer,
	set string,
	seeds []uint64,
) ([]journal.Message, error) {
	var raws []string
	var firstSeeds []uint64
	for _, seed := range seeds {
		raw, ok := p.originals.recall(originalKey(set, seed))
		if !ok {
			return nil, fmt.Errorf(
				"%w: %s resends the original's bytes, and no original %s for seed %d was sent",
				ErrConflict, p.name(), set, seed)
		}
		if !slices.Contains(raws, raw) {
			raws = append(raws, raw)
			firstSeeds = append(firstSeeds, seed)
		}
	}
	out := make([]journal.Message, 0, len(raws))
	for index, raw := range raws {
		m, err := s.prepare(p, &outgoing{
			purpose:   journal.PurposeDocument,
			raw:       []byte(raw),
			seed:      firstSeeds[index],
			variation: string(partner.KindDuplicate) + " resend",
			resend:    true,
		})
		if err != nil {
			return out, err
		}
		out = append(out, s.sendNow(ctx, p, m))
	}
	return out, nil
}

func transactionsIn(written partner.Written) int {
	count := 0
	for _, group := range written.Groups {
		count += len(group.Transactions)
	}
	return count
}
