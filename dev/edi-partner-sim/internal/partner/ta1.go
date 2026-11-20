package partner

import (
	"fmt"
	"strings"
	"time"
)

const (
	TA1Accepted         = "A"
	TA1AcceptedWithNote = "E"
	TA1Rejected         = "R"
	TA1NoError          = "000"
)

type TA1 struct {
	Control string
	Date    string
	Time    string
	Code    string
	Note    string
}

func (w *Writer) WriteTA1(at time.Time, ta1 TA1) Written {
	terminator := w.envelope.Separators.Segment + w.envelope.Terminator.Suffix()
	control := w.controls.Interchange.Next()
	var out strings.Builder
	out.WriteString(w.isa(at, control, isaVersion(w.envelope.Version), false))
	out.WriteString(terminator)
	out.WriteString(w.encode(Segment{ID: "TA1", Elements: single(
		zeroPad(ta1.Control, 9),
		ta1.Date,
		ta1.Time,
		ta1.Code,
		ta1.Note,
	)}))
	out.WriteString(terminator)
	out.WriteString(w.encode(Segment{ID: "IEA", Elements: single(
		"0",
		fmt.Sprintf("%09d", control),
	)}))
	out.WriteString(terminator)
	return Written{Bytes: []byte(out.String()), At: at, Interchange: control}
}

func zeroPad(value string, width int) string {
	if len(value) >= width {
		return value
	}
	return strings.Repeat("0", width-len(value)) + value
}
