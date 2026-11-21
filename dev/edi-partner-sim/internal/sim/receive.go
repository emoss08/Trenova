package sim

import (
	"slices"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/acknowledge"
	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/reply"
	"github.com/emoss08/trenova/edi-partner-sim/internal/transcript"
)

const acknowledgmentGroup = "FA"

type inbound struct {
	raw       []byte
	transport string
	fileName  string
	as2       *journal.AS2
	mdn       *journal.MDN
	err       string
}

func (s *Server) receive(p *peer, in *inbound) journal.Message {
	m := &journal.Message{
		Partner:   p.name(),
		Direction: journal.FromTrenova,
		Purpose:   journal.PurposeDocument,
		State:     journal.StateReceived,
		Transport: in.transport,
		FileName:  in.fileName,
		Raw:       string(in.raw),
		AS2:       in.as2,
		MDN:       in.mdn,
		Error:     in.err,
		Sets:      []string{},
	}
	if in.err != "" {
		m.State = journal.StateUnreadable
		return s.journal.Add("message.received", m)
	}
	judgement, err := p.guides.Judge(in.raw)
	if err != nil {
		m.State, m.Error = journal.StateUnreadable, err.Error()
		return s.journal.Add("message.received", m)
	}
	m.Judgement = judgement
	accepted := judgement.Accepted()
	m.Accepted = &accepted
	fill(m, judgement)
	m.Documents = transcript.Build(judgement, transcript.Options{
		Judged:  true,
		Earlier: s.earlierCharge(p.name()),
	})
	links := s.linkAcks(p, m, judgement)
	recorded := s.journal.Add("message.received", m)
	targets := make([]string, 0, len(links))
	for _, link := range links {
		targets = append(targets, link.target)
	}
	s.journal.LinkAck(recorded.ID, targets...)
	s.settleLinks(recorded, links)
	if m.Purpose == journal.PurposeAck {
		s.changed()
		return recorded
	}
	s.answerTA1(p, recorded, judgement)
	decisions := s.acknowledgeGroups(p, recorded, judgement)
	if p.carrier != nil {
		s.answerTenders(p, recorded, judgement, decisions)
	}
	s.changed()
	final, _ := s.journal.Get(recorded.ID)
	return final
}

type ackLink struct {
	target   string
	received acknowledge.Received
	ta1      *acknowledge.ReceivedTA1
}

func (s *Server) linkAcks(p *peer, m *journal.Message, j *oracle.Judgement) []ackLink {
	var links []ackLink
	documents := false
	for _, ic := range j.Interchanges {
		for _, ta1 := range acknowledge.ReadTA1(ic) {
			if sent, ok := s.journal.Find(p.name(), func(c *journal.Message) bool {
				return c.Direction == journal.ToTrenova && sameControl(c.Interchange, ta1.Control)
			}); ok {
				received := ta1
				links = append(links, ackLink{target: sent.ID, ta1: &received})
			}
		}
		for _, g := range ic.Groups {
			if g.FunctionalID != acknowledgmentGroup {
				documents = true
				continue
			}
			for _, rep := range g.Reports {
				received, ok := acknowledge.Read(rep)
				if !ok {
					continue
				}
				if sent, found := s.journal.Find(p.name(), acknowledges(received)); found {
					links = append(links, ackLink{target: sent.ID, received: received})
				}
			}
		}
	}
	if !documents && len(j.Interchanges) > 0 {
		m.Purpose = journal.PurposeAck
		if len(m.Groups) == 0 {
			m.Purpose = journal.PurposeTA1
		}
	}
	for _, link := range links {
		if m.RespondsTo == "" {
			m.RespondsTo = link.target
		}
		if target, ok := s.journal.Get(link.target); ok {
			for _, key := range target.Keys {
				if !slices.Contains(m.Keys, key) {
					m.Keys = append(m.Keys, key)
				}
			}
		}
	}
	return links
}

func acknowledges(received acknowledge.Received) func(*journal.Message) bool {
	return func(c *journal.Message) bool {
		if c.Direction != journal.ToTrenova {
			return false
		}
		return slices.ContainsFunc(c.Groups, func(g journal.Group) bool {
			return g.ID == received.GroupID && sameControl(g.Control, received.GroupControl)
		})
	}
}

func (s *Server) settleLinks(ack journal.Message, links []ackLink) {
	at := s.clock.Now().UTC()
	for _, link := range links {
		if link.ta1 != nil {
			ta1 := *link.ta1
			s.journal.Update("ta1.received", link.target, func(stored *journal.Message) {
				stored.TA1 = &journal.TA1{
					Status:    journal.AckReceived,
					Code:      ta1.Code,
					Note:      ta1.Note,
					MessageID: ack.ID,
					At:        &at,
				}
				stored.TA1Received = append(stored.TA1Received, ta1)
			})
			continue
		}
		received := link.received
		s.journal.Update("ack.received", link.target, func(stored *journal.Message) {
			if stored.Ack == nil || stored.Ack.Status != journal.AckReceived {
				stored.Ack = &journal.Ack{Set: received.Set}
			}
			stored.Ack.Status, stored.Ack.MessageID, stored.Ack.At = journal.AckReceived, ack.ID, &at
			stored.Ack.Code = worse(stored.Ack.Code, received.Code)
			for _, transaction := range received.Transactions {
				stored.Ack.Transactions = append(stored.Ack.Transactions, journal.Transaction{
					Set:     transaction.Set,
					Control: transaction.Control,
					Code:    transaction.Code,
				})
			}
		})
	}
}

var ackSeverity = map[string]int{
	acknowledge.Accepted:           1,
	acknowledge.AcceptedWithErrors: 2,
	acknowledge.Partial:            3,
	acknowledge.Rejected:           4,
}

func worse(a, b string) string {
	if ackSeverity[b] > ackSeverity[a] {
		return b
	}
	return a
}

func (s *Server) answerTA1(p *peer, received journal.Message, j *oracle.Judgement) {
	var outgoingTA1s []partner.TA1
	for _, ic := range j.Interchanges {
		if ic.AckRequested {
			outgoingTA1s = append(outgoingTA1s, acknowledge.TA1(ic))
		}
	}
	if len(outgoingTA1s) == 0 {
		return
	}
	due := s.clock.Now().Add(p.def.Partner.Acknowledgment.Delay)
	writer := p.def.Partner.Writer(p.controls)
	for _, ta1 := range outgoingTA1s {
		written := writer.WriteTA1(s.local(p), ta1)
		m, err := s.prepare(p, &outgoing{
			purpose:    journal.PurposeTA1,
			raw:        written.Bytes,
			respondsTo: received.ID,
			keys:       received.Keys,
		})
		if err != nil {
			s.logger.Error("a TA1 could not be prepared", "partner", p.name(), "error", err)
			continue
		}
		queued := s.enqueue(p, m, due)
		s.journal.LinkAck(queued.ID, received.ID)
		dueAt := due.UTC()
		s.journal.Update("ta1.scheduled", received.ID, func(stored *journal.Message) {
			stored.TA1 = &journal.TA1{
				Status:    journal.AckScheduled,
				Code:      ta1.Code,
				Note:      ta1.Note,
				MessageID: queued.ID,
				At:        &dueAt,
			}
		})
	}
}

func (s *Server) acknowledgeGroups(
	p *peer,
	received journal.Message,
	j *oracle.Judgement,
) map[string]string {
	rules := &p.def.Partner.Acknowledgment
	plan := acknowledge.Plan(rules, j)
	withhold := s.applyOverrides(p, j, plan)
	decisions := map[string]string{}
	ack := &journal.Ack{Set: string(rules.Sends), Status: journal.AckNone}
	var transactions []partner.Transaction
	for _, input := range plan {
		built, err := acknowledge.Build(rules, input)
		if err != nil {
			s.logger.Error(
				"an acknowledgment could not be built",
				"partner",
				p.name(),
				"error",
				err,
			)
			continue
		}
		transactions = append(transactions, built)
		for _, t := range input.Transactions {
			decisions[input.GroupControl+"/"+t.Control] = t.Code
			ack.Code = worse(ack.Code, t.Code)
			ack.Transactions = append(ack.Transactions, journal.Transaction{
				Set:     t.Set,
				Control: t.Control,
				Code:    t.Code,
			})
		}
	}
	switch {
	case rules.Sends == partner.AckNone || len(transactions) == 0:
	case withhold || p.silent():
		ack.Status = journal.AckWithheld
	default:
		s.queueAcks(p, received, transactions, ack)
	}
	event := ""
	if ack.Status != journal.AckNone {
		event = "ack." + string(ack.Status)
	}
	s.journal.Update(event, received.ID, func(stored *journal.Message) { stored.Ack = ack })
	return decisions
}

func (s *Server) queueAcks(
	p *peer,
	received journal.Message,
	transactions []partner.Transaction,
	ack *journal.Ack,
) {
	writtens, err := p.def.Partner.Writer(p.controls).Write(s.local(p), transactions)
	if err != nil {
		ack.Status, ack.Error = journal.AckFailed, err.Error()
		return
	}
	due := s.clock.Now().Add(p.def.Partner.Acknowledgment.Delay)
	dueAt := due.UTC()
	ack.Status, ack.DueAt = journal.AckScheduled, &dueAt
	for _, written := range writtens {
		m, prepareErr := s.prepare(p, &outgoing{
			purpose:    journal.PurposeAck,
			raw:        written.Bytes,
			respondsTo: received.ID,
			keys:       received.Keys,
		})
		if prepareErr != nil {
			ack.Status, ack.Error = journal.AckFailed, prepareErr.Error()
			return
		}
		queued := s.enqueue(p, m, due)
		s.journal.LinkAck(queued.ID, received.ID)
		if ack.MessageID == "" {
			ack.MessageID = queued.ID
		}
	}
}

func (s *Server) answerTenders(
	p *peer,
	received journal.Message,
	j *oracle.Judgement,
	decisions map[string]string,
) {
	var answered []string
	for _, ic := range j.Interchanges {
		for _, g := range ic.Groups {
			for _, rep := range g.Reports {
				if rep.Set != "204" ||
					decisions[g.ControlNumber+"/"+rep.ControlNumber] == acknowledge.Rejected {
					continue
				}
				tender, err := reply.ReadTender(rep, p.def.Partner.Location())
				if err != nil {
					s.logger.Warn("a tender could not be read", "partner", p.name(), "error", err)
					continue
				}
				if tender.Purpose != "" && tender.Purpose != "00" {
					continue
				}
				answered = append(answered, s.answer(p, received, tender)...)
			}
		}
	}
	if len(answered) > 0 {
		s.journal.Update("answer.scheduled", received.ID, func(stored *journal.Message) {
			stored.Answers = append(stored.Answers, answered...)
		})
	}
}

func (s *Server) local(p *peer) time.Time {
	return s.clock.Now().In(p.def.Partner.Location())
}
