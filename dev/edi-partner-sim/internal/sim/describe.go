package sim

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/journal"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/transcript"
)

var unjudged = new(oracle.GuideSet)

func describe(raw []byte) (*oracle.Judgement, error) {
	return unjudged.Judge(raw)
}

func fill(m *journal.Message, j *oracle.Judgement) {
	for _, ic := range j.Interchanges {
		if m.Interchange == "" {
			m.Interchange = ic.ControlNumber
		}
		for _, g := range ic.Groups {
			group := journal.Group{ID: g.FunctionalID, Control: g.ControlNumber, Version: g.Version}
			for _, rep := range g.Reports {
				group.Transactions = append(group.Transactions, journal.Transaction{
					Set:     rep.Set,
					Control: rep.ControlNumber,
				})
				if !slices.Contains(m.Sets, rep.Set) {
					m.Sets = append(m.Sets, rep.Set)
				}
				if key := transcript.KeyOf(rep); key != "" && !slices.Contains(m.Keys, key) {
					m.Keys = append(m.Keys, key)
				}
			}
			m.Groups = append(m.Groups, group)
		}
		if len(ic.TA1) > 0 && !slices.Contains(m.Sets, "TA1") {
			m.Sets = append(m.Sets, "TA1")
		}
	}
	if m.Sets == nil {
		m.Sets = []string{}
	}
}

func writtenOf(j *oracle.Judgement, at time.Time) partner.Written {
	written := partner.Written{At: at}
	if len(j.Interchanges) == 0 {
		return written
	}
	ic := j.Interchanges[0]
	if stamped, err := time.ParseInLocation("0601021504", ic.Date+ic.Time, at.Location()); err == nil {
		written.At = stamped
	}
	written.Interchange, _ = strconv.ParseInt(ic.ControlNumber, 10, 64)
	for _, g := range ic.Groups {
		control, _ := strconv.ParseInt(g.ControlNumber, 10, 64)
		group := partner.WrittenGroup{ID: g.FunctionalID, Version: g.Version, Control: control}
		for _, rep := range g.Reports {
			group.Transactions = append(group.Transactions, partner.WrittenTransaction{
				Set:     rep.Set,
				Control: rep.ControlNumber,
			})
		}
		written.Groups = append(written.Groups, group)
	}
	return written
}

func fileNameFor(p *peer, j *oracle.Judgement, at time.Time) string {
	written := writtenOf(j, at)
	if p.def.Partner.Transport.SFTP != nil {
		if name, err := p.def.Partner.FileName(written); err == nil {
			return name
		}
	}
	set := "ta1"
	if len(written.Groups) > 0 && len(written.Groups[0].Transactions) > 0 {
		set = written.Groups[0].Transactions[0].Set
	}
	return fmt.Sprintf("%s-%s-%09d.edi", p.name(), set, written.Interchange)
}

func sameControl(a, b string) bool {
	return strings.TrimLeft(a, "0 ") == strings.TrimLeft(b, "0 ")
}

func (s *Server) earlierCharge(name string) func(string) (int64, bool) {
	return func(key string) (int64, bool) {
		var charge int64
		_, found := s.journal.Find(name, func(m *journal.Message) bool {
			for index := range m.Documents {
				d := &m.Documents[index]
				if d.Set == "204" && d.ChargeCents > 0 && slices.Contains(d.Keys, key) {
					charge = d.ChargeCents
					return true
				}
			}
			return false
		})
		return charge, found
	}
}
