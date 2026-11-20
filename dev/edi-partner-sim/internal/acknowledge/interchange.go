package acknowledge

import (
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

func TA1(ic *oracle.Interchange) partner.TA1 {
	ta1 := partner.TA1{
		Control: ic.ControlNumber,
		Date:    ic.Date,
		Time:    ic.Time,
		Code:    partner.TA1Accepted,
		Note:    partner.TA1NoError,
	}
	for i := range ic.Findings {
		f := &ic.Findings[i]
		if len(f.Code) != len(partner.TA1NoError) {
			continue
		}
		if f.Severity == oracle.SeverityError {
			ta1.Code, ta1.Note = partner.TA1Rejected, f.Code
			return ta1
		}
		if ta1.Code == partner.TA1Accepted {
			ta1.Code, ta1.Note = partner.TA1AcceptedWithNote, f.Code
		}
	}
	return ta1
}

type Received struct {
	Set          string                `json:"set"`
	GroupID      string                `json:"groupId"`
	GroupControl string                `json:"groupControl"`
	Code         string                `json:"code"`
	Transactions []ReceivedTransaction `json:"transactions,omitempty"`
}

type ReceivedTransaction struct {
	Set     string `json:"set"`
	Control string `json:"control"`
	Code    string `json:"code"`
}

func Read(rep *oracle.Report) (Received, bool) {
	if rep.Set != string(partner.Ack997) && rep.Set != string(partner.Ack999) {
		return Received{}, false
	}
	received := Received{Set: rep.Set}
	for index := range rep.Segments {
		segment := &rep.Segments[index]
		switch segment.ID {
		case "AK1":
			received.GroupID = segment.Element(1)
			received.GroupControl = segment.Element(2)
		case "AK2":
			received.Transactions = append(received.Transactions, ReceivedTransaction{
				Set:     segment.Element(1),
				Control: segment.Element(2),
			})
		case "AK5", "IK5":
			if last := len(received.Transactions) - 1; last >= 0 {
				received.Transactions[last].Code = segment.Element(1)
			}
		case "AK9":
			received.Code = segment.Element(1)
		}
	}
	return received, received.GroupControl != ""
}

type ReceivedTA1 struct {
	Control string `json:"control"`
	Code    string `json:"code"`
	Note    string `json:"note"`
}

func ReadTA1(ic *oracle.Interchange) []ReceivedTA1 {
	out := make([]ReceivedTA1, 0, len(ic.TA1))
	for index := range ic.TA1 {
		segment := &ic.TA1[index]
		out = append(out, ReceivedTA1{
			Control: segment.Element(1),
			Code:    segment.Element(4),
			Note:    segment.Element(5),
		})
	}
	return out
}
