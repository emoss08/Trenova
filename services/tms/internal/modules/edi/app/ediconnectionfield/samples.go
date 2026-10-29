package ediconnectionfield

import (
	"context"
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/x12field"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediconnectiondraftports"
)

const (
	isaElementCount = 16
	maxSamples      = 10
	maxSampleBytes  = 64 * 1024
)

type Sample struct {
	Page        int
	Raw         string
	Parsed      *ediinboundservice.ParsedSample
	Err         error
	Direction   edidocument.DocumentDirection
	Identifying bool
}

func (p *Pages) joined() (text string, starts, numbers []int) {
	var builder strings.Builder
	starts = make([]int, 0, len(p.order))
	for _, number := range p.order {
		if builder.Len() > 0 {
			builder.WriteByte('\n')
		}
		starts = append(starts, builder.Len())
		builder.WriteString(p.pages[number])
	}
	return builder.String(), starts, p.order
}

func pageAt(offset int, starts, numbers []int) int {
	page := numbers[0]
	for index, start := range starts {
		if start > offset {
			break
		}
		page = numbers[index]
	}
	return page
}

func FindSamples(pages *Pages) []*Sample {
	if len(pages.order) == 0 {
		return nil
	}
	text, starts, numbers := pages.joined()
	samples := make([]*Sample, 0, 2)
	for offset := 0; offset < len(text) && len(samples) < maxSamples; {
		index := strings.Index(text[offset:], "ISA")
		if index < 0 {
			break
		}
		start := offset + index
		raw, end, ok := interchangeAt(text, start)
		if !ok {
			offset = start + 3
			continue
		}
		samples = append(samples, &Sample{
			Page: pageAt(start, starts, numbers),
			Raw:  raw,
		})
		offset = end
	}
	return samples
}

func interchangeAt(text string, start int) (raw string, end int, found bool) {
	if start+4 > len(text) || (start > 0 && isWordByte(text[start-1])) {
		return "", 0, false
	}
	separator := text[start+3]
	if isWordByte(separator) || separator == ' ' || separator == '\n' || separator == '\r' {
		return "", 0, false
	}

	count := 0
	cursor := start + 3
	for cursor < len(text) && count < isaElementCount {
		if text[cursor] == separator {
			count++
		}
		cursor++
	}
	if count < isaElementCount || cursor+1 >= len(text) {
		return "", 0, false
	}
	terminator := text[cursor+1]
	if terminator == '\r' {
		terminator = '\n'
	}

	limit := min(len(text), start+maxSampleBytes)
	trailer := "IEA" + string(separator)
	tail := strings.Index(text[start:limit], trailer)
	if tail < 0 {
		return "", 0, false
	}
	tail += start
	stop := strings.IndexByte(text[tail:limit], terminator)
	end = limit
	if stop >= 0 {
		end = tail + stop + 1
	}

	return cleanSample(text[start:end], terminator), end, true
}

func isWordByte(b byte) bool {
	return unicode.IsLetter(rune(b)) || unicode.IsDigit(rune(b))
}

func cleanSample(raw string, terminator byte) string {
	if terminator == '\n' {
		return strings.ReplaceAll(raw, "\r", "")
	}
	lines := strings.Split(strings.ReplaceAll(raw, "\r", ""), "\n")
	for index, line := range lines {
		lines[index] = strings.TrimRight(line, " \t")
	}
	return strings.Join(lines, "")
}

func (s *Sample) Parse(ctx context.Context) {
	s.Parsed, s.Err = ediinboundservice.ParseSample(ctx, s.Raw)
}

func (s *Sample) TransactionSet() edidocument.TransactionSet {
	if s.Parsed == nil || len(s.Parsed.Transactions) == 0 {
		return ""
	}
	return s.Parsed.Transactions[0].TransactionSet
}

func (s *Sample) ISASegment() string {
	if s.Parsed == nil {
		return ""
	}
	separator := s.Parsed.Separators.Segment
	if separator == "" {
		return ""
	}
	index := strings.Index(s.Raw, separator)
	if index < 0 {
		return ""
	}
	return s.Raw[:index+len(separator)]
}

func (s *Sample) source(note string) ediconnectiondraftports.DraftSource {
	return ediconnectiondraftports.DraftSource{
		Kind:  ediconnectiondraftports.SourceKindSample,
		Page:  s.Page,
		Quote: ClampQuote(x12field.RedactSecrets(s.ISASegment())),
		Note:  note,
	}
}

func (s *Sample) Orient(partnerIDs []string) {
	if s.Parsed == nil {
		return
	}
	for _, id := range partnerIDs {
		switch id {
		case s.Parsed.SenderID:
			s.Direction = edidocument.DocumentDirectionInbound
			s.Identifying = true
			return
		case s.Parsed.ReceiverID:
			s.Direction = edidocument.DocumentDirectionOutbound
			s.Identifying = true
			return
		}
	}
	switch s.TransactionSet() {
	case edidocument.TransactionSet204:
		s.Direction = edidocument.DocumentDirectionInbound
		s.Identifying = true
	case edidocument.TransactionSet210,
		edidocument.TransactionSet214,
		edidocument.TransactionSet990:
		s.Direction = edidocument.DocumentDirectionOutbound
		s.Identifying = true
	case edidocument.TransactionSet997, edidocument.TransactionSet999:
	}
}

type sampleParties struct {
	partnerQualifier string
	partnerID        string
	ourQualifier     string
	ourID            string
	partnerGS        string
	ourGS            string
}

func (s *Sample) parties() sampleParties {
	p := s.Parsed
	if s.Direction == edidocument.DocumentDirectionOutbound {
		return sampleParties{
			partnerQualifier: p.ReceiverQualifier,
			partnerID:        p.ReceiverID,
			ourQualifier:     p.SenderQualifier,
			ourID:            p.SenderID,
			partnerGS:        p.ApplicationReceiver,
			ourGS:            p.ApplicationSender,
		}
	}
	return sampleParties{
		partnerQualifier: p.SenderQualifier,
		partnerID:        p.SenderID,
		ourQualifier:     p.ReceiverQualifier,
		ourID:            p.ReceiverID,
		partnerGS:        p.ApplicationSender,
		ourGS:            p.ApplicationReceiver,
	}
}

func (s *Sample) EnvelopeValues() map[string]string {
	parties := s.parties()
	return map[string]string{
		PartnerQualifier: parties.partnerQualifier,
		PartnerID:        parties.partnerID,
		OurQualifier:     parties.ourQualifier,
		OurID:            parties.ourID,
		PartnerGSID:      parties.partnerGS,
		OurGSID:          parties.ourGS,
		UsageIndicator:   s.Parsed.UsageIndicator,
		X12Version:       s.Parsed.GroupVersion,
		ElementSep:       s.Parsed.Separators.Element,
		SegmentSep:       s.Parsed.Separators.Segment,
		ComponentSep:     s.Parsed.Separators.Component,
	}
}

func (s *Sample) Contribute(e *Evidence) {
	if s.Parsed == nil || !s.Identifying {
		return
	}
	note := "Read from the sample " + string(s.TransactionSet()) + " in the document"
	values := s.EnvelopeValues()
	if repetition := s.Parsed.Separators.Repetition; repetition != "" &&
		!strings.EqualFold(repetition, "U") {
		values[RepetitionSep] = repetition
	}
	for _, key := range SampleFieldOrder {
		value := values[key]
		if value == "" {
			continue
		}
		spec := Catalog[key]
		normalized, err := spec.Normalize(value)
		if err != nil {
			continue
		}
		e.Add(key, &Candidate{
			Value:      normalized,
			confidence: 1,
			source:     s.source(note),
			Status:     ediconnectiondraftports.ValueStatusAccepted,
		})
	}
}

var SampleFieldOrder = []string{
	PartnerQualifier, PartnerID, OurQualifier, OurID,
	PartnerGSID, OurGSID, UsageIndicator, X12Version,
	ElementSep, SegmentSep, ComponentSep, RepetitionSep,
}
