package sim

import (
	"fmt"
	"net/http"
	"slices"
	"strings"

	"github.com/emoss08/trenova/edi-partner-sim/internal/acknowledge"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/transcript"
)

const (
	OverrideWithhold = "withhold"
	OverrideReject   = "reject"
)

type AckOverride struct {
	Set    string `json:"set,omitempty"`
	Key    string `json:"key,omitempty"`
	Action string `json:"action"`
}

type AckOverrideRequest struct {
	Overrides []AckOverride `json:"overrides"`
	Replace   bool          `json:"replace,omitempty"`
}

func (o AckOverride) matches(set, key string) bool {
	return (o.Set == "" || o.Set == set) && (o.Key == "" || o.Key == key)
}

func (s *Server) OverrideAcks(name string, req *AckOverrideRequest) ([]AckOverride, error) {
	p, ok := s.peer(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnknownPartner, name)
	}
	if p.def.Partner.Acknowledgment.Sends == partner.AckNone {
		return nil, fmt.Errorf("%w: %s sends no acknowledgment to withhold or reject",
			ErrConflict, name)
	}
	entries := make([]AckOverride, 0, len(req.Overrides))
	for index, override := range req.Overrides {
		action := strings.TrimSpace(override.Action)
		if action != OverrideWithhold && action != OverrideReject {
			return nil, fmt.Errorf("%w: overrides[%d].action %q is not withhold or reject",
				ErrInvalid, index, override.Action)
		}
		entries = append(entries, AckOverride{
			Set:    strings.TrimSpace(override.Set),
			Key:    strings.TrimSpace(override.Key),
			Action: action,
		})
	}
	p.mu.Lock()
	if req.Replace {
		p.overrides = nil
	}
	p.overrides = append(p.overrides, entries...)
	queued := slices.Clone(p.overrides)
	p.mu.Unlock()
	s.changed()
	return queued, nil
}

func (p *peer) takeOverride(set, key string) (AckOverride, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	index := slices.IndexFunc(p.overrides, func(o AckOverride) bool { return o.matches(set, key) })
	if index < 0 {
		return AckOverride{}, false
	}
	override := p.overrides[index]
	p.overrides = slices.Delete(p.overrides, index, index+1)
	return override, true
}

func (s *Server) applyOverrides(
	p *peer,
	j *oracle.Judgement,
	plan []acknowledge.Input,
) (withhold bool) {
	keys := map[string]string{}
	for _, ic := range j.Interchanges {
		for _, g := range ic.Groups {
			for _, rep := range g.Reports {
				keys[g.ControlNumber+"/"+rep.ControlNumber] = transcript.KeyOf(rep)
			}
		}
	}
	for index := range plan {
		input := &plan[index]
		for t := range input.Transactions {
			transaction := &input.Transactions[t]
			key := keys[input.GroupControl+"/"+transaction.Control]
			override, ok := p.takeOverride(transaction.Set, key)
			if !ok {
				continue
			}
			switch override.Action {
			case OverrideWithhold:
				withhold = true
			case OverrideReject:
				transaction.Code = acknowledge.Rejected
			}
		}
	}
	return withhold
}

func (s *Server) handleOverrides(w http.ResponseWriter, r *http.Request) {
	p, ok := s.requirePeer(w, r)
	if !ok {
		return
	}
	p.mu.Lock()
	queued := slices.Clone(p.overrides)
	p.mu.Unlock()
	if queued == nil {
		queued = []AckOverride{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"overrides": queued})
}

func (s *Server) handleAddOverrides(w http.ResponseWriter, r *http.Request) {
	req := new(AckOverrideRequest)
	if !decode(w, r, req) {
		return
	}
	queued, err := s.OverrideAcks(r.PathValue("partner"), req)
	if err != nil {
		writeFailure(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"overrides": queued})
}
