package ediinboundservice

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/app/ladingdoc"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/pkg/lading"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/x12"
)

var errNoReceivingVersion = errors.New("the partner has no active Lading version")

type readTransaction struct {
	result   *lading.TransactionResult
	unmapped ladingdoc.Unmapped
	flow     string
	clause   *lading.Clause
	lookups  []lading.LookupAnswer
}

// reading is what the partner's active version made of a file: every transaction a receive
// clause takes, read by its map and checked by its rules against the partner's guide.
type reading struct {
	programs *edilading.Programs
	read     map[string]*readTransaction
	skipped  map[edi.TransactionSet]string
}

func readKey(set edi.TransactionSet, control string) string {
	return string(set) + "/" + control
}

func (r *reading) transaction(transaction *parsedTransaction) *readTransaction {
	return r.read[readKey(transaction.set, transaction.controlNumber)]
}

func (s *Service) readFile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partner *edi.EDIPartner,
	interchange *parsedInterchange,
	raw string,
) (*reading, error) {
	programs, err := s.lading.Load(ctx, tenantInfo, partner.ID)
	if errors.Is(err, edilading.ErrNoProject) {
		return nil, errNoReceivingVersion
	}
	if err != nil {
		return nil, err
	}
	if programs.Active == nil {
		return nil, errNoReceivingVersion
	}
	r := &reading{
		programs: programs,
		read:     map[string]*readTransaction{},
		skipped:  map[edi.TransactionSet]string{},
	}
	for _, set := range receivedSets(interchange) {
		flow, clause := programs.Active.Manifest.Receiving(string(set))
		if clause == nil {
			r.skipped[set] = fmt.Sprintf(
				"version %d of the partner's Lading project receives no %s",
				programs.Active.Version.Number,
				set,
			)
			continue
		}
		resolved, runErr := s.lading.Resolve(ctx, &edilading.RunRequest{
			TenantInfo: tenantInfo,
			PartnerID:  partner.ID,
			Program:    programs.Active,
			Map:        clause.Map,
			Input:      lading.X12Input(raw),
			Rules:      true,
			Now:        time.Now(),
		})
		if runErr != nil {
			return nil, runErr
		}
		result := resolved.Result
		if result.Error != nil {
			r.skipped[set] = fmt.Sprintf(
				"map %s could not read the file: %s",
				clause.Map,
				result.Error,
			)
			continue
		}
		unmapped := ladingdoc.UnmappedOf(result.Unmapped)
		for index := range result.Transactions {
			transaction := &result.Transactions[index]
			r.read[readKey(set, transaction.Control)] = &readTransaction{
				result:   transaction,
				unmapped: unmapped,
				flow:     flow.Name,
				clause:   clause,
				lookups:  resolved.Lookups,
			}
		}
	}
	return r, nil
}

func receivedSets(interchange *parsedInterchange) []edi.TransactionSet {
	sets := make([]edi.TransactionSet, 0, 2)
	for index := range interchange.transactions {
		set := interchange.transactions[index].set
		if set.IsAcknowledgment() || slices.Contains(sets, set) {
			continue
		}
		sets = append(sets, set)
	}
	return sets
}

// apply puts what the partner's guide found on a transaction in place of the standard reading,
// so the message, its diagnostics and the acknowledgment all report the partner's rules.
func (r *readTransaction) apply(interchange *parsedInterchange, transaction *parsedTransaction) {
	transaction.findings = r.result.Findings
	transaction.status = x12.TransactionAccepted
	if r.result.Rejected() {
		transaction.status = x12.TransactionRejected
	}
	group := interchange.groups[transaction.group].group
	if group == nil {
		return
	}
	for index := range group.Transactions {
		read := &group.Transactions[index]
		if read.ControlNumber == transaction.controlNumber {
			read.Findings = transaction.findings
			read.Status = transaction.status
		}
	}
}

func (r *readTransaction) payload(set edi.TransactionSet) (edi.DocumentPayload, error) {
	payload := edi.DocumentPayload{TransactionSet: set}
	document := r.result.Document
	if len(document) == 0 {
		return payload, nil
	}
	switch set {
	case edi.TransactionSet204:
		tender, _, err := ladingdoc.ToLoadTender(document, r.unmapped)
		if err != nil {
			return payload, err
		}
		payload.LoadTender = tender
		payload.PurposeCode = tender.PurposeCode
	case edi.TransactionSet990:
		answer, err := ladingdoc.ToTenderAnswer(document)
		if err != nil {
			return payload, err
		}
		payload.TenderResponse = &edi.TenderResponsePayload{
			BOL:             answer.Key,
			ResponseCode:    answer.ReservationCode,
			RejectionReason: answer.Reason,
		}
	case edi.TransactionSet214:
		status, err := ladingdoc.ToShipmentStatus(document)
		if err != nil {
			return payload, err
		}
		payload.ShipmentStatus = status
	case edi.TransactionSet210:
		invoice, err := ladingdoc.ToFreightInvoice(document)
		if err != nil {
			return payload, err
		}
		payload.FreightInvoice = invoice
	case edi.TransactionSet997, edi.TransactionSet999:
	}
	return payload, nil
}

func outcomeInputs(result *lading.TransactionResult) []lading.OutcomeInput {
	inputs := make([]lading.OutcomeInput, 0, len(result.Outcomes)+1)
	rejected := false
	for _, outcome := range result.Outcomes {
		rejected = rejected || outcome.Kind == lading.OutcomeReject
		inputs = append(inputs, lading.OutcomeInput{
			Kind:    outcome.Kind,
			Message: outcome.Message,
			Code:    outcome.Code,
		})
	}
	if result.Status == lading.TransactionRejected && !rejected {
		inputs = append(inputs, lading.OutcomeInput{
			Kind:    lading.OutcomeReject,
			Message: rejectionMessage(result.Findings),
		})
	}
	return inputs
}

func rejectionMessage(findings []x12.Finding) string {
	for index := range findings {
		if findings[index].IsError() {
			return findings[index].Message.String()
		}
	}
	return "The transaction breaks the partner's guide"
}

type received struct {
	file        *edi.EDIInboundFile
	partner     *edi.EDIPartner
	message     *edi.EDIMessage
	transaction *parsedTransaction
	read        *readTransaction
	programs    *edilading.Programs
}

// receive gives a read transaction to the partner's flow and carries out what the flow asks.
func (s *Service) receive(ctx context.Context, in *received) transactionOutcome {
	outcome := transactionOutcome{message: in.message}
	tenantInfo := inboundFileTenant(in.file)

	key := ""
	if in.transaction.set == edi.TransactionSet204 && in.message.PayloadSnapshot.LoadTender != nil {
		key, _ = in.message.PayloadSnapshot.LoadTender.RatingDetail["externalShipmentId"].(string)
	}
	dispatched, err := s.flows.Dispatch(ctx, &edilading.DispatchRequest{
		TenantInfo: tenantInfo,
		PartnerID:  in.partner.ID,
		Program:    in.programs.Active,
		Flow:       in.read.flow,
		FallbackKey: firstNonEmpty(
			key,
			string(in.transaction.set)+"-"+in.transaction.controlNumber,
		),
		Event: lading.FlowEvent{
			Kind:     lading.EventReceived,
			Set:      string(in.transaction.set),
			Document: in.read.result.Document,
			Outcomes: outcomeInputs(in.read.result),
		},
		MessageID: in.message.ID,
	})
	if err != nil {
		outcome.err = fmt.Errorf("the partner's flow %s: %w", in.read.flow, err)
		return outcome
	}
	if _, err = s.messageRepo.LinkMessageFlow(ctx, &edimessageports.LinkEDIMessageFlowRequest{
		ID:             in.message.ID,
		TenantInfo:     tenantInfo,
		FlowInstanceID: dispatched.Instance.ID,
	}); err != nil {
		outcome.err = err
		return outcome
	}

	transfer := dispatched.Instance.TransferID
	for index := range dispatched.Actions {
		action := &dispatched.Actions[index]
		created, warnings, actErr := s.act(ctx, in, dispatched, action)
		outcome.warnings = append(outcome.warnings, warnings...)
		if actErr != nil {
			outcome.err = actErr
			return outcome
		}
		if created != nil {
			transfer = created.ID
			outcome.tendered = true
		}
	}
	if transfer.IsNotNil() && transfer != dispatched.Instance.TransferID {
		if dispatched.Instance, err = s.flows.Attach(
			ctx,
			dispatched.Instance,
			transfer,
			"",
		); err != nil {
			outcome.err = err
			return outcome
		}
	}
	if _, err = s.ediService.ExecuteFlowActions(
		ctx,
		tenantInfo,
		dispatched,
		&ediservice.FlowActionContext{TransferID: transfer},
	); err != nil {
		outcome.warnings = append(outcome.warnings, fmt.Sprintf(
			"the partner's flow %s could not finish what it asked: %s",
			in.read.flow,
			err.Error(),
		))
	}
	return outcome
}

func (s *Service) act(
	ctx context.Context,
	in *received,
	dispatched *edilading.Dispatched,
	action *lading.FlowAction,
) (*edi.EDITransfer, []string, error) {
	switch action.Kind {
	case lading.ActionCreate, lading.ActionUpdate:
		return s.routeLoadTender(ctx, in.file, in.partner, in.message, in.transaction)
	case lading.ActionCancel:
		key := ""
		if tender := in.message.PayloadSnapshot.LoadTender; tender != nil {
			key, _ = tender.RatingDetail["externalShipmentId"].(string)
		}
		warnings, err := s.ediService.CancelInboundTender(
			ctx,
			&ediservice.CancelInboundTenderRequest{
				TenantInfo: inboundFileTenant(in.file),
				Partner:    in.partner,
				Key:        key,
				ShipmentID: dispatched.Instance.ShipmentID,
				Control:    in.transaction.controlNumber,
			},
		)
		return nil, warnings, err
	case lading.ActionApply:
		warnings, err := s.applyReceived(ctx, in)
		return nil, warnings, err
	case lading.ActionIgnore:
		return nil, []string{fmt.Sprintf(
			"transaction %s/%s recorded without processing: the partner's flow %s ignores it",
			in.transaction.set,
			in.transaction.controlNumber,
			in.read.flow,
		)}, nil
	}
	return nil, nil, nil
}

func (s *Service) applyReceived(ctx context.Context, in *received) ([]string, error) {
	payload := in.message.PayloadSnapshot
	switch in.transaction.set {
	case edi.TransactionSet990:
		if payload.TenderResponse == nil {
			return nil, nil
		}
		answer, err := ladingdoc.ToTenderAnswer(in.read.result.Document)
		if err != nil {
			return nil, err
		}
		return s.routeTenderResponse(
			ctx,
			in.file,
			in.partner,
			in.transaction,
			&tenderResponseDetails{
				scac:            answer.SCAC,
				shipmentRef:     answer.Key,
				reservationCode: answer.ReservationCode,
				remarks:         answer.Reason,
				references:      answer.References,
			},
		)
	case edi.TransactionSet214:
		if payload.ShipmentStatus == nil {
			return nil, nil
		}
		return s.routeShipmentStatus(
			ctx,
			in.file,
			in.partner,
			in.transaction,
			statusDetails(in.transaction, payload.ShipmentStatus),
		)
	case edi.TransactionSet210:
		return s.routeFreightInvoice(ctx, in.file, in.partner, in.message, in.transaction)
	case edi.TransactionSet204, edi.TransactionSet997, edi.TransactionSet999:
	}
	return []string{fmt.Sprintf(
		"transaction %s/%s recorded without processing: Trenova applies no %s",
		in.transaction.set,
		in.transaction.controlNumber,
		in.transaction.set,
	)}, nil
}

func statusDetails(
	transaction *parsedTransaction,
	status *edi.ShipmentStatusPayload,
) *shipmentStatusDetails {
	details := &shipmentStatusDetails{
		controlNumber: transaction.controlNumber,
		referenceID:   status.ProNumber,
		shipmentRef:   firstNonEmpty(status.References["shipmentId"], status.BOL),
		statusCode:    status.StatusCode,
		reasonCode:    status.StatusReasonCode,
		eventAt:       status.EventDate,
		events:        status.Events,
	}
	for _, value := range status.References {
		if value != "" && !slices.Contains(details.references, value) {
			details.references = append(details.references, value)
		}
	}
	slices.Sort(details.references)
	return details
}

// shadow compares the shadow version with the active one on each set of a file, once, and
// records the result against the set's first message.
func (s *Service) shadow(
	ctx context.Context,
	file *edi.EDIInboundFile,
	r *reading,
	first map[edi.TransactionSet]*edi.EDIMessage,
) {
	if s.replayer == nil || r.programs.Shadow == nil {
		return
	}
	tenantInfo := inboundFileTenant(file)
	for set, message := range first {
		_, clause := r.programs.Active.Manifest.Receiving(string(set))
		if clause == nil {
			continue
		}
		var lookups []lading.LookupAnswer
		for _, read := range r.read {
			if read.clause == clause {
				lookups = read.lookups
				break
			}
		}
		s.replayer.Shadow(ctx, &edilading.ShadowRunRequest{
			TenantInfo: tenantInfo,
			Programs:   r.programs,
			Map:        clause.Map,
			MessageID:  message.ID,
			Set:        set,
			Input:      lading.X12Input(file.RawContent),
			Lookups:    lookups,
			Now:        time.Now(),
		})
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
