package ediinboundservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/x12"
	"github.com/emoss08/trenova/shared/stringutils"
)

var (
	errEmptyFile     = errors.New("inbound file is empty")
	errNoSegments    = errors.New("inbound file does not contain any X12 segments")
	errNoInterchange = errors.New(
		"inbound file does not start with an ISA interchange header",
	)
	errNoTransactions             = errors.New("inbound file does not contain any X12 transactions")
	errTransactionWithoutSegments = errors.New("transaction does not contain any segments")
	errRejectedTransaction        = errors.New("rejected, so it was not processed")
)

// parseInterchange reads a file through the EDI module against the partner's guides.
// A returned problem is the file's fault and quarantines it; an error is the
// module's and is retried.
func parseInterchange(
	ctx context.Context,
	rawX12 string,
) (interchange *parsedInterchange, problem, err error) {
	if strings.TrimSpace(rawX12) == "" {
		return nil, errEmptyFile, nil
	}
	inspection, err := edix12inspect.InspectX12(
		ctx,
		&edix12inspect.InspectX12Request{RawX12: rawX12},
	)
	if err != nil {
		return nil, nil, err
	}
	reading := inspection.Reading
	if len(inspection.Segments) == 0 {
		return nil, errNoSegments, nil
	}
	if len(reading.Interchanges) == 0 || reading.Interchanges[0].Header == nil {
		return nil, errNoInterchange, nil
	}
	if reading.Summary.Transactions == 0 {
		return nil, errNoTransactions, nil
	}

	first := &reading.Interchanges[0]
	interchange = &parsedInterchange{
		inspection:        inspection,
		reading:           reading,
		controlNumber:     first.Header.ControlNumber,
		senderQualifier:   first.Header.SenderQualifier,
		senderID:          first.Header.SenderID,
		receiverQualifier: first.Header.ReceiverQualifier,
		receiverID:        first.Header.ReceiverID,
		version:           first.Header.Version,
		usage:             first.Header.Usage,
		transactions:      make([]parsedTransaction, 0, reading.Summary.Transactions),
	}
	for interchangeIndex := range reading.Interchanges {
		for groupIndex := range reading.Interchanges[interchangeIndex].Groups {
			group := &reading.Interchanges[interchangeIndex].Groups[groupIndex]
			parsed := parsedGroup{group: group}
			if group.Header != nil {
				parsed.functionalGroupID = group.Header.FunctionalID
				parsed.controlNumber = group.Header.ControlNumber
				parsed.applicationSender = group.Header.Sender
				parsed.applicationReceiver = group.Header.Receiver
				parsed.version = group.Header.Version
			}
			interchange.groups = append(interchange.groups, parsed)
			for transactionIndex := range group.Transactions {
				transaction, buildErr := buildTransaction(
					inspection,
					&group.Transactions[transactionIndex],
					len(interchange.groups)-1,
					&parsed,
				)
				if buildErr != nil {
					return nil, buildErr, nil
				}
				interchange.transactions = append(interchange.transactions, *transaction)
			}
		}
	}
	if len(interchange.groups) > 0 {
		lead := &interchange.groups[0]
		interchange.functionalGroupID = lead.functionalGroupID
		interchange.groupControlNumber = lead.controlNumber
		interchange.applicationSender = lead.applicationSender
		interchange.applicationReceiver = lead.applicationReceiver
	}
	return interchange, nil, nil
}

func buildTransaction(
	inspection *edix12inspect.InspectX12Result,
	transaction *x12.Transaction,
	group int,
	parsed *parsedGroup,
) (*parsedTransaction, error) {
	if transaction.FirstSegment > transaction.LastSegment ||
		transaction.LastSegment >= len(inspection.Segments) {
		return nil, fmt.Errorf(
			"%w: transaction %d",
			errTransactionWithoutSegments,
			transaction.Index+1,
		)
	}
	segments := inspection.Segments[transaction.FirstSegment : transaction.LastSegment+1]
	raw := strings.Builder{}
	for index := range segments {
		raw.WriteString(segments[index].RawWithTerminator)
	}
	return &parsedTransaction{
		set:                edi.TransactionSet(transaction.TransactionSet),
		controlNumber:      transaction.ControlNumber,
		groupControlNumber: parsed.controlNumber,
		functionalGroupID:  parsed.functionalGroupID,
		segments:           segments,
		raw:                raw.String(),
		version:            stringutils.FirstNonEmpty(transaction.Convention, parsed.version),
		group:              group,
		status:             transaction.Status,
		findings:           transaction.Findings,
	}, nil
}

func findSegment(segments []edix12inspect.X12Segment, segmentID string) *edix12inspect.X12Segment {
	for index := range segments {
		if segments[index].SegmentID == segmentID {
			return &segments[index]
		}
	}
	return nil
}

func elementValue(segment *edix12inspect.X12Segment, position int) string {
	if segment == nil {
		return ""
	}
	for index := range segment.Elements {
		if segment.Elements[index].Position == position {
			return segment.Elements[index].Value
		}
	}
	return ""
}

func (t *parsedTransaction) documentPayload() edi.DocumentPayload {
	switch t.set {
	case edi.TransactionSet997, edi.TransactionSet999:
		entries := parseAcknowledgments(t)
		ack := edi.FunctionalAcknowledgmentPayload{}
		if len(entries) > 0 {
			first := entries[0]
			ack.OriginalFunctionalGroupID = first.originalFunctionalGroupID
			ack.OriginalGroupControlNumber = first.originalGroupControl
			ack.OriginalTransactionSet = transactionSetFromCode(first.originalTransactionSet)
			ack.OriginalTransactionControlNumber = first.originalControlNumber
			ack.GroupAcknowledgmentCode = first.groupAcknowledgmentCode
			ack.TransactionAcknowledgmentCode = first.acknowledgmentCode
			ack.AcceptedTransactionSetCount = first.acceptedCount
			ack.ReceivedTransactionSetCount = first.receivedCount
			ack.IncludedTransactionSetCount = first.includedCount
			ack.Diagnostics = first.diagnostics
		}
		if t.set == edi.TransactionSet999 {
			implementation := edi.ImplementationAckPayload(ack)
			return edi.DocumentPayload{
				TransactionSet:               edi.TransactionSet999,
				ImplementationAcknowledgment: &implementation,
			}
		}
		return edi.DocumentPayload{
			TransactionSet:           edi.TransactionSet997,
			FunctionalAcknowledgment: &ack,
		}
	}
	return edi.DocumentPayload{TransactionSet: t.set}
}

func parseAcknowledgments(t *parsedTransaction) []acknowledgmentEntry {
	ak1 := findSegment(t.segments, "AK1")
	ak9 := findSegment(t.segments, "AK9")
	base := acknowledgmentEntry{}
	if ak1 != nil {
		base.originalFunctionalGroupID = strings.TrimSpace(elementValue(ak1, 1))
		base.originalGroupControl = strings.TrimSpace(elementValue(ak1, 2))
	}
	if ak9 != nil {
		base.groupAcknowledgmentCode = strings.TrimSpace(elementValue(ak9, 1))
		base.includedCount = parseInt64(elementValue(ak9, 2))
		base.receivedCount = parseInt64(elementValue(ak9, 3))
		base.acceptedCount = parseInt64(elementValue(ak9, 4))
	}

	entries := make([]acknowledgmentEntry, 0, 2)
	var current *acknowledgmentEntry
	for index := range t.segments {
		segment := &t.segments[index]
		switch segment.SegmentID {
		case "AK2", "IK2":
			if current != nil {
				entries = append(entries, *current)
			}
			entry := base
			entry.originalTransactionSet = strings.TrimSpace(elementValue(segment, 1))
			entry.originalControlNumber = strings.TrimSpace(elementValue(segment, 2))
			current = &entry
		case "AK5", "IK5":
			if current != nil {
				current.acknowledgmentCode = strings.TrimSpace(elementValue(segment, 1))
			}
		case "AK3", "IK3":
			if current != nil {
				current.diagnostics = append(current.diagnostics, edi.AcknowledgmentDiagnostic{
					SegmentID:       strings.TrimSpace(elementValue(segment, 1)),
					SegmentPosition: parseInt64(elementValue(segment, 2)),
					ErrorCode:       strings.TrimSpace(elementValue(segment, 4)),
					Message:         "segment level acknowledgment error",
				})
			}
		case "AK4", "IK4":
			if current != nil {
				current.diagnostics = append(current.diagnostics, edi.AcknowledgmentDiagnostic{
					ElementPosition: parseInt64(elementValue(segment, 1)),
					ErrorCode:       strings.TrimSpace(elementValue(segment, 3)),
					Message:         "element level acknowledgment error",
				})
			}
		}
	}
	if current != nil {
		entries = append(entries, *current)
	}
	if len(entries) == 0 && (ak1 != nil || ak9 != nil) {
		entries = append(entries, base)
	}
	return entries
}

func transactionSetFromCode(code string) edi.TransactionSet {
	code = strings.TrimSpace(code)
	switch code {
	case "SM":
		return edi.TransactionSet204
	case "IM":
		return edi.TransactionSet210
	case "QM":
		return edi.TransactionSet214
	case "GF":
		return edi.TransactionSet990
	default:
		return edi.TransactionSet(code)
	}
}

func parseInt64(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}
	return parsed
}
