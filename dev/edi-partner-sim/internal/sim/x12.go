package sim

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type X12Envelope struct {
	SenderQualifier          string `json:"senderQualifier"`
	SenderID                 string `json:"senderId"`
	ReceiverQualifier        string `json:"receiverQualifier"`
	ReceiverID               string `json:"receiverId"`
	InterchangeControlNumber string `json:"interchangeControlNumber"`
	FunctionalGroupID        string `json:"functionalGroupId"`
	GroupControlNumber       string `json:"groupControlNumber"`
	TransactionSet           string `json:"transactionSet"`
	TransactionControlNumber string `json:"transactionControlNumber"`
}

func ParseX12Envelope(raw string) (*X12Envelope, error) {
	trimmed := strings.TrimSpace(raw)
	if len(trimmed) < 106 || !strings.HasPrefix(trimmed, "ISA") {
		return nil, errors.New("payload does not start with an ISA segment")
	}
	elementSeparator := string(trimmed[3])
	segmentTerminator := string(trimmed[105])

	envelope := &X12Envelope{}
	for _, segment := range strings.Split(trimmed, segmentTerminator) {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		elements := strings.Split(segment, elementSeparator)
		switch elements[0] {
		case "ISA":
			if len(elements) > 13 {
				envelope.SenderQualifier = strings.TrimSpace(elements[5])
				envelope.SenderID = strings.TrimSpace(elements[6])
				envelope.ReceiverQualifier = strings.TrimSpace(elements[7])
				envelope.ReceiverID = strings.TrimSpace(elements[8])
				envelope.InterchangeControlNumber = strings.TrimSpace(elements[13])
			}
		case "GS":
			if len(elements) > 6 {
				envelope.FunctionalGroupID = strings.TrimSpace(elements[1])
				envelope.GroupControlNumber = strings.TrimSpace(elements[6])
			}
		case "ST":
			if len(elements) > 2 && envelope.TransactionSet == "" {
				envelope.TransactionSet = strings.TrimSpace(elements[1])
				envelope.TransactionControlNumber = strings.TrimSpace(elements[2])
			}
		}
	}
	if envelope.TransactionSet == "" {
		return nil, errors.New("payload does not contain an ST segment")
	}
	return envelope, nil
}

func padISA(value string) string {
	if len(value) >= 15 {
		return value[:15]
	}
	return value + strings.Repeat(" ", 15-len(value))
}

type Build997Input struct {
	SenderID       string
	ReceiverID     string
	ControlNumber  int64
	Original       *X12Envelope
	AcceptanceCode string
}

func Build997(input Build997Input) string {
	now := time.Now().UTC()
	acceptance := input.AcceptanceCode
	if acceptance == "" {
		acceptance = "A"
	}
	isaControl := fmt.Sprintf("%09d", input.ControlNumber)
	segments := []string{
		"ISA*00*          *00*          *ZZ*" + padISA(input.SenderID) +
			"*ZZ*" + padISA(input.ReceiverID) +
			"*" + now.Format("060102") + "*" + now.Format("1504") +
			"*^*00401*" + isaControl + "*0*T*>",
		"GS*FA*" + input.SenderID + "*" + input.ReceiverID +
			"*" + now.Format("20060102") + "*" + now.Format("1504") +
			"*" + strconv.FormatInt(input.ControlNumber, 10) + "*X*004010",
		"ST*997*0001",
		"AK1*" + input.Original.FunctionalGroupID + "*" + input.Original.GroupControlNumber,
		"AK2*" + input.Original.TransactionSet + "*" + input.Original.TransactionControlNumber,
		"AK5*" + acceptance,
		"AK9*" + acceptance + "*1*1*1",
		"SE*6*0001",
		"GE*1*" + strconv.FormatInt(input.ControlNumber, 10),
		"IEA*1*" + isaControl,
	}
	return strings.Join(segments, "~") + "~"
}

type BuildLoadTenderInput struct {
	SenderID      string
	ReceiverID    string
	ControlNumber int64
	ShipmentID    string
}

func BuildLoadTender204(input BuildLoadTenderInput) string {
	now := time.Now().UTC()
	isaControl := fmt.Sprintf("%09d", input.ControlNumber)
	control := strconv.FormatInt(input.ControlNumber, 10)
	segments := []string{
		"ISA*00*          *00*          *ZZ*" + padISA(input.SenderID) +
			"*ZZ*" + padISA(input.ReceiverID) +
			"*" + now.Format("060102") + "*" + now.Format("1504") +
			"*^*00401*" + isaControl + "*0*T*>",
		"GS*SM*" + input.SenderID + "*" + input.ReceiverID +
			"*" + now.Format("20060102") + "*" + now.Format("1504") +
			"*" + control + "*X*004010",
		"ST*204*0001",
		"B2**SIML**" + input.ShipmentID + "**PP",
		"B2A*00",
		"L11*" + input.ShipmentID + "*CR",
		"S5*1*CL",
		"G62*64*" + now.Format("20060102") + "*1*" + now.Format("1504"),
		"N1*SF*Simulator Shipper*93*SIM-SHIP",
		"N3*100 Simulator Way",
		"N4*Chicago*IL*60601*US",
		"S5*2*CU",
		"G62*68*" + now.AddDate(0, 0, 2).Format("20060102") + "*1*" + now.Format("1504"),
		"N1*ST*Simulator Consignee*93*SIM-CONS",
		"N3*200 Receiver Road",
		"N4*Dallas*TX*75201*US",
		"L3*1000*G***150000*",
		"SE*15*0001",
		"GE*1*" + control,
		"IEA*1*" + isaControl,
	}
	return strings.Join(segments, "~") + "~"
}
