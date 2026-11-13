package playsendtest

import (
	"context"
	"strings"

	"go.temporal.io/sdk/client"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/drivermessage"
	"github.com/emoss08/trenova/internal/modules/agent/app/requestemail"
	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// Driver is one driver as the notification path sees them.
type Driver struct {
	Name string
	// OnDash is a portal login; Phone a number on file for a text.
	OnDash bool
	Phone  string
}

// Drivers is the driver notification path for a test: who can be reached
// how, and what was sent.
type Drivers struct {
	Known map[pulid.ID]Driver
	// Notified are the Dash messages sent, in order.
	Notified []*drivernotificationservice.DriverNotification
}

var _ drivermessage.Notifier = (*Drivers)(nil)

func (d *Drivers) Notify(_ context.Context, req *drivernotificationservice.DriverNotification) {
	if d.Known[req.WorkerID].OnDash {
		d.Notified = append(d.Notified, req)
	}
}

func (d *Drivers) Preview(
	_ context.Context,
	req *drivernotificationservice.DriverNotification,
) (*notificationports.DriverNotificationPreview, error) {
	driver := d.Known[req.WorkerID]
	words, _ := req.Context.(notificationtemplate.DriverNotificationContext)
	preview := &notificationports.DriverNotificationPreview{
		WorkerID:   req.WorkerID,
		WorkerName: driver.Name,
		Reachable:  driver.OnDash,
		Title:      words.AlertTitle,
		Message:    words.AlertMessage,
		Priority:   req.Priority,
		Link:       req.Link,
	}
	if !driver.OnDash && strings.TrimSpace(driver.Phone) != "" {
		preview.Text = &notificationports.DriverSMSPreview{
			PhoneNumber: driver.Phone,
			Message:     words.AlertTitle + ": " + words.AlertMessage,
		}
	}

	return preview, nil
}

// Plans is the organization's plan for a test: whether it includes text
// messages.
type Plans struct {
	platformplanports.PlanService

	Texts bool
}

func (p *Plans) RequireCapability(
	_ context.Context,
	_ pagination.TenantInfo,
	capability platformplan.Capability,
) error {
	if capability == platformplan.CapabilitySMS && !p.Texts {
		return errortypes.NewPlanRestrictionError(capability.String(), "", "")
	}

	return nil
}

// Texts records the text messages a test sent.
type Texts struct {
	kernelports.WorkflowStarter

	Sent []any
}

func (t *Texts) StartWorkflow(
	_ context.Context,
	_ client.StartWorkflowOptions,
	_ any,
	args ...any,
) (client.WorkflowRun, error) {
	t.Sent = append(t.Sent, args...)

	return nil, nil //nolint:nilnil // the caller only needs to know it started
}

// Messages builds the driver message sender over the test's drivers.
func Messages(drivers *Drivers, plans *Plans, texts *Texts) *drivermessage.Sender {
	return drivermessage.New(drivers, plans, texts)
}

// Mail is the request mailer for a test: what was rendered and sent.
type Mail struct {
	Mailer   *agenttoolstest.FakeMailer
	Renderer *agenttoolstest.FakeRenderer
	// NoProfile leaves the organization without an email profile to send
	// from; Refused puts every recipient on its do-not-email list.
	NoProfile bool
	Refused   bool
}

func NewMail() *Mail {
	return &Mail{Mailer: &agenttoolstest.FakeMailer{}, Renderer: &agenttoolstest.FakeRenderer{}}
}

// Requests builds the request email sender over the test's mail.
func (m *Mail) Requests() *requestemail.Sender {
	params := requestemail.Params{Email: m.Mailer, Templates: m.Renderer}
	if !m.NoProfile {
		params.Senders = agenttoolstest.FakeSenders{Refused: m.Refused}
	}

	return requestemail.New(params)
}
