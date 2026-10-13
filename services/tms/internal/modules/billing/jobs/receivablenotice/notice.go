// Package receivablenotice tells the people who collect receivables which
// customers an automatic email could not reach.
package receivablenotice

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

const (
	maxNamedCustomers = 5
	maxRecipients     = 25
	// repeatAfter keeps the same set of customers from being reported more
	// than once a week.
	repeatAfter = 7 * 24 * time.Hour
)

// Unreachable is one run's customers that had something to be sent and
// nobody to send it to.
type Unreachable struct {
	Tenant    pagination.TenantInfo
	Customers []string
	Now       int64
	// EventType, Title, Source and Link describe the notification; Reason
	// follows the customer names: "have overdue invoices but no ...".
	EventType string
	Title     string
	Reason    string
	Source    string
	Link      string
}

// Report notifies the people who record customer payments, naming the first
// few customers. It logs rather than fails: the emails that could be sent
// were sent.
func Report(
	ctx context.Context,
	notifier *notificationservice.Service,
	l *zap.Logger,
	u *Unreachable,
) {
	if notifier == nil || len(u.Customers) == 0 {
		return
	}
	named := u.Customers
	if len(named) > maxNamedCustomers {
		named = named[:maxNamedCustomers]
	}
	message := strings.Join(named, ", ")
	if rest := len(u.Customers) - len(named); rest > 0 {
		message += " and " + strconv.Itoa(rest) + " more"
	}
	correlation := u.EventType + ":" + strings.Join(u.Customers, "|")
	if _, err := notifier.NotifyPermitted(ctx, notificationservice.NotifyPermittedRequest{
		Tenant:      u.Tenant,
		Resource:    permission.ResourceCustomerPayment,
		Operation:   permission.OpCreate,
		Limit:       maxRecipients,
		Now:         u.Now,
		DedupeSince: u.Now - int64(repeatAfter.Seconds()),
		Notification: notification.Notification{
			EventType:     u.EventType,
			Priority:      notification.PriorityMedium,
			Title:         u.Title,
			Message:       message + " " + u.Reason,
			Source:        u.Source,
			CorrelationID: &correlation,
			Data:          map[string]any{"link": u.Link},
		},
	}); err != nil {
		l.Warn("failed to report customers no email could reach",
			zap.String("eventType", u.EventType), zap.Error(err))
	}
}
