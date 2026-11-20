package oracle

type Judgement struct {
	Interchanges []*Interchange `json:"interchanges"`
	Findings     []Finding      `json:"findings,omitempty"`
}

type Interchange struct {
	ControlNumber     string    `json:"controlNumber"`
	SenderQualifier   string    `json:"senderQualifier"`
	SenderID          string    `json:"senderId"`
	ReceiverQualifier string    `json:"receiverQualifier"`
	ReceiverID        string    `json:"receiverId"`
	Version           string    `json:"version"`
	Date              string    `json:"date"`
	Time              string    `json:"time"`
	AckRequested      bool      `json:"ackRequested"`
	Offset            int       `json:"offset"`
	Findings          []Finding `json:"findings,omitempty"`
	TA1               []Segment `json:"ta1,omitempty"`
	Groups            []*Group  `json:"groups"`
	Header            Segment   `json:"-"`
	Trailer           Segment   `json:"-"`
}

type Group struct {
	FunctionalID  string    `json:"functionalId"`
	SenderCode    string    `json:"senderCode"`
	ReceiverCode  string    `json:"receiverCode"`
	ControlNumber string    `json:"controlNumber"`
	Version       string    `json:"version"`
	Findings      []Finding `json:"findings,omitempty"`
	Reports       []*Report `json:"reports"`
	Header        Segment   `json:"-"`
	Trailer       Segment   `json:"-"`
}

type Report struct {
	Set                      string    `json:"set"`
	ControlNumber            string    `json:"controlNumber"`
	Version                  string    `json:"version"`
	GroupControlNumber       string    `json:"groupControlNumber"`
	InterchangeControlNumber string    `json:"interchangeControlNumber"`
	Guide                    string    `json:"guide,omitempty"`
	GuideFile                string    `json:"guideFile,omitempty"`
	Segments                 []Segment `json:"segments"`
	Findings                 []Finding `json:"findings,omitempty"`
	Omitted                  int       `json:"omitted,omitempty"`
}

func (r *Report) Accepted() bool {
	return !hasErrors(r.Findings)
}

func (r *Report) add(f Finding) {
	if len(r.Findings) >= maxFindingsPerTransaction {
		r.Omitted++
		return
	}
	r.Findings = append(r.Findings, f)
}

func (g *Group) Accepted() bool {
	if hasErrors(g.Findings) {
		return false
	}
	for _, r := range g.Reports {
		if !r.Accepted() {
			return false
		}
	}
	return true
}

func (i *Interchange) Accepted() bool {
	if hasErrors(i.Findings) {
		return false
	}
	for _, g := range i.Groups {
		if !g.Accepted() {
			return false
		}
	}
	return true
}

func (j *Judgement) Accepted() bool {
	if hasErrors(j.Findings) || len(j.Interchanges) == 0 {
		return false
	}
	for _, i := range j.Interchanges {
		if !i.Accepted() {
			return false
		}
	}
	return true
}

func (j *Judgement) Reports() []*Report {
	var out []*Report
	for _, i := range j.Interchanges {
		for _, g := range i.Groups {
			out = append(out, g.Reports...)
		}
	}
	return out
}
