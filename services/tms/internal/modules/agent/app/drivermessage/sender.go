package drivermessage

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

// Sender sends a driver a message from the office in Dash, where they have a
// login. It is the one owner of that choice, for an agent's tool and for a
// person's own request.
type Sender struct {
	drivers Notifier
}

func New(drivers Notifier) *Sender {
	return &Sender{drivers: drivers}
}

// Provide builds the Sender over the driver notification service.
func Provide(drivers *drivernotificationservice.Service) *Sender {
	return New(drivers)
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
}

// Reach renders the message and says how it would get to the driver. It
// sends nothing.
func (s *Sender) Reach(ctx context.Context, message *Message) (*Reach, error) {
	notice, err := s.drivers.Preview(ctx, message.Notification())
	if err != nil {
		return nil, err
	}

	reach := &Reach{Notice: notice, Channel: ChannelNone}
	if notice.Reachable {
		reach.Channel = ChannelDash
	}

	return reach, nil
}

// Send sends the message in Dash and answers which way that was. A driver
// with no login is sent nothing and the answer is ChannelNone: the
// notification path has always passed over a driver with no login, and a
// caller that must not lose the message asks Reach first.
func (s *Sender) Send(ctx context.Context, message *Message) (Channel, error) {
	request := message.Notification()
	notice, err := s.drivers.Preview(ctx, request)

	s.drivers.Notify(ctx, request)
	if err == nil && notice.Reachable {
		return ChannelDash, nil
	}

	return ChannelNone, nil
}
