package sim

import (
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/reply"
)

func (s *Server) scenarioFor(p *peer, tender *reply.Tender) (reply.Scenario, bool) {
	if entry, ok := p.takeScript(tender.Key); ok {
		if entry.Silent {
			return reply.Scenario{}, false
		}
		if entry.Scenario.Seed == 0 {
			entry.Scenario.Seed = s.seedFor(p, tender.Key)
		}
		return entry.Scenario, true
	}
	policy := p.def.Partner.Answers
	seed := s.seedFor(p, tender.Key)
	switch policy.Policy {
	case partner.AnswerAccept:
		return reply.Scenario{Seed: seed}, true
	case partner.AnswerDecline:
		return reply.Scenario{Seed: seed, Decline: policy.Decline}, true
	case partner.AnswerDraw:
		return reply.DrawScenario(seed), true
	default:
		return reply.Scenario{}, false
	}
}

func (s *Server) seedFor(p *peer, key string) uint64 {
	return max(hash(p.name()+"/"+key)^s.options.Seed, 1)
}

func (s *Server) answer(p *peer, received journal.Message, tender *reply.Tender) []string {
	scenario, ok := s.scenarioFor(p, tender)
	if !ok {
		return nil
	}
	replies, err := p.carrier.Converse(tender, scenario)
	if err != nil {
		s.logger.Error("the carrier could not answer", "partner", p.name(),
			"tender", tender.Key, "error", err)
		return nil
	}
	pace := p.def.Partner.Answers.Pace
	start := s.clock.Now()
	ids := make([]string, 0, len(replies))
	for index, r := range replies {
		written, writeErr := p.carrier.Write(r, p.controls)
		if writeErr != nil {
			s.logger.Error("a carrier reply could not be written", "partner", p.name(),
				"set", r.Set, "error", writeErr)
			continue
		}
		m, prepareErr := s.prepare(p, &outgoing{
			purpose:    journal.PurposeAnswer,
			raw:        written.Bytes,
			respondsTo: received.ID,
			keys:       []string{tender.Key},
			seed:       scenario.Seed,
			variation:  scenario.String(),
		})
		if prepareErr != nil {
			s.logger.Error("a carrier reply could not be prepared", "partner", p.name(),
				"set", r.Set, "error", prepareErr)
			continue
		}
		queued := s.enqueue(p, m, start.Add(time.Duration(index+1)*pace))
		ids = append(ids, queued.ID)
	}
	return ids
}
