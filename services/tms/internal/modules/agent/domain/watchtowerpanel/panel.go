package watchtowerpanel

import (
	"reflect"
	"strings"

	"github.com/emoss08/trenova/shared/recordids"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxPanelText   = 160
	maxPanelPapers = 8
)

// Kind names the view an item's panel is drawn with. A panel says what is
// happening on an item in the shape made for it: a stop and its window, a
// detention clock, a paper that is missing, a message, credentials coming
// due. The source fills it as it describes the item, and exactly one of the
// panel's views is set: the one its kind names.
type Kind string

const (
	KindStop     = Kind("Stop")
	KindClock    = Kind("Clock")
	KindPaper    = Kind("Paper")
	KindMessage  = Kind("Message")
	KindExpiry   = Kind("Expiry")
	KindCoverage = Kind("Coverage")
	KindContact  = Kind("Contact")
	KindRun      = Kind("Run")
	KindBooks    = Kind("Books")
	KindMoney    = Kind("Money")
	KindChange   = Kind("Change")
	KindAsk      = Kind("Ask")
	KindFinding  = Kind("Finding")
	KindFile     = Kind("File")
	KindSighting = Kind("Sighting")
)

func (k Kind) String() string { return string(k) }

func AllKinds() []Kind {
	return []Kind{
		KindStop, KindClock, KindPaper, KindMessage,
		KindExpiry, KindCoverage, KindContact, KindRun,
		KindBooks, KindMoney, KindChange, KindAsk,
		KindFinding, KindFile, KindSighting,
	}
}

type StopRole string

const (
	StopRolePickup   = StopRole("Pickup")
	StopRoleDelivery = StopRole("Delivery")
)

type StopVerdict string

const (
	StopMissed      = StopVerdict("Missed")
	StopLate        = StopVerdict("Late")
	StopAtRisk      = StopVerdict("AtRisk")
	StopArrivedLate = StopVerdict("ArrivedLate")
)

type StopPanel struct {
	Role           StopRole    `json:"role"`
	Verdict        StopVerdict `json:"verdict"`
	Place          string      `json:"place"`
	WindowEnd      int64       `json:"windowEnd,omitempty"`
	EstimatedAt    int64       `json:"estimatedAt,omitempty"`
	ArrivedAt      int64       `json:"arrivedAt,omitempty"`
	LastHeardAt    int64       `json:"lastHeardAt,omitempty"`
	MilesRemaining int64       `json:"milesRemaining,omitempty"`
	Stale          bool        `json:"stale,omitempty"`
}

type ClockPanel struct {
	Place           string `json:"place"`
	Customer        string `json:"customer,omitempty"`
	ArrivedAt       int64  `json:"arrivedAt,omitempty"`
	FreeEndsAt      int64  `json:"freeEndsAt,omitempty"`
	DepartedAt      int64  `json:"departedAt,omitempty"`
	FreeMinutes     int64  `json:"freeMinutes"`
	BillableMinutes int64  `json:"billableMinutes"`
	Accrued         string `json:"accrued,omitempty"`
}

type Paper struct {
	Name   string `json:"name"`
	OnFile bool   `json:"onFile"`
}

type PaperPanel struct {
	Customer    string  `json:"customer,omitempty"`
	DeliveredAt int64   `json:"deliveredAt,omitempty"`
	Papers      []Paper `json:"papers"`
	Holds       string  `json:"holds,omitempty"`
}

type MessagePanel struct {
	FromName    string `json:"fromName,omitempty"`
	FromAddress string `json:"fromAddress,omitempty"`
	Subject     string `json:"subject,omitempty"`
	ReadAs      string `json:"readAs,omitempty"`
	Why         string `json:"why,omitempty"`
	ReceivedAt  int64  `json:"receivedAt,omitempty"`
	Held        bool   `json:"held,omitempty"`
}

type ExpiringPaper struct {
	Name      string `json:"name"`
	ExpiresAt int64  `json:"expiresAt,omitempty"`
	DaysLeft  int64  `json:"daysLeft"`
}

type ExpiryPanel struct {
	Who    string          `json:"who"`
	Papers []ExpiringPaper `json:"papers"`
}

type CoveragePanel struct {
	StartsAt  int64  `json:"startsAt,omitempty"`
	Reason    string `json:"reason,omitempty"`
	Exhausted bool   `json:"exhausted,omitempty"`
	EndedAt   int64  `json:"endedAt,omitempty"`
}

type ContactPanel struct {
	Who         string `json:"who"`
	LastHeardAt int64  `json:"lastHeardAt,omitempty"`
	EveryHours  int64  `json:"everyHours"`
	DueAt       int64  `json:"dueAt,omitempty"`
}

type RunPanel struct {
	Agent    string `json:"agent"`
	Reason   string `json:"reason,omitempty"`
	FailedAt int64  `json:"failedAt,omitempty"`
}

type BooksPanel struct {
	System    string `json:"system"`
	Count     int64  `json:"count,omitempty"`
	Since     int64  `json:"since,omitempty"`
	ExpiresAt int64  `json:"expiresAt,omitempty"`
	Down      bool   `json:"down,omitempty"`
}

type MoneyStake string

const (
	StakeAdvance    = MoneyStake("Advance")
	StakeChargeback = MoneyStake("Chargeback")
	StakeInvoice    = MoneyStake("Invoice")
)

func (s MoneyStake) String() string { return string(s) }

type MoneyPanel struct {
	Stake  MoneyStake `json:"stake"`
	Party  string     `json:"party,omitempty"`
	Amount string     `json:"amount,omitempty"`
	Count  int64      `json:"count,omitempty"`
	DueAt  int64      `json:"dueAt,omitempty"`
	Reason string     `json:"reason,omitempty"`
}

type ChangePanel struct {
	Who  string `json:"who"`
	What string `json:"what"`
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	Ref  string `json:"ref,omitempty"`
	At   int64  `json:"at,omitempty"`
}

type AskPanel struct {
	Category string `json:"category,omitempty"`
	Tried    string `json:"tried,omitempty"`
}

type FindingPanel struct {
	Advice string `json:"advice,omitempty"`
	Detail string `json:"detail,omitempty"`
}

type FilePanel struct {
	Name       string `json:"name"`
	Reason     string `json:"reason,omitempty"`
	ReceivedAt int64  `json:"receivedAt,omitempty"`
}

type SightingPanel struct {
	Departure bool   `json:"departure,omitempty"`
	Place     string `json:"place,omitempty"`
	SeenAt    int64  `json:"seenAt,omitempty"`
	Why       string `json:"why,omitempty"`
}

type Panel struct {
	Kind     Kind           `json:"kind"`
	Stop     *StopPanel     `json:"stop,omitempty"`
	Clock    *ClockPanel    `json:"clock,omitempty"`
	Paper    *PaperPanel    `json:"paper,omitempty"`
	Message  *MessagePanel  `json:"message,omitempty"`
	Expiry   *ExpiryPanel   `json:"expiry,omitempty"`
	Coverage *CoveragePanel `json:"coverage,omitempty"`
	Contact  *ContactPanel  `json:"contact,omitempty"`
	Run      *RunPanel      `json:"run,omitempty"`
	Books    *BooksPanel    `json:"books,omitempty"`
	Money    *MoneyPanel    `json:"money,omitempty"`
	Change   *ChangePanel   `json:"change,omitempty"`
	Ask      *AskPanel      `json:"ask,omitempty"`
	Finding  *FindingPanel  `json:"finding,omitempty"`
	File     *FilePanel     `json:"file,omitempty"`
	Sighting *SightingPanel `json:"sighting,omitempty"`
}

const maxPanelProse = 600

// plain is text as a person reads it on the Watchtower: with no internal
// record id in it. A panel is stored with its item and drawn as it stands,
// so its words lose any id here, as the item's own do.
func plain(text string) string {
	return strings.TrimSpace(recordids.Strip(text))
}

func panelProse(text string) string {
	return stringutils.Ellipsize(plain(text), maxPanelProse)
}

func panelText(text string) string {
	return stringutils.Ellipsize(plain(text), maxPanelText)
}

func Normalize(panel *Panel) *Panel {
	if panel == nil {
		return nil
	}

	out := &Panel{Kind: panel.Kind}
	switch panel.Kind {
	case KindStop:
		if panel.Stop == nil {
			return nil
		}
		stop := *panel.Stop
		stop.Place = panelText(stop.Place)
		out.Stop = &stop
	case KindClock:
		if panel.Clock == nil {
			return nil
		}
		clock := *panel.Clock
		clock.Place = panelText(clock.Place)
		clock.Customer = panelText(clock.Customer)
		clock.Accrued = panelText(clock.Accrued)
		clock.FreeMinutes = max(clock.FreeMinutes, 0)
		clock.BillableMinutes = max(clock.BillableMinutes, 0)
		out.Clock = &clock
	case KindPaper:
		if panel.Paper == nil {
			return nil
		}
		paper := *panel.Paper
		paper.Customer = panelText(paper.Customer)
		paper.Holds = panelText(paper.Holds)
		papers := make([]Paper, 0, min(len(paper.Papers), maxPanelPapers))
		for _, entry := range paper.Papers {
			name := panelText(entry.Name)
			if name == "" || len(papers) == maxPanelPapers {
				continue
			}
			papers = append(papers, Paper{Name: name, OnFile: entry.OnFile})
		}
		if len(papers) == 0 {
			return nil
		}
		paper.Papers = papers
		out.Paper = &paper
	case KindMessage:
		if panel.Message == nil {
			return nil
		}
		message := *panel.Message
		message.FromName = panelText(message.FromName)
		message.FromAddress = stringutils.Ellipsize(
			strings.TrimSpace(message.FromAddress),
			maxPanelText,
		)
		message.Subject = panelText(message.Subject)
		message.ReadAs = panelText(message.ReadAs)
		message.Why = panelText(message.Why)
		out.Message = &message
	case KindExpiry:
		if panel.Expiry == nil {
			return nil
		}
		expiry := *panel.Expiry
		expiry.Who = panelText(expiry.Who)
		papers := make([]ExpiringPaper, 0, min(len(expiry.Papers), maxPanelPapers))
		for _, entry := range expiry.Papers {
			name := panelText(entry.Name)
			if name == "" || len(papers) == maxPanelPapers {
				continue
			}
			papers = append(papers, ExpiringPaper{
				Name:      name,
				ExpiresAt: max(entry.ExpiresAt, 0),
				DaysLeft:  entry.DaysLeft,
			})
		}
		if len(papers) == 0 {
			return nil
		}
		expiry.Papers = papers
		out.Expiry = &expiry
	case KindCoverage:
		if panel.Coverage == nil {
			return nil
		}
		coverage := *panel.Coverage
		coverage.Reason = panelText(coverage.Reason)
		out.Coverage = &coverage
	case KindContact:
		if panel.Contact == nil {
			return nil
		}
		contact := *panel.Contact
		contact.Who = panelText(contact.Who)
		contact.EveryHours = max(contact.EveryHours, 0)
		out.Contact = &contact
	case KindRun:
		if panel.Run == nil {
			return nil
		}
		run := *panel.Run
		run.Agent = panelText(run.Agent)
		run.Reason = panelText(run.Reason)
		out.Run = &run
	case KindBooks:
		if panel.Books == nil {
			return nil
		}
		books := *panel.Books
		books.System = panelText(books.System)
		books.Count = max(books.Count, 0)
		out.Books = &books
	case KindMoney:
		if panel.Money == nil {
			return nil
		}
		money := *panel.Money
		money.Party = panelText(money.Party)
		money.Amount = panelText(money.Amount)
		money.Reason = panelProse(money.Reason)
		money.Count = max(money.Count, 0)
		out.Money = &money
	case KindChange:
		if panel.Change == nil {
			return nil
		}
		change := *panel.Change
		change.Who = panelText(change.Who)
		change.What = panelText(change.What)
		change.From = panelText(change.From)
		change.To = panelText(change.To)
		change.Ref = panelText(change.Ref)
		out.Change = &change
	case KindAsk:
		if panel.Ask == nil {
			return nil
		}
		ask := *panel.Ask
		ask.Category = panelText(ask.Category)
		ask.Tried = panelProse(ask.Tried)
		out.Ask = &ask
	case KindFinding:
		if panel.Finding == nil {
			return nil
		}
		finding := *panel.Finding
		finding.Advice = panelProse(finding.Advice)
		finding.Detail = panelProse(finding.Detail)
		if finding.Advice == "" && finding.Detail == "" {
			return nil
		}
		out.Finding = &finding
	case KindFile:
		if panel.File == nil {
			return nil
		}
		file := *panel.File
		file.Name = panelText(file.Name)
		file.Reason = panelProse(file.Reason)
		out.File = &file
	case KindSighting:
		if panel.Sighting == nil {
			return nil
		}
		sighting := *panel.Sighting
		sighting.Place = panelText(sighting.Place)
		sighting.Why = panelProse(sighting.Why)
		out.Sighting = &sighting
	default:
		return nil
	}

	return out
}

func Same(a, b *Panel) bool {
	return reflect.DeepEqual(a, b)
}

func (r StopRole) String() string { return string(r) }

func (v StopVerdict) String() string { return string(v) }
