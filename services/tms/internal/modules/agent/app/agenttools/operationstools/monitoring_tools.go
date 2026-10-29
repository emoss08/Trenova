package operationstools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/detentionpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/outboundpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/app/customerupdateservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionnoticeservice"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentionreview"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentcommentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const (
	toolDetentionGetOccurrence = "detention_get_occurrence"
)

var driverMessagePriorities = agenttoolschema.Source(
	"notification.priority",
	[]notification.Priority{
		notification.PriorityLow,
		notification.PriorityMedium,
		notification.PriorityHigh,
		notification.PriorityCritical,
	},
)

var waiverReasons = agenttoolschema.Source("detention.waiverReason", []detention.WaiverReason{
	detention.WaiverReasonWeather,
	detention.WaiverReasonFacilityClosure,
	detention.WaiverReasonCarrierFault,
	detention.WaiverReasonEquipmentIssue,
	detention.WaiverReasonCustomerGoodwill,
	detention.WaiverReasonDataCorrection,
	detention.WaiverReasonForceMajeure,
	detention.WaiverReasonOther,
})

// driverNotifier is the driver portal's notification path. The wording
// comes from the organization's template for the kind; the tool supplies
// the title and the text, and the template decides how they are dressed.
type driverNotifier interface {
	Notify(ctx context.Context, req *drivernotificationservice.DriverNotification)
	Preview(
		ctx context.Context,
		req *drivernotificationservice.DriverNotification,
	) (*notificationports.DriverNotificationPreview, error)
}

const MaxDriverMessageChars = 500

const DispatchMessageEvent = "dash.dispatch_message"

type NotifyDriverTool struct {
	drivers driverNotifier
}

func NewNotifyDriverTool(drivers driverNotifier) agenttoolports.AgentTool {
	return &NotifyDriverTool{drivers: drivers}
}

func (t *NotifyDriverTool) Name() string { return "worker_send_message" }

func (t *NotifyDriverTool) SearchTerms() []string {
	return []string{"text the driver", "message the driver", "sms", "driver app"}
}

func (t *NotifyDriverTool) Description() string {
	return "Send a driver a message in the Dash app, such as a moved pickup, a new " +
		"delivery window, a weather warning on their route, or a request to call " +
		"dispatch. Passing on what the person told you needs no record checked or changed " +
		"first: send it as they said it, naming the load when you know it. Keep it " +
		"to what the driver needs to do; it goes to their phone. A driver without " +
		"portal access cannot be reached this way, and the message is not " +
		"delivered; say so if a reply matters. It is a message, not a record: put " +
		"anything the office needs later on the shipment with shipment_add_comment."
}

func (t *NotifyDriverTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workerId": agenttoolschema.RecordIDText(
				permission.ResourceWorker,
				"The driver's id, from the board, tracking or worker_list.",
			),
			"title": map[string]any{
				"type":        "string",
				"description": "A short subject line, such as \"Delivery moved to 3 PM\".",
			},
			"message": map[string]any{
				"type": "string",
				"description": fmt.Sprintf(
					"The message, at most %d characters, in plain words.", MaxDriverMessageChars,
				),
			},
			"priority": agenttoolschema.Enum(
				"How urgently the phone should show it. Default medium.",
				driverMessagePriorities,
			),
			toolparams.ParamShipmentID: agenttoolschema.RecordIDText(
				permission.ResourceShipment,
				"Optional: the shipment the message is about, so Dash can open "+
					"it; from dispatch_get_board, shipment_get_tracking or shipment_list.",
			),
		},
		"required":             []string{"workerId", "title", "message"},
		"additionalProperties": false,
	}
}

func (t *NotifyDriverTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceDriverMessage,
		Operation:     permission.OpCreate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressDriverVisible},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Sends a driver a message the model wrote to their phone.",
	}
}

func (t *NotifyDriverTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	request, err := t.request(&params)
	if err != nil {
		return err
	}

	t.drivers.Notify(ctx, request)

	return nil
}

// request is the message the preview renders and the write sends: the
// dispatch template, the model's title and text, and the shipment it opens.
func (t *NotifyDriverTool) request(
	params *agenttoolports.ToolExecuteParams,
) (*drivernotificationservice.DriverNotification, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return nil, err
	}

	workerID, err := toolparams.RequirePulid(params.Params, "workerId")
	if err != nil {
		return nil, err
	}
	title, err := toolparams.RequireString(params.Params, "title")
	if err != nil {
		return nil, err
	}
	message, err := toolparams.RequireString(params.Params, "message")
	if err != nil {
		return nil, err
	}
	if len(message) > MaxDriverMessageChars {
		return nil, fmt.Errorf(
			"parameter \"message\" is %d characters; keep it under %d so it reads on a phone",
			len(message), MaxDriverMessageChars,
		)
	}

	priority := notification.Priority(toolparams.OptionalString(params.Params, "priority"))
	switch priority {
	case notification.PriorityLow, notification.PriorityMedium,
		notification.PriorityHigh, notification.PriorityCritical:
	case "":
		priority = notification.PriorityMedium
	default:
		return nil, errors.New("parameter \"priority\" must be low, medium, high or critical")
	}

	request := &drivernotificationservice.DriverNotification{
		TenantInfo: toolparams.TenantFrom(*params),
		WorkerID:   workerID,
		EventType:  DispatchMessageEvent,
		Priority:   priority,
		Context: notificationtemplate.DriverNotificationContext{
			AlertTitle:   strings.TrimSpace(title),
			AlertMessage: strings.TrimSpace(message),
		},
	}
	if raw := toolparams.OptionalString(params.Params, toolparams.ParamShipmentID); raw != "" {
		shipmentID, parseErr := pulid.Parse(raw)
		if parseErr != nil {
			return nil, fmt.Errorf("parameter \"shipmentId\" is not a valid id: %w", parseErr)
		}
		request.RelatedEntities = map[string]any{toolparams.ParamShipmentID: shipmentID.String()}
		request.Link = "/loads/" + shipmentID.String()
	}

	return request, nil
}

// customerMailer is what customer_send_email needs: the shipment for its customer
// and PRO, the customer for its recipients, the template to dress the prose,
// the mailer to send it, and the comment thread to record that it went.
type CustomerMailer struct {
	Email     emailports.EmailService                        `json:"-"`
	Templates documenttemplateports.DocumentTemplateResolver `json:"-"`
	orgRepo   tenantports.OrganizationRepository
	inliner   documenttemplateports.AssetInliner
	Customers customerports.CustomerRepository            `json:"-"`
	Shipments shipmentports.ShipmentRepository            `json:"-"`
	Comments  shipmentcommentports.ShipmentCommentService `json:"-"`
	Senders   emailports.EmailSenderResolver              `json:"-"`
}

type emailCustomerParams struct {
	fx.In

	Email     emailports.EmailService
	Templates documenttemplateports.DocumentTemplateResolver
	OrgRepo   tenantports.OrganizationRepository
	Inliner   documenttemplateports.AssetInliner
	Customers customerports.CustomerRepository
	Shipments shipmentports.ShipmentRepository
	Comments  shipmentcommentports.ShipmentCommentService
	Senders   emailports.EmailSenderResolver `optional:"true"`
}

type EmailCustomerTool struct {
	Deps CustomerMailer `json:"-"`
}

func NewEmailCustomerTool(p emailCustomerParams) agenttoolports.AgentTool {
	return &EmailCustomerTool{Deps: CustomerMailer{
		Email:     p.Email,
		Templates: p.Templates,
		orgRepo:   p.OrgRepo,
		inliner:   p.Inliner,
		Customers: p.Customers,
		Shipments: p.Shipments,
		Comments:  p.Comments,
		Senders:   p.Senders,
	}}
}

const emailCustomerToolName = customerupdateservice.SourceAgentEmail

func (t *EmailCustomerTool) Name() string { return emailCustomerToolName }

func (t *EmailCustomerTool) Recipe() []string {
	return []string{
		toolparams.ToolShipmentGetTracking,
		"customer_get_update_preferences",
		emailCustomerToolName,
	}
}

func (t *EmailCustomerTool) Description() string {
	return "Email a shipment's customer a status update: a delay and the new expected " +
		"time, a delivery confirmation, a request to reschedule. The recipients are " +
		"the customer's own notice contacts on file, never addresses you supply, and " +
		"the message goes out in the organization's letterhead. Write the body as " +
		"plain prose with the facts the customer needs; never quote internal cost, " +
		"margin, or the driver's hours. The email is recorded on the shipment as a " +
		"customer update. Every send needs a person's approval."
}

func (t *EmailCustomerTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			toolparams.ParamShipmentID: agenttoolschema.RecordIDText(
				permission.ResourceShipment,
				"The shipment the update is about, from "+
					"shipment_list or this run's subject. Its customer is who receives it.",
			),
			"profileId": agenttoolschema.RecordIDText(
				permission.ResourceEmailProfile,
				"The email profile to send from, from inbox_list_email_profiles. Leave it out to "+
					"send from the profile the organization assigns to operations email, which "+
					"is right unless the person named another.",
			),
			"subject": map[string]any{"type": "string", "description": "The subject line."},
			"body": map[string]any{
				"type": "string",
				"description": "The update in plain prose. The template adds the greeting, the " +
					"shipment reference and the sign-off, so leave those out.",
			},
		},
		"required":             []string{toolparams.ParamShipmentID, "subject", "body"},
		"additionalProperties": false,
	}
}

func (t *EmailCustomerTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceCustomerCommunication,
		Operation:     permission.OpCreate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agenttooltrust.ToolEffectChange,
		Idempotent:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Emails the customer's contacts a message the model wrote.",
	}
}

func (t *EmailCustomerTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamShipmentID, permission.ResourceShipment)
}

func (t *EmailCustomerTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	composed, err := t.compose(ctx, &params)
	if err != nil {
		return err
	}
	if composed.AlreadyTold {
		return outboundpreview.ErrCustomerAlreadyTold
	}

	if _, err = t.Deps.Email.Send(ctx, composed.Send); err != nil {
		return err
	}

	return t.record(ctx, composed)
}

// record leaves the email on the shipment's thread so the desk sees what the
// customer was told. A failure to record never undoes a send that already
// happened; it is reported so the run shows it.
func (t *EmailCustomerTool) record(
	ctx context.Context,
	composed *outboundpreview.CustomerEmail,
) error {
	if t.Deps.Comments == nil {
		return nil
	}

	if _, err := t.Deps.Comments.CreateSystem(
		ctx,
		outboundpreview.CustomerUpdateComment(composed),
	); err != nil {
		return fmt.Errorf("the email was sent but could not be recorded on the shipment: %w", err)
	}

	return nil
}

type detentionNoticeSender interface {
	SendOccurrenceNotice(
		ctx context.Context,
		params detentionnoticeservice.SendOccurrenceNoticeParams,
	) (*detention.DetentionOccurrence, error)
	PreviewOccurrenceNotice(
		ctx context.Context,
		params *detentionnoticeservice.SendOccurrenceNoticeParams,
	) (*detentionnoticeservice.NoticePreview, error)
}

type detentionWaiver interface {
	Waive(
		ctx context.Context,
		params detentionreview.WaiveParams,
	) (*detention.DetentionOccurrence, error)
	PreviewWaive(
		ctx context.Context,
		params *detentionreview.WaiveParams,
	) (*detentionreview.OccurrenceChange, error)
}

type SendDetentionNoticeTool struct {
	detention detentionNoticeSender
}

func NewSendDetentionNoticeTool(detentions detentionNoticeSender) agenttoolports.AgentTool {
	return &SendDetentionNoticeTool{detention: detentions}
}

func ProvideSendDetentionNoticeTool(
	detentions *detentionnoticeservice.Service,
) agenttoolports.AgentTool {
	return NewSendDetentionNoticeTool(detentions)
}

func (t *SendDetentionNoticeTool) Name() string { return "detention_send_notice" }

func (t *SendDetentionNoticeTool) Prerequisites() []string {
	return []string{"detention_list_occurrences", toolDetentionGetOccurrence}
}

func (t *SendDetentionNoticeTool) Recipe() []string {
	return []string{toolDetentionGetOccurrence, "detention_send_notice"}
}

func (t *SendDetentionNoticeTool) Description() string {
	return "Send the customer the detention notice for an occurrence on the desk. The " +
		"notice is the organization's own template with the arrival, the free time " +
		"and the rate; nothing is composed here. Send it when detention_list_occurrences " +
		"shows the notice window open or overdue, because a charge with no notice " +
		"inside the policy's window is often not collectable. An occurrence whose " +
		"policy needs no notice is refused."
}

func (t *SendDetentionNoticeTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"occurrenceId": agenttoolschema.KindID(
				"The occurrence id from detention_list_occurrences.",
				recordkind.DetentionOccurrence,
			),
		},
		"required":             []string{"occurrenceId"},
		"additionalProperties": false,
	}
}

func (t *SendDetentionNoticeTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceCustomerCommunication,
		Operation:     permission.OpCreate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Sends the customer a detention notice that starts a charge.",
	}
}

func (t *SendDetentionNoticeTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.detention.SendOccurrenceNotice(ctx, request)

	return err
}

func (t *SendDetentionNoticeTool) request(
	params *agenttoolports.ToolExecuteParams,
) (detentionnoticeservice.SendOccurrenceNoticeParams, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return detentionnoticeservice.SendOccurrenceNoticeParams{}, err
	}

	occurrenceID, err := toolparams.RequirePulid(params.Params, "occurrenceId")
	if err != nil {
		return detentionnoticeservice.SendOccurrenceNoticeParams{}, err
	}

	return detentionnoticeservice.SendOccurrenceNoticeParams{
		OccurrenceID: occurrenceID,
		TenantInfo:   toolparams.TenantFrom(*params),
		UserID:       params.Actor.UserID,
	}, nil
}

type WaiveDetentionTool struct {
	detention detentionWaiver
}

func NewWaiveDetentionTool(detentions detentionWaiver) agenttoolports.AgentTool {
	return &WaiveDetentionTool{detention: detentions}
}

func ProvideWaiveDetentionTool(detentions *detentionreview.Service) agenttoolports.AgentTool {
	return NewWaiveDetentionTool(detentions)
}

func (t *WaiveDetentionTool) Name() string { return "detention_waive" }

func (t *WaiveDetentionTool) Recipe() []string {
	return []string{"detention_list_occurrences", toolDetentionGetOccurrence, "detention_waive"}
}

func (t *WaiveDetentionTool) Description() string {
	return "Waive a detention charge, with a coded reason and a note. This gives up " +
		"revenue, so propose it only when the cause is plainly the carrier's own " +
		"(a late truck, an equipment problem), the weather, or a data correction, " +
		"and say why in the note. A person decides."
}

func (t *WaiveDetentionTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"occurrenceId": agenttoolschema.KindID(
				"The occurrence id from detention_list_occurrences.",
				recordkind.DetentionOccurrence,
			),
			changepreview.FieldReason: agenttoolschema.Enum(
				"The coded reason for the waiver.",
				waiverReasons,
			),
			"note": map[string]any{
				"type":        "string",
				"description": "Why, in a sentence a person can read on the occurrence later.",
			},
		},
		"required":             []string{"occurrenceId", changepreview.FieldReason, "note"},
		"additionalProperties": false,
	}
}

func (t *WaiveDetentionTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceDetention,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierPropose,
		MaxTier:       agent.TierPropose,
		Egress:        []agent.EgressClass{agent.EgressMoney},
		Effect:        agenttooltrust.ToolEffectChange,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale: "Gives up detention revenue the organization would otherwise bill; only a " +
			"person approves it.",
	}
}

func (t *WaiveDetentionTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}
	if !params.ApprovedFromProposal() {
		return detentionpreview.ErrApprovalNeedsAPerson
	}

	request, err := t.request(&params)
	if err != nil {
		return err
	}

	_, err = t.detention.Waive(ctx, request)

	return err
}

func (t *WaiveDetentionTool) request(
	params *agenttoolports.ToolExecuteParams,
) (detentionreview.WaiveParams, error) {
	if err := toolguard.GuardExecute(t, *params); err != nil {
		return detentionreview.WaiveParams{}, err
	}

	occurrenceID, err := toolparams.RequirePulid(params.Params, "occurrenceId")
	if err != nil {
		return detentionreview.WaiveParams{}, err
	}
	rawReason, err := toolparams.RequireString(params.Params, changepreview.FieldReason)
	if err != nil {
		return detentionreview.WaiveParams{}, err
	}
	reason, err := detention.WaiverReasonFromString(rawReason)
	if err != nil {
		return detentionreview.WaiveParams{}, fmt.Errorf(
			"parameter \"reason\" is not a waiver reason: %w", err,
		)
	}
	note, err := toolparams.RequireString(params.Params, "note")
	if err != nil {
		return detentionreview.WaiveParams{}, err
	}

	return detentionreview.WaiveParams{
		OccurrenceID: occurrenceID,
		TenantInfo:   toolparams.TenantFrom(*params),
		Reason:       reason,
		Note:         strings.TrimSpace(note),
		UserID:       params.Actor.UserID,
	}, nil
}
