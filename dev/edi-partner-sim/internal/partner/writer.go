package partner

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Counter struct {
	mu   sync.Mutex
	next int64
}

func NewCounter(start int64) *Counter {
	return &Counter{next: max(start, 1)}
}

func (c *Counter) Peek() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.next
}

func (c *Counter) Next() int64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	value := c.next
	c.next++
	if c.next > 999_999_999 {
		c.next = 1
	}
	return value
}

type Controls struct {
	Interchange *Counter
	Group       *Counter
}

type Transaction struct {
	Set      string
	Segments []Segment
}

type Written struct {
	Bytes       []byte
	At          time.Time
	Interchange int64
	Groups      []WrittenGroup
}

type WrittenGroup struct {
	ID           string
	Version      string
	Control      int64
	Transactions []WrittenTransaction
}

type WrittenTransaction struct {
	Set     string
	Control string
}

type Writer struct {
	envelope Envelope
	from     Identity
	to       Identity
	batching Batching
	ta1      bool
	controls Controls
}

func (p *Partner) Writer(controls Controls) *Writer {
	return &Writer{
		envelope: p.Envelope,
		from:     p.Them,
		to:       p.Trenova,
		batching: p.Batching,
		ta1:      p.Acknowledgment.TA1,
		controls: controls,
	}
}

func (w *Writer) Write(at time.Time, transactions []Transaction) ([]Written, error) {
	groups, err := w.batch(transactions)
	if err != nil {
		return nil, err
	}
	var out []Written
	start := 0
	for start < len(groups) {
		end := start + 1
		for end < len(groups) && end-start < w.batching.GroupsPerInterchange &&
			isaVersion(groups[end].version) == isaVersion(groups[start].version) {
			end++
		}
		out = append(out, w.interchange(at, groups[start:end]))
		start = end
	}
	return out, nil
}

type pendingGroup struct {
	id           string
	version      string
	transactions []Transaction
}

func isaVersion(version string) string {
	return version[:5]
}

func (w *Writer) batch(transactions []Transaction) ([]pendingGroup, error) {
	var groups []pendingGroup
	for _, transaction := range transactions {
		id, ok := groupOf[transaction.Set]
		if !ok {
			return nil, fmt.Errorf("x12 writer: %q has no functional group", transaction.Set)
		}
		version := w.envelope.VersionOf(transaction.Set)
		last := len(groups) - 1
		if last < 0 || groups[last].id != id || groups[last].version != version ||
			len(groups[last].transactions) >= w.batching.TransactionsPerGroup {
			groups = append(groups, pendingGroup{id: id, version: version})
			last++
		}
		groups[last].transactions = append(groups[last].transactions, transaction)
	}
	return groups, nil
}

func (w *Writer) interchange(at time.Time, groups []pendingGroup) Written {
	separators := w.envelope.Separators
	terminator := separators.Segment + w.envelope.Terminator.Suffix()
	control := w.controls.Interchange.Next()
	written := Written{At: at, Interchange: control}
	var out strings.Builder
	write := func(segment Segment) {
		out.WriteString(w.encode(segment))
		out.WriteString(terminator)
	}
	out.WriteString(w.isa(at, control, isaVersion(groups[0].version), w.ta1))
	out.WriteString(terminator)
	for _, group := range groups {
		groupControl := w.controls.Group.Next()
		record := WrittenGroup{ID: group.id, Version: group.version, Control: groupControl}
		write(Segment{ID: "GS", Elements: single(
			group.id,
			w.from.Group,
			w.to.Group,
			at.Format("20060102"),
			at.Format("1504"),
			strconv.FormatInt(groupControl, 10),
			"X",
			group.version,
		)})
		for index, transaction := range group.transactions {
			number := fmt.Sprintf("%04d", index+1)
			record.Transactions = append(
				record.Transactions,
				WrittenTransaction{Set: transaction.Set, Control: number},
			)
			header := single(transaction.Set, number)
			if len(group.version) > len("004010") {
				header = single(transaction.Set, number, group.version)
			}
			write(Segment{ID: "ST", Elements: header})
			for _, segment := range transaction.Segments {
				write(segment)
			}
			write(Segment{ID: "SE", Elements: single(
				strconv.Itoa(len(transaction.Segments)+2),
				number,
			)})
		}
		write(Segment{ID: "GE", Elements: single(
			strconv.Itoa(len(group.transactions)),
			strconv.FormatInt(groupControl, 10),
		)})
		written.Groups = append(written.Groups, record)
	}
	write(Segment{ID: "IEA", Elements: single(
		strconv.Itoa(len(groups)),
		fmt.Sprintf("%09d", control),
	)})
	written.Bytes = []byte(out.String())
	return written
}

func (w *Writer) isa(at time.Time, control int64, version string, ta1 bool) string {
	separators := w.envelope.Separators
	pad := func(value string, width int) string {
		if w.envelope.SqueezedISA {
			return value
		}
		return fmt.Sprintf("%-*s", width, value)
	}
	repetition := separators.Repetition
	if repetition == "" {
		repetition = "U"
	}
	acknowledgment := "0"
	if ta1 {
		acknowledgment = "1"
	}
	return strings.Join([]string{
		"ISA",
		"00", pad("", 10),
		"00", pad("", 10),
		w.from.Qualifier, pad(w.from.ID, 15),
		w.to.Qualifier, pad(w.to.ID, 15),
		at.Format("060102"),
		at.Format("1504"),
		repetition,
		version,
		fmt.Sprintf("%09d", control),
		acknowledgment,
		w.envelope.Usage,
		separators.Component,
	}, separators.Element)
}

func (w *Writer) encode(segment Segment) string {
	separators := w.envelope.Separators
	var pairs []string
	for _, separator := range []string{
		separators.Element,
		separators.Segment,
		separators.Component,
		separators.Repetition,
	} {
		if separator != "" {
			pairs = append(pairs, separator, " ")
		}
	}
	clean := strings.NewReplacer(pairs...)
	elements := make([]string, 0, len(segment.Elements))
	for _, components := range segment.Elements {
		values := make([]string, 0, len(components))
		for _, component := range components {
			values = append(values, clean.Replace(component))
		}
		elements = append(elements, strings.Join(trimTrailing(values), separators.Component))
	}
	elements = trimTrailing(elements)
	if len(elements) == 0 {
		return segment.ID
	}
	return segment.ID + separators.Element + strings.Join(elements, separators.Element)
}

func trimTrailing(values []string) []string {
	end := len(values)
	for end > 0 && values[end-1] == "" {
		end--
	}
	return values[:end]
}

func single(values ...string) [][]string {
	out := make([][]string, 0, len(values))
	for _, value := range values {
		out = append(out, []string{value})
	}
	return out
}
