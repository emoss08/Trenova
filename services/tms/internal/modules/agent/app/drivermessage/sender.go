package drivermessage

import (
	"context"
	"fmt"
	"strings"

	"go.temporal.io/sdk/client"

	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/jobs/smsjobs/smscontract"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	// DispatchEvent is the notification kind a message from the office to a
	// driver goes out as.
	DispatchEvent = "dash.dispatch_message"
	// MaxChars is the longest message that reads on a phone.
	MaxChars = 500
)

// Channel is how a message reaches a driver.
type Channel string

const (
	// ChannelDash is the driver's portal, for a driver with a login.
	ChannelDash = Channel("Dash")
	// ChannelText is a text to the number on file, for a driver with no
	// login, when the organization's plan includes text messages.
	ChannelText = Channel("Text")
	// ChannelNone is a driver nothing reaches.
	ChannelNone = Channel("")
)

// Notifier is the driver portal's notification path: the wording comes from
// the organization's template for the kind, and Preview renders what Notify
// would send without sending it.
type Notifier interface {
	Notify(ctx context.Context, req *drivernotificationservice.DriverNotification)
	Preview(
		ctx context.Context,
		req *drivernotificationservice.DriverNotification,
	) (*notificationports.DriverNotificationPreview, error)
}

// Sender sends a driver a message from the office where they can be
// reached: in Dash when they have a login, by text to the number on file
// when they do not and the plan includes text messages. It is the one owner
// of that choice, for an agent's tool and for a person's own request, and
// the number is read here and handed to nobody.
type Sender struct {
	drivers   Notifier
	plans     platformplanports.PlanService
	workflows kernelports.WorkflowStarter
}

func New(
	drivers Notifier,
	plans platformplanports.PlanService,
	workflows kernelports.WorkflowStarter,
) *Sender {
	return &Sender{drivers: drivers, plans: plans, workflows: workflows}
}

// Provide builds the Sender over the driver notification service.
func Provide(
	drivers *drivernotificationservice.Service,
	plans platformplanports.PlanService,
	workflows kernelports.WorkflowStarter,
) *Sender {
	return New(drivers, plans, workflows)
}

// Message is a message from the office to one driver.
type Message struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	Title      string
	Text       string
	// Priority defaults to medium.
	Priority notification.Priority
	// ShipmentID is the load the message opens in Dash, when it is about one.
	ShipmentID pulid.ID
}

// Notification is the message as the driver notification path takes it.
func (m *Message) Notification() *drivernotificationservice.DriverNotification {
	priority := m.Priority
	if priority == "" {
		priority = notification.PriorityMedium
	}
	request := &drivernotificationservice.DriverNotification{
		TenantInfo: m.TenantInfo,
		WorkerID:   m.WorkerID,
		EventType:  DispatchEvent,
		Priority:   priority,
		Context: notificationtemplate.DriverNotificationContext{
			AlertTitle:   strings.TrimSpace(m.Title),
			AlertMessage: strings.TrimSpace(m.Text),
		},
	}
	if m.ShipmentID.IsNotNil() {
		request.RelatedEntities = map[string]any{"shipmentId": m.ShipmentID.String()}
		request.Link = "/loads/" + m.ShipmentID.String()
	}

	return request
}

// Reach is how a message would reach its driver, and the message as they
// would read it.
type Reach struct {
	Channel Channel
	Notice  *notificationports.DriverNotificationPreview
	text    *notificationports.DriverSMSPreview
}

// Text is the message as the text it would go out as; empty unless the
// channel is text.
func (r *Reach) Text() string {
	if r == nil || r.text == nil {
		return ""
	}

	return r.text.Message
}

// Reach renders the message and says how it would get to the driver. It
// sends nothing.
func (s *Sender) Reach(ctx context.Context, message *Message) (*Reach, error) {
	notice, err := s.drivers.Preview(ctx, message.Notification())
	if err != nil {
		return nil, err
	}

	reach := &Reach{Notice: notice, Channel: ChannelNone}
	switch {
	case notice.Reachable:
		reach.Channel = ChannelDash
	case notice.Text != nil && s.texts(ctx, message.TenantInfo):
		reach.Channel = ChannelText
		reach.text = notice.Text
	}

	return reach, nil
}

// Texts reports whether the organization's plan includes text messages.
func (s *Sender) Texts(ctx context.Context, tenant pagination.TenantInfo) bool {
	return s.texts(ctx, tenant)
}

func (s *Sender) texts(ctx context.Context, tenant pagination.TenantInfo) bool {
	allowed, err := planservice.Allows(ctx, s.plans, tenant, platformplan.CapabilitySMS)

	return err == nil && allowed
}

// Send sends the message the way Reach says it goes and answers which way
// that was. A driver nothing reaches is sent nothing and the answer is
// ChannelNone: the notification path has always passed over a driver with
// no login, and a caller that must not lose the message asks Reach first.
func (s *Sender) Send(ctx context.Context, message *Message) (Channel, error) {
	request := message.Notification()
	notice, err := s.drivers.Preview(ctx, request)
	if err == nil && !notice.Reachable && notice.Text != nil && s.texts(ctx, message.TenantInfo) {
		return ChannelText, s.sendText(ctx, message, notice.Text)
	}

	s.drivers.Notify(ctx, request)
	if err == nil && notice.Reachable {
		return ChannelDash, nil
	}

	return ChannelNone, nil
}

func (s *Sender) sendText(
	ctx context.Context,
	message *Message,
	text *notificationports.DriverSMSPreview,
) error {
	_, err := s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID: fmt.Sprintf(
			"driver-message-%s-%d",
			message.WorkerID.String(),
			timeutils.NowUnixMilli(),
		),
		TaskQueue:     temporaltype.SMSTaskQueue,
		StaticSummary: "Sending a driver a dispatch message by text",
	}, "SendSMSWorkflow", &smscontract.SendSMSPayload{
		PhoneNumber:    text.PhoneNumber,
		Message:        text.Message,
		OrganizationID: message.TenantInfo.OrgID,
		BusinessUnitID: message.TenantInfo.BuID,
	})
	if err != nil {
		return fmt.Errorf("send the driver the message by text: %w", err)
	}

	return nil
}
