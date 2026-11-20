package journal

import (
	"slices"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/acknowledge"
	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
)

type Direction string

const (
	FromTrenova Direction = "received"
	ToTrenova   Direction = "sent"
)

type State string

const (
	StateReceived   State = "received"
	StateUnreadable State = "unreadable"
	StateScheduled  State = "scheduled"
	StateDelivered  State = "delivered"
	StateFailed     State = "failed"
	StateCancelled  State = "cancelled"
)

type Purpose string

const (
	PurposeDocument Purpose = "document"
	PurposeAck      Purpose = "ack"
	PurposeTA1      Purpose = "ta1"
	PurposeAnswer   Purpose = "answer"
)

type AckStatus string

const (
	AckNone      AckStatus = "none"
	AckScheduled AckStatus = "scheduled"
	AckSent      AckStatus = "sent"
	AckFailed    AckStatus = "failed"
	AckWithheld  AckStatus = "withheld"
	AckAwaiting  AckStatus = "awaiting"
	AckReceived  AckStatus = "received"
)

type Transaction struct {
	Set     string `json:"set"`
	Control string `json:"control"`
	Code    string `json:"code,omitempty"`
}

type Group struct {
	ID           string        `json:"id"`
	Control      string        `json:"control"`
	Version      string        `json:"version"`
	Transactions []Transaction `json:"transactions"`
}

type Ack struct {
	Set          string        `json:"set,omitempty"`
	Status       AckStatus     `json:"status"`
	Code         string        `json:"code,omitempty"`
	MessageID    string        `json:"messageId,omitempty"`
	DueAt        *time.Time    `json:"dueAt,omitempty"`
	At           *time.Time    `json:"at,omitempty"`
	Transactions []Transaction `json:"transactions,omitempty"`
	Error        string        `json:"error,omitempty"`
}

type TA1 struct {
	Status    AckStatus  `json:"status"`
	Code      string     `json:"code,omitempty"`
	Note      string     `json:"note,omitempty"`
	MessageID string     `json:"messageId,omitempty"`
	At        *time.Time `json:"at,omitempty"`
}

type MDN struct {
	Mode      string     `json:"mode"`
	Status    string     `json:"status"`
	Signed    bool       `json:"signed,omitempty"`
	Text      string     `json:"text,omitempty"`
	At        *time.Time `json:"at,omitempty"`
	ReturnURL string     `json:"returnUrl,omitempty"`
}

type AS2 struct {
	MessageID  string `json:"messageId"`
	Signed     bool   `json:"signed"`
	Encrypted  bool   `json:"encrypted"`
	Compressed bool   `json:"compressed"`
}

type Message struct {
	ID          string                    `json:"id"`
	Partner     string                    `json:"partner"`
	Direction   Direction                 `json:"direction"`
	Purpose     Purpose                   `json:"purpose"`
	State       State                     `json:"state"`
	At          time.Time                 `json:"at"`
	DueAt       *time.Time                `json:"dueAt,omitempty"`
	DeliveredAt *time.Time                `json:"deliveredAt,omitempty"`
	PickedUpAt  *time.Time                `json:"pickedUpAt,omitempty"`
	Transport   string                    `json:"transport"`
	FileName    string                    `json:"fileName,omitempty"`
	Sets        []string                  `json:"sets"`
	Keys        []string                  `json:"keys,omitempty"`
	Interchange string                    `json:"interchange,omitempty"`
	Groups      []Group                   `json:"groups,omitempty"`
	Documents   []Document                `json:"documents,omitempty"`
	Seed        uint64                    `json:"seed,omitempty"`
	Variation   string                    `json:"variation,omitempty"`
	Resend      bool                      `json:"resend,omitempty"`
	Raw         string                    `json:"raw"`
	Judgement   *oracle.Judgement         `json:"judgement,omitempty"`
	Accepted    *bool                     `json:"accepted,omitempty"`
	Ack         *Ack                      `json:"ack,omitempty"`
	TA1         *TA1                      `json:"ta1,omitempty"`
	TA1Received []acknowledge.ReceivedTA1 `json:"ta1Received,omitempty"`
	MDN         *MDN                      `json:"mdn,omitempty"`
	AS2         *AS2                      `json:"as2,omitempty"`
	RespondsTo  string                    `json:"respondsTo,omitempty"`
	Answers     []string                  `json:"answers,omitempty"`
	Error       string                    `json:"error,omitempty"`
}

func (m *Message) Clone() Message {
	out := *m
	out.Sets = slices.Clone(m.Sets)
	out.Keys = slices.Clone(m.Keys)
	out.Answers = slices.Clone(m.Answers)
	out.TA1Received = slices.Clone(m.TA1Received)
	out.DueAt = clonePtr(m.DueAt)
	out.DeliveredAt = clonePtr(m.DeliveredAt)
	out.PickedUpAt = clonePtr(m.PickedUpAt)
	out.Accepted = clonePtr(m.Accepted)
	out.AS2 = clonePtr(m.AS2)
	if m.Documents != nil {
		out.Documents = make([]Document, len(m.Documents))
		for index := range m.Documents {
			out.Documents[index] = m.Documents[index].clone()
		}
	}
	out.Groups = make([]Group, len(m.Groups))
	for index := range m.Groups {
		out.Groups[index] = m.Groups[index]
		out.Groups[index].Transactions = slices.Clone(m.Groups[index].Transactions)
	}
	if m.Ack != nil {
		ack := *m.Ack
		ack.DueAt = clonePtr(m.Ack.DueAt)
		ack.At = clonePtr(m.Ack.At)
		ack.Transactions = slices.Clone(m.Ack.Transactions)
		out.Ack = &ack
	}
	if m.TA1 != nil {
		ta1 := *m.TA1
		ta1.At = clonePtr(m.TA1.At)
		out.TA1 = &ta1
	}
	if m.MDN != nil {
		mdn := *m.MDN
		mdn.At = clonePtr(m.MDN.At)
		out.MDN = &mdn
	}
	return out
}

func (m *Message) Summary() Summary {
	summary := Summary{
		ID:        m.ID,
		Direction: m.Direction,
		Purpose:   m.Purpose,
		State:     m.State,
		At:        m.At,
		Sets:      slices.Clone(m.Sets),
		Accepted:  clonePtr(m.Accepted),
		Variation: m.Variation,
	}
	if m.Ack != nil {
		summary.Ack = string(m.Ack.Status)
		summary.AckCode = m.Ack.Code
	}
	return summary
}

type Summary struct {
	ID        string    `json:"id"`
	Direction Direction `json:"direction"`
	Purpose   Purpose   `json:"purpose"`
	State     State     `json:"state"`
	At        time.Time `json:"at"`
	Sets      []string  `json:"sets"`
	Variation string    `json:"variation,omitempty"`
	Accepted  *bool     `json:"accepted,omitempty"`
	Ack       string    `json:"ack,omitempty"`
	AckCode   string    `json:"ackCode,omitempty"`
}

func clonePtr[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}
