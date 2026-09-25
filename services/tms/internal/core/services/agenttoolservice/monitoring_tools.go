package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/internal/core/services/drivernotificationservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/fx"
)

// serviceFailureDecider is the slice of the service failure service the
// monitoring writes need. Detection runs the same evaluator the stop actuals
// run, so an agent asking for it cannot invent a failure; resolving one goes
// through the same lifecycle a person's click does.
type serviceFailureDecider interface {
	EvaluateShipment(
		ctx context.Context,
		req *serviceports.EvaluateShipmentServiceFailuresRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ServiceFailureEvaluationResult, error)
	GetByID(
		ctx context.Context,
		req *repositories.GetServiceFailureByIDRequest,
	) (*servicefailure.ServiceFailure, error)
	Resolve(
		ctx context.Context,
		req *serviceports.ServiceFailureLifecycleRequest,
		actor *serviceports.RequestActor,
	) (*servicefailure.ServiceFailure, error)
}

type evaluateServiceFailuresTool struct {
	failures serviceFailureDecider
}

func newEvaluateServiceFailuresTool(failures serviceFailureDecider) serviceports.AgentTool {
	return &evaluateServiceFailuresTool{failures: failures}
}

func (t *evaluateServiceFailuresTool) Name() string { return "evaluate_service_failures" }

func (t *evaluateServiceFailuresTool) Description() string {
	return "Run the service failure check on one shipment now, opening a failure for a " +
		"late or missed stop that has none. Every stop is compared against its window " +
		"and the grace period. Use it when get_shipment_tracking shows a stop " +
		"late or overdue and list_service_failures shows nothing for it, so the " +
		"failure is on record before anyone is told. It records what the stop " +
		"actuals prove and nothing else; it cannot create a failure from a guess."
}

func (t *evaluateServiceFailuresTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": map[string]any{
				"type": "string",
				"description": "The shipment to check, from get_shipment_tracking, " +
					"list_shipments or this run's subject.",
			},
			"force": map[string]any{
				"type": "boolean",
				"description": "Re-check stops that were already evaluated. Default false; " +
					"set it only when an actual arrival was corrected.",
			},
		},
		"required":             []string{"shipmentId"},
		"additionalProperties": false,
	}
}

func (t *evaluateServiceFailuresTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceServiceFailure,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Opens service failures from stop actuals inside Trenova; an open failure " +
			"sends nothing.",
	}
}

func (t *evaluateServiceFailuresTool) Target(
	params map[string]any,
) (serviceports.ToolTarget, bool) {
	return targetOf(params, "shipmentId", permission.ResourceShipment)
}

func (t *evaluateServiceFailuresTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return err
	}

	_, err = t.failures.EvaluateShipment(ctx, &serviceports.EvaluateShipmentServiceFailuresRequest{
		TenantInfo: tenantFrom(params),
		ShipmentID: shipmentID,
		Force:      optionalBool(params.Params, "force"),
	}, params.Actor)

	return err
}

type resolveServiceFailureTool struct {
	failures serviceFailureDecider
}

func newResolveServiceFailureTool(failures serviceFailureDecider) serviceports.AgentTool {
	return &resolveServiceFailureTool{failures: failures}
}

func (t *resolveServiceFailureTool) Name() string { return "resolve_service_failure" }

func (t *resolveServiceFailureTool) Description() string {
	return "Close an open service failure with the reason it happened and a note on " +
		"what was done. The reason code comes from list_service_failure_reason_codes " +
		"and must match the cause; a failure that already carries one keeps it " +
		"unless you send another. Do not resolve a failure nobody has looked into: " +
		"the note should say what was learned, not that the stop was late."
}

func (t *resolveServiceFailureTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"serviceFailureId": map[string]any{
				"type":        "string",
				"description": "The failure's id, from list_service_failures.",
			},
			"reasonCodeId": map[string]any{
				"type": "string",
				"description": "The reason code, from list_service_failure_reason_codes. " +
					"Required when the failure has none; otherwise replaces it.",
			},
			"notes": map[string]any{
				"type":        "string",
				"description": "What was found and done. Required.",
			},
		},
		"required":             []string{"serviceFailureId", "notes"},
		"additionalProperties": false,
	}
}

func (t *resolveServiceFailureTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceServiceFailure,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Resolving a failure generates an EDI 214 to the customer's trading " +
			"partner carrying the reason chosen.",
	}
}

func (t *resolveServiceFailureTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "serviceFailureId", permission.ResourceServiceFailure)
}

func (t *resolveServiceFailureTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	failureID, err := requirePulid(params.Params, "serviceFailureId")
	if err != nil {
		return err
	}
	notes, err := requireString(params.Params, "notes")
	if err != nil {
		return err
	}

	tenant := tenantFrom(params)
	existing, err := t.failures.GetByID(ctx, &repositories.GetServiceFailureByIDRequest{
		ID:         failureID,
		TenantInfo: tenant,
	})
	if err != nil {
		return err
	}
	if existing.IsTerminal() {
		return fmt.Errorf(
			"service failure %s is already %s and cannot be resolved again",
			existing.Number, existing.Status,
		)
	}

	request := &serviceports.ServiceFailureLifecycleRequest{
		TenantInfo: tenant,
		ID:         existing.ID,
		ShipmentID: existing.ShipmentID,
		Notes:      strings.TrimSpace(notes),
		Version:    existing.Version,
	}
	if raw := optionalString(params.Params, "reasonCodeId"); raw != "" {
		reasonID, parseErr := pulid.Parse(raw)
		if parseErr != nil {
			return fmt.Errorf("parameter \"reasonCodeId\" is not a valid id: %w", parseErr)
		}
		request.ReasonCodeID = reasonID
	} else if existing.ReasonCodeID == nil || existing.ReasonCodeID.IsNil() {
		return errors.New(
			"this failure has no reason code yet; pick one from " +
				"list_service_failure_reason_codes and send it as reasonCodeId",
		)
	}

	_, err = t.failures.Resolve(ctx, request, params.Actor)

	return err
}

// driverNotifier is the driver portal's notification path. The wording
// comes from the organization's template for the kind; the tool supplies
// the title and the text, and the template decides how they are dressed.
type driverNotifier interface {
	Notify(ctx context.Context, req *drivernotificationservice.DriverNotification)
	Preview(
		ctx context.Context,
		req *drivernotificationservice.DriverNotification,
	) (*serviceports.DriverNotificationPreview, error)
}

const (
	maxDriverMessageChars = 500
	dispatchMessageEvent  = "dash.dispatch_message"
)

type notifyDriverTool struct {
	drivers driverNotifier
}

func newNotifyDriverTool(drivers driverNotifier) serviceports.AgentTool {
	return &notifyDriverTool{drivers: drivers}
}

func (t *notifyDriverTool) Name() string { return "notify_driver" }

func (t *notifyDriverTool) Description() string {
	return "Send a driver a message in the Dash app, such as a changed appointment, " +
		"a weather warning on their route, or a request to call dispatch. Keep it " +
		"to what the driver needs to do; it goes to their phone. A driver without " +
		"portal access cannot be reached this way, and the message is not " +
		"delivered; say so if a reply matters. It is a message, not a record: put " +
		"anything the office needs later on the shipment with add_shipment_comment."
}

func (t *notifyDriverTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workerId": map[string]any{
				"type":        "string",
				"description": "The driver's id, from the board, tracking or search_worker.",
			},
			"title": map[string]any{
				"type":        "string",
				"description": "A short subject line, such as \"Delivery moved to 3 PM\".",
			},
			"message": map[string]any{
				"type": "string",
				"description": fmt.Sprintf(
					"The message, at most %d characters, in plain words.", maxDriverMessageChars,
				),
			},
			"priority": map[string]any{
				"type":        "string",
				"enum":        []string{"low", "medium", "high", "critical"},
				"description": "How urgently the phone should show it. Default medium.",
			},
			"shipmentId": map[string]any{
				"type": "string",
				"description": "Optional: the shipment the message is about, so Dash can open " +
					"it; from get_dispatch_board, get_shipment_tracking or search_shipments.",
			},
		},
		"required":             []string{"workerId", "title", "message"},
		"additionalProperties": false,
	}
}

func (t *notifyDriverTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceDriverMessage,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressDriverVisible},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Sends a driver a message the model wrote to their phone.",
	}
}

func (t *notifyDriverTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	request, err := t.request(params)
	if err != nil {
		return err
	}

	t.drivers.Notify(ctx, request)

	return nil
}

// request is the message the preview renders and the write sends: the
// dispatch template, the model's title and text, and the shipment it opens.
func (t *notifyDriverTool) request(
	params serviceports.ToolExecuteParams,
) (*drivernotificationservice.DriverNotification, error) {
	if err := guardExecute(t, params); err != nil {
		return nil, err
	}

	workerID, err := requirePulid(params.Params, "workerId")
	if err != nil {
		return nil, err
	}
	title, err := requireString(params.Params, "title")
	if err != nil {
		return nil, err
	}
	message, err := requireString(params.Params, "message")
	if err != nil {
		return nil, err
	}
	if len(message) > maxDriverMessageChars {
		return nil, fmt.Errorf(
			"parameter \"message\" is %d characters; keep it under %d so it reads on a phone",
			len(message), maxDriverMessageChars,
		)
	}

	priority := notification.Priority(optionalString(params.Params, "priority"))
	switch priority {
	case notification.PriorityLow, notification.PriorityMedium,
		notification.PriorityHigh, notification.PriorityCritical:
	case "":
		priority = notification.PriorityMedium
	default:
		return nil, errors.New("parameter \"priority\" must be low, medium, high or critical")
	}

	request := &drivernotificationservice.DriverNotification{
		TenantInfo: tenantFrom(params),
		WorkerID:   workerID,
		EventType:  dispatchMessageEvent,
		Priority:   priority,
		Context: documenttemplate.DriverNotificationContext{
			AlertTitle:   strings.TrimSpace(title),
			AlertMessage: strings.TrimSpace(message),
		},
	}
	if raw := optionalString(params.Params, "shipmentId"); raw != "" {
		shipmentID, parseErr := pulid.Parse(raw)
		if parseErr != nil {
			return nil, fmt.Errorf("parameter \"shipmentId\" is not a valid id: %w", parseErr)
		}
		request.RelatedEntities = map[string]any{"shipmentId": shipmentID.String()}
		request.Link = "/loads/" + shipmentID.String()
	}

	return request, nil
}

// customerMailer is what email_customer needs: the shipment for its customer
// and PRO, the customer for its recipients, the template to dress the prose,
// the mailer to send it, and the comment thread to record that it went.
type customerMailer struct {
	email     serviceports.EmailService
	templates serviceports.DocumentTemplateResolver
	orgRepo   repositories.OrganizationRepository
	inliner   serviceports.AssetInliner
	customers repositories.CustomerRepository
	shipments repositories.ShipmentRepository
	comments  serviceports.ShipmentCommentService
	senders   serviceports.EmailSenderResolver
}

type emailCustomerParams struct {
	fx.In

	Email     serviceports.EmailService
	Templates serviceports.DocumentTemplateResolver
	OrgRepo   repositories.OrganizationRepository
	Inliner   serviceports.AssetInliner
	Customers repositories.CustomerRepository
	Shipments repositories.ShipmentRepository
	Comments  serviceports.ShipmentCommentService
	Senders   serviceports.EmailSenderResolver `optional:"true"`
}

type emailCustomerTool struct {
	deps customerMailer
}

func newEmailCustomerTool(p emailCustomerParams) serviceports.AgentTool {
	return &emailCustomerTool{deps: customerMailer{
		email:     p.Email,
		templates: p.Templates,
		orgRepo:   p.OrgRepo,
		inliner:   p.Inliner,
		customers: p.Customers,
		shipments: p.Shipments,
		comments:  p.Comments,
		senders:   p.Senders,
	}}
}

func (t *emailCustomerTool) Name() string { return "email_customer" }

func (t *emailCustomerTool) Description() string {
	return "Email a shipment's customer a status update: a delay and the new expected " +
		"time, a delivery confirmation, a request to reschedule. The recipients are " +
		"the customer's own notice contacts on file, never addresses you supply, and " +
		"the message goes out in the organization's letterhead. Write the body as " +
		"plain prose with the facts the customer needs; never quote internal cost, " +
		"margin, or the driver's hours. The email is recorded on the shipment as a " +
		"customer update. Every send needs a person's approval."
}

func (t *emailCustomerTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": map[string]any{
				"type": "string",
				"description": "The shipment the update is about, from search_shipments, " +
					"list_shipments or this run's subject. Its customer is who receives it.",
			},
			"profileId": map[string]any{
				"type": "string",
				"description": "The email profile to send from; list_email_profiles names them. " +
					"With one profile there is nothing to choose.",
			},
			"subject": map[string]any{"type": "string", "description": "The subject line."},
			"body": map[string]any{
				"type": "string",
				"description": "The update in plain prose. The template adds the greeting, the " +
					"shipment reference and the sign-off, so leave those out.",
			},
		},
		"required":             []string{"shipmentId", "profileId", "subject", "body"},
		"additionalProperties": false,
	}
}

func (t *emailCustomerTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceCustomerCommunication,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		Idempotent:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Emails the customer's contacts a message the model wrote.",
	}
}

func (t *emailCustomerTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "shipmentId", permission.ResourceShipment)
}

func (t *emailCustomerTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	composed, err := t.compose(ctx, params)
	if err != nil {
		return err
	}
	if composed.alreadyTold {
		return ErrCustomerAlreadyTold
	}

	if _, err = t.deps.email.Send(ctx, composed.send); err != nil {
		return err
	}

	return t.record(ctx, composed)
}

// recipients are the customer's notice contacts, read from the record. The
// model names the shipment; the customer's own profile says who is written to.
func (t *emailCustomerTool) recipients(
	ctx context.Context,
	sp *shipment.Shipment,
	tenant pagination.TenantInfo,
) ([]string, string, error) {
	if sp.CustomerID.IsNil() {
		return nil, "", errors.New("the shipment has no customer to write to")
	}

	entity, err := t.deps.customers.GetByID(ctx, repositories.GetCustomerByIDRequest{
		ID:                    sp.CustomerID,
		TenantInfo:            tenant,
		CustomerFilterOptions: repositories.CustomerFilterOptions{IncludeEmailProfile: true},
	})
	if err != nil {
		return nil, "", err
	}

	var recipients []string
	if entity.EmailProfile != nil {
		recipients = stringutils.SplitEmailList(entity.EmailProfile.ToRecipients)
	}
	if len(recipients) == 0 {
		return nil, "", fmt.Errorf(
			"%s has no notice recipients on file; a person must add an email profile "+
				"to the customer before anything can be sent",
			entity.Name,
		)
	}

	return recipients, entity.Name, nil
}

// record leaves the email on the shipment's thread so the desk sees what the
// customer was told. A failure to record never undoes a send that already
// happened; it is reported so the run shows it.
func (t *emailCustomerTool) record(ctx context.Context, composed *customerEmail) error {
	if t.deps.comments == nil {
		return nil
	}

	if _, err := t.deps.comments.CreateSystem(ctx, customerUpdateComment(composed)); err != nil {
		return fmt.Errorf("the email was sent but could not be recorded on the shipment: %w", err)
	}

	return nil
}

// detentionActor is the slice of the detention service the tools act
// through: send the customer notice, or waive the charge.
type detentionActor interface {
	SendOccurrenceNotice(
		ctx context.Context,
		params detentionservice.SendOccurrenceNoticeParams,
	) (*detention.DetentionOccurrence, error)
	Waive(
		ctx context.Context,
		params detentionservice.WaiveParams,
	) (*detention.DetentionOccurrence, error)
	PreviewOccurrenceNotice(
		ctx context.Context,
		params detentionservice.SendOccurrenceNoticeParams,
	) (*detentionservice.NoticePreview, error)
	PreviewWaive(
		ctx context.Context,
		params detentionservice.WaiveParams,
	) (*detentionservice.OccurrenceChange, error)
}

type sendDetentionNoticeTool struct {
	detention detentionActor
}

func newSendDetentionNoticeTool(detention detentionActor) serviceports.AgentTool {
	return &sendDetentionNoticeTool{detention: detention}
}

func (t *sendDetentionNoticeTool) Name() string { return "send_detention_notice" }

func (t *sendDetentionNoticeTool) Description() string {
	return "Send the customer the detention notice for an occurrence on the desk. The " +
		"notice is the organization's own template with the arrival, the free time " +
		"and the rate; nothing is composed here. Send it when list_detention_desk " +
		"shows the notice window open or overdue, because a charge with no notice " +
		"inside the policy's window is often not collectable. An occurrence whose " +
		"policy needs no notice is refused."
}

func (t *sendDetentionNoticeTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"occurrenceId": map[string]any{
				"type":        "string",
				"description": "The occurrence id from list_detention_desk.",
			},
		},
		"required":             []string{"occurrenceId"},
		"additionalProperties": false,
	}
}

func (t *sendDetentionNoticeTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceCustomerCommunication,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale:     "Sends the customer a detention notice that starts a charge.",
	}
}

func (t *sendDetentionNoticeTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	request, err := t.request(params)
	if err != nil {
		return err
	}

	_, err = t.detention.SendOccurrenceNotice(ctx, request)

	return err
}

func (t *sendDetentionNoticeTool) request(
	params serviceports.ToolExecuteParams,
) (detentionservice.SendOccurrenceNoticeParams, error) {
	if err := guardExecute(t, params); err != nil {
		return detentionservice.SendOccurrenceNoticeParams{}, err
	}

	occurrenceID, err := requirePulid(params.Params, "occurrenceId")
	if err != nil {
		return detentionservice.SendOccurrenceNoticeParams{}, err
	}

	return detentionservice.SendOccurrenceNoticeParams{
		OccurrenceID: occurrenceID,
		TenantInfo:   tenantFrom(params),
		UserID:       params.Actor.UserID,
	}, nil
}

type waiveDetentionTool struct {
	detention detentionActor
}

func newWaiveDetentionTool(detention detentionActor) serviceports.AgentTool {
	return &waiveDetentionTool{detention: detention}
}

func (t *waiveDetentionTool) Name() string { return "waive_detention" }

func (t *waiveDetentionTool) Description() string {
	return "Waive a detention charge, with a coded reason and a note. This gives up " +
		"revenue, so propose it only when the cause is plainly the carrier's own " +
		"(a late truck, an equipment problem), the weather, or a data correction, " +
		"and say why in the note. A person decides."
}

func (t *waiveDetentionTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"occurrenceId": map[string]any{
				"type":        "string",
				"description": "The occurrence id from list_detention_desk.",
			},
			"reason": map[string]any{
				"type": "string",
				"enum": []string{
					"Weather", "FacilityClosure", "CarrierFault", "EquipmentIssue",
					"CustomerGoodwill", "DataCorrection", "ForceMajeure", "Other",
				},
				"description": "The coded reason for the waiver.",
			},
			"note": map[string]any{
				"type":        "string",
				"description": "Why, in a sentence a person can read on the occurrence later.",
			},
		},
		"required":             []string{"occurrenceId", "reason", "note"},
		"additionalProperties": false,
	}
}

func (t *waiveDetentionTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceDetentionPolicy,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agent.ToolEffectChange,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "Gives up detention revenue the organization would otherwise bill; only a " +
			"person approves it.",
	}
}

func (t *waiveDetentionTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return ErrApprovalNeedsAPerson
	}

	request, err := t.request(params)
	if err != nil {
		return err
	}

	_, err = t.detention.Waive(ctx, request)

	return err
}

func (t *waiveDetentionTool) request(
	params serviceports.ToolExecuteParams,
) (detentionservice.WaiveParams, error) {
	if err := guardExecute(t, params); err != nil {
		return detentionservice.WaiveParams{}, err
	}

	occurrenceID, err := requirePulid(params.Params, "occurrenceId")
	if err != nil {
		return detentionservice.WaiveParams{}, err
	}
	rawReason, err := requireString(params.Params, "reason")
	if err != nil {
		return detentionservice.WaiveParams{}, err
	}
	reason, err := detention.WaiverReasonFromString(rawReason)
	if err != nil {
		return detentionservice.WaiveParams{}, fmt.Errorf(
			"parameter \"reason\" is not a waiver reason: %w", err,
		)
	}
	note, err := requireString(params.Params, "note")
	if err != nil {
		return detentionservice.WaiveParams{}, err
	}

	return detentionservice.WaiveParams{
		OccurrenceID: occurrenceID,
		TenantInfo:   tenantFrom(params),
		Reason:       reason,
		Note:         strings.TrimSpace(note),
		UserID:       params.Actor.UserID,
	}, nil
}
