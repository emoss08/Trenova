package partner

import (
	"slices"
	"time"
)

type Role string

const (
	RoleShipper Role = "shipper"
	RoleCarrier Role = "carrier"
)

type Terminator string

const (
	TerminatorNone Terminator = "none"
	TerminatorLF   Terminator = "lf"
	TerminatorCRLF Terminator = "crlf"
)

func (t Terminator) Suffix() string {
	switch t {
	case TerminatorLF:
		return "\n"
	case TerminatorCRLF:
		return "\r\n"
	default:
		return ""
	}
}

type AckSet string

const (
	Ack997  AckSet = "997"
	Ack999  AckSet = "999"
	AckNone AckSet = "none"
)

type RejectWhen string

const (
	RejectNever  RejectWhen = "never"
	RejectErrors RejectWhen = "errors"
	RejectAny    RejectWhen = "any"
)

type Partner struct {
	Name           string         `yaml:"name"`
	Display        string         `yaml:"display"`
	Role           Role           `yaml:"role"`
	Zone           string         `yaml:"zone"`
	Them           Identity       `yaml:"them"`
	Trenova        Identity       `yaml:"trenova"`
	Envelope       Envelope       `yaml:"envelope"`
	Transport      Transport      `yaml:"transport"`
	Batching       Batching       `yaml:"batching"`
	Acknowledgment Acknowledgment `yaml:"acknowledgment"`
	Answers        *Answers       `yaml:"answers"`
	Sends          []string       `yaml:"sends"`
	Receives       []string       `yaml:"receives"`
}

type Identity struct {
	Qualifier string `yaml:"qualifier"`
	ID        string `yaml:"id"`
	Group     string `yaml:"group"`
	SCAC      string `yaml:"scac"`
}

type Separators struct {
	Element    string `yaml:"element"`
	Segment    string `yaml:"segment"`
	Component  string `yaml:"component"`
	Repetition string `yaml:"repetition"`
}

type Envelope struct {
	Version     string            `yaml:"version"`
	Versions    map[string]string `yaml:"versions"`
	Separators  Separators        `yaml:"separators"`
	Terminator  Terminator        `yaml:"terminator"`
	SqueezedISA bool              `yaml:"squeezedIsa"`
	Usage       string            `yaml:"usage"`
}

func (e *Envelope) VersionOf(set string) string {
	if version, ok := e.Versions[set]; ok {
		return version
	}
	return e.Version
}

type Transport struct {
	AS2  *AS2  `yaml:"as2"`
	SFTP *SFTP `yaml:"sftp"`
}

type AS2 struct {
	ID         string `yaml:"id"`
	Signed     bool   `yaml:"signed"`
	Encrypted  bool   `yaml:"encrypted"`
	Compressed bool   `yaml:"compressed"`
	MDN        string `yaml:"mdn"`
}

type SFTP struct {
	FileName string `yaml:"fileName"`
	Inbound  string `yaml:"inbound"`
	Outbound string `yaml:"outbound"`
}

type Batching struct {
	TransactionsPerGroup int `yaml:"transactionsPerGroup"`
	GroupsPerInterchange int `yaml:"groupsPerInterchange"`
}

type Acknowledgment struct {
	Sends   AckSet        `yaml:"sends"`
	Expects AckSet        `yaml:"expects"`
	TA1     bool          `yaml:"ta1"`
	Delay   time.Duration `yaml:"delay"`
	Reject  RejectRules   `yaml:"reject"`
	Never   float64       `yaml:"never"`
}

type AnswerPolicy string

const (
	AnswerAccept  AnswerPolicy = "accept"
	AnswerDraw    AnswerPolicy = "draw"
	AnswerDecline AnswerPolicy = "decline"
	AnswerNone    AnswerPolicy = "none"
)

type Answers struct {
	Policy  AnswerPolicy  `yaml:"policy"`
	Decline string        `yaml:"decline"`
	Pace    time.Duration `yaml:"pace"`
}

type RejectRules struct {
	When RejectWhen `yaml:"when"`
	Sets []string   `yaml:"sets"`
}

func (p *Partner) Location() *time.Location {
	location, err := time.LoadLocation(p.Zone)
	if err != nil {
		return time.UTC
	}
	return location
}

func (p *Partner) SendsSet(set string) bool {
	return slices.Contains(p.Sends, set)
}
