package ediconnectiondraft

import (
	"strings"
	"unicode"

	"github.com/emoss08/trenova/internal/modules/edi/app/ediinboundservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
)

const (
	isaElementCount = 16
	maxSamples      = 10
	maxSampleBytes  = 64 * 1024
)

type specSample struct {
	page        int
	raw         string
	parsed      *ediinboundservice.ParsedSample
	err         error
	direction   edi.DocumentDirection
	identifying bool
}

func (p *pageIndex) joined() (text string, starts, numbers []int) {
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

func findSamples(pages *pageIndex) []*specSample {
	if len(pages.order) == 0 {
		return nil
	}
	text, starts, numbers := pages.joined()
	samples := make([]*specSample, 0, 2)
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
		samples = append(samples, &specSample{
			page: pageAt(start, starts, numbers),
			raw:  raw,
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

func (s *specSample) parse() {
	s.parsed, s.err = ediinboundservice.ParseSample(s.raw)
}

func (s *specSample) transactionSet() edi.TransactionSet {
	if s.parsed == nil || len(s.parsed.Transactions) == 0 {
		return ""
	}
	return s.parsed.Transactions[0].TransactionSet
}

func (s *specSample) isaSegment() string {
	if s.parsed == nil {
		return ""
	}
	separator := s.parsed.Separators.Segment
	if separator == "" {
		return ""
	}
	index := strings.Index(s.raw, separator)
	if index < 0 {
		return ""
	}
	return s.raw[:index+len(separator)]
}

func (s *specSample) source(note string) ediports.DraftSource {
	return ediports.DraftSource{
		Kind:  ediports.SourceKindSample,
		Page:  s.page,
		Quote: clampQuote(redact(s.isaSegment())),
		Note:  note,
	}
}

func (s *specSample) orient(partnerIDs []string) {
	if s.parsed == nil {
		return
	}
	for _, id := range partnerIDs {
		switch id {
		case s.parsed.SenderID:
			s.direction = edi.DocumentDirectionInbound
			s.identifying = true
			return
		case s.parsed.ReceiverID:
			s.direction = edi.DocumentDirectionOutbound
			s.identifying = true
			return
		}
	}
	switch s.transactionSet() {
	case edi.TransactionSet204:
		s.direction = edi.DocumentDirectionInbound
		s.identifying = true
	case edi.TransactionSet210, edi.TransactionSet214, edi.TransactionSet990:
		s.direction = edi.DocumentDirectionOutbound
		s.identifying = true
	case edi.TransactionSet997, edi.TransactionSet999:
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

func (s *specSample) parties() sampleParties {
	p := s.parsed
	if s.direction == edi.DocumentDirectionOutbound {
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

func (s *specSample) envelopeValues() map[string]string {
	parties := s.parties()
	return map[string]string{
		fieldPartnerQualifier: parties.partnerQualifier,
		fieldPartnerID:        parties.partnerID,
		fieldOurQualifier:     parties.ourQualifier,
		fieldOurID:            parties.ourID,
		fieldPartnerGSID:      parties.partnerGS,
		fieldOurGSID:          parties.ourGS,
		fieldUsageIndicator:   s.parsed.UsageIndicator,
		fieldX12Version:       s.parsed.GroupVersion,
		fieldElementSep:       s.parsed.Separators.Element,
		fieldSegmentSep:       s.parsed.Separators.Segment,
		fieldComponentSep:     s.parsed.Separators.Component,
	}
}

func (s *specSample) contribute(e *evidence) {
	if s.parsed == nil || !s.identifying {
		return
	}
	note := "Read from the sample " + string(s.transactionSet()) + " in the document"
	values := s.envelopeValues()
	if repetition := s.parsed.Separators.Repetition; repetition != "" &&
		!strings.EqualFold(repetition, "U") {
		values[fieldRepetitionSep] = repetition
	}
	for _, key := range sampleFieldOrder {
		value := values[key]
		if value == "" {
			continue
		}
		spec := fieldCatalog[key]
		normalized, err := spec.normalize(value)
		if err != nil {
			continue
		}
		e.add(key, &candidate{
			value:      normalized,
			confidence: 1,
			source:     s.source(note),
			status:     ediports.ValueStatusAccepted,
		})
	}
}

var sampleFieldOrder = []string{
	fieldPartnerQualifier, fieldPartnerID, fieldOurQualifier, fieldOurID,
	fieldPartnerGSID, fieldOurGSID, fieldUsageIndicator, fieldX12Version,
	fieldElementSep, fieldSegmentSep, fieldComponentSep, fieldRepetitionSep,
}
