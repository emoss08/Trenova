package workspacefixtures

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/homelayout/homelayoutdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification/notificationdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagefavorite"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagefavorite/pagefavoritedb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	homeScreenNotificationSource = "home_screen_fixture"
	homeScreenNotificationEvent  = "fixture.home_screen"
	cannedCustomerScorecard      = "customer-scorecard"
)

var ErrHomeScreenLayoutInvalid = errors.New("the seeded home screen is not a valid layout")

type HomeScreenSeed struct{}

func (s *HomeScreenSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	admin, err := tenantfixtures.AdminUser(ctx, db)
	if err != nil {
		return err
	}

	if err = s.ensureLayout(ctx, db, org, admin); err != nil {
		return err
	}
	if err = s.ensureFavorites(ctx, db, org, admin); err != nil {
		return err
	}

	return s.ensureNotifications(ctx, db, org, admin)
}

func HomeScreenDocument() *homelayout.Document {
	widgets := []homelayout.Widget{
		homeWidget(homelayout.WidgetAnnouncement, 12, 2, homelayout.WidgetConfig{
			Text: "Peak season starts November 23. Time off between then and December 31 " +
				"needs a manager's approval, and every reefer load gets a pre-trip " +
				"temperature check before it leaves the yard.",
		}),
		homeWidget(homelayout.WidgetKPIRow, 12, 2, homelayout.WidgetConfig{
			Metrics: []string{
				"activeShipments",
				"readyToDispatch",
				"onTimePercent",
				"emptyMilePercent",
				"marginPercent",
				"arOverdue",
			},
		}),
		homeWidget(homelayout.WidgetKPI, 3, 3, homelayout.WidgetConfig{Metric: "revenueToday"}),
		homeWidget(homelayout.WidgetOnTimeGoal, 3, 3, homelayout.WidgetConfig{}),
		homeWidget(homelayout.WidgetARSnapshot, 6, 3, homelayout.WidgetConfig{}),
		homeWidget(homelayout.WidgetUnassigned, 6, 5, homelayout.WidgetConfig{Limit: 8}),
		homeWidget(homelayout.WidgetExceptions, 6, 5, homelayout.WidgetConfig{Limit: 8}),
		homeWidget(homelayout.WidgetTomorrowsPickups, 8, 5, homelayout.WidgetConfig{Limit: 10}),
		homeWidget(homelayout.WidgetDetentionWatch, 4, 5, homelayout.WidgetConfig{Limit: 8}),
		homeWidget(homelayout.WidgetMap, 8, 5, homelayout.WidgetConfig{}),
		homeWidget(homelayout.WidgetFleetStatus, 4, 3, homelayout.WidgetConfig{}),
		homeWidget(homelayout.WidgetQuickActions, 4, 2, homelayout.WidgetConfig{}),
		homeWidget(homelayout.WidgetRevenueTrend, 8, 4, homelayout.WidgetConfig{WindowDays: 90}),
		homeWidget(homelayout.WidgetWorkerAttention, 4, 4, homelayout.WidgetConfig{}),
		homeWidget(homelayout.WidgetLaneHeatmap, 6, 5, homelayout.WidgetConfig{}),
		homeWidget(homelayout.WidgetCustomerMix, 6, 5, homelayout.WidgetConfig{}),
		homeWidget(homelayout.WidgetReport, 8, 5, homelayout.WidgetConfig{
			CannedKey: cannedCustomerScorecard,
			Limit:     10,
		}),
		homeWidget(homelayout.WidgetAIInsights, 4, 5, homelayout.WidgetConfig{}),
		homeWidget(homelayout.WidgetActivity, 6, 4, homelayout.WidgetConfig{Limit: 10}),
		homeWidget(homelayout.WidgetNotifications, 3, 4, homelayout.WidgetConfig{Limit: 8}),
		homeWidget(homelayout.WidgetFavorites, 3, 4, homelayout.WidgetConfig{}),
		homeWidget(homelayout.WidgetJumpBackIn, 4, 3, homelayout.WidgetConfig{Limit: 6}),
		homeWidget(homelayout.WidgetExpiringCredentials, 4, 3, homelayout.WidgetConfig{}),
		homeWidget(homelayout.WidgetMyApprovals, 4, 3, homelayout.WidgetConfig{}),
	}
	for i := range widgets {
		widgets[i].ID = homelayout.NextWidgetID(widgets[:i])
	}

	return &homelayout.Document{
		SchemaVersion: homelayout.DocumentSchemaVersion,
		Mode:          homelayout.ModeCustom,
		Density:       homelayout.DensityComfortable,
		Layout:        &homelayout.Layout{Widgets: widgets},
	}
}

func homeWidget(key string, w, h int, config homelayout.WidgetConfig) homelayout.Widget {
	return homelayout.Widget{Key: key, W: w, H: h, Config: config}
}

func (s *HomeScreenSeed) ensureLayout(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
	admin *tenant.User,
) error {
	document := HomeScreenDocument()
	multiErr := errortypes.NewMultiError()
	document.Validate(multiErr)
	if multiErr.HasErrors() {
		return fmt.Errorf("%w: %s", ErrHomeScreenLayoutInvalid, multiErr.Error())
	}

	p := homelayoutdb.Preferences
	existing, err := p.AllTenants().Where(
		p.OrganizationID.Eq(org.ID),
		p.BusinessUnitID.Eq(org.BusinessUnitID),
		p.UserID.Eq(admin.ID),
	).First(ctx, db)
	switch {
	case errors.Is(err, keel.ErrNotFound):
		if err = p.Create(ctx, db, &homelayout.Preference{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			UserID:         admin.ID,
			Preferences:    document,
		}); err != nil {
			return fmt.Errorf("create the administrator's home screen: %w", err)
		}
	case err != nil:
		return fmt.Errorf("look for the administrator's home screen: %w", err)
	default:
		existing.Preferences = document
		if err = p.Save(ctx, db, existing, p.Preferences); err != nil {
			return fmt.Errorf("replace the administrator's home screen: %w", err)
		}
	}

	return nil
}

type homeFavorite struct {
	url   string
	title string
}

func homeFavorites() []homeFavorite {
	return []homeFavorite{
		{url: "/dispatch/console", title: "Console"},
		{url: "/dispatch/outlook", title: "Freight outlook"},
		{url: "/shipment-management/shipments", title: "Shipments"},
		{url: "/billing/queue", title: "Billing queue"},
		{url: "/accounting/ar/aging", title: "AR aging"},
	}
}

func (s *HomeScreenSeed) ensureFavorites(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
	admin *tenant.User,
) error {
	favorites := homeFavorites()
	urls := make([]string, 0, len(favorites))
	for _, favorite := range favorites {
		urls = append(urls, favorite.url)
	}

	f := pagefavoritedb.PageFavorites
	pinned, err := f.AllTenants().Where(
		f.OrganizationID.Eq(org.ID),
		f.BusinessUnitID.Eq(org.BusinessUnitID),
		f.UserID.Eq(admin.ID),
		f.PageURL.In(urls...),
	).Limit(len(urls)).List(ctx, db)
	if err != nil {
		return fmt.Errorf("list the administrator's pinned pages: %w", err)
	}
	have := make(map[string]struct{}, len(pinned))
	for _, favorite := range pinned {
		have[favorite.PageURL] = struct{}{}
	}

	missing := make([]*pagefavorite.PageFavorite, 0, len(favorites))
	for _, favorite := range favorites {
		if _, ok := have[favorite.url]; ok {
			continue
		}
		missing = append(missing, &pagefavorite.PageFavorite{
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
			UserID:         admin.ID,
			PageURL:        favorite.url,
			PageTitle:      favorite.title,
		})
	}
	if len(missing) == 0 {
		return nil
	}
	if err = f.CreateMany(ctx, db, missing); err != nil {
		return fmt.Errorf("pin pages for the administrator: %w", err)
	}

	return nil
}

type homeNotification struct {
	priority notification.Priority
	title    string
	message  string
	link     string
}

func homeNotifications() []homeNotification {
	return []homeNotification{
		{
			priority: notification.PriorityCritical,
			title:    "Reefer alarm on a load in transit",
			message:  "The trailer is reading 41°F against a 34°F setpoint. Call the driver before the next check call.",
			link:     "/shipment-management/shipments",
		},
		{
			priority: notification.PriorityHigh,
			title:    "Three invoices are waiting on your approval",
			message:  "They have been in the queue since yesterday and hold up this week's statement run.",
			link:     "/billing/pending-approvals",
		},
		{
			priority: notification.PriorityMedium,
			title:    "A carrier's cargo insurance expires in 9 days",
			message:  "Tenders to this carrier stop once the policy lapses. Ask for the renewal certificate.",
			link:     "/dispatch/carriers",
		},
		{
			priority: notification.PriorityLow,
			title:    "Tomorrow's pickups are fully covered",
			message:  "Every pickup window has a driver and a tractor assigned.",
			link:     "/dispatch/console",
		},
	}
}

func (s *HomeScreenSeed) ensureNotifications(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
	admin *tenant.User,
) error {
	n := notificationdb.Notifications
	seeded, err := n.AllTenants().Where(
		n.OrganizationID.Eq(org.ID),
		n.TargetUserID.Eq(admin.ID),
		n.Source.Eq(homeScreenNotificationSource),
	).Exists(ctx, db)
	if err != nil {
		return fmt.Errorf("look for the administrator's seeded notifications: %w", err)
	}
	if seeded {
		return nil
	}

	notices := homeNotifications()
	rows := make([]*notification.Notification, 0, len(notices))
	for _, notice := range notices {
		buID := org.BusinessUnitID
		targetUserID := admin.ID
		rows = append(rows, &notification.Notification{
			OrganizationID: org.ID,
			BusinessUnitID: &buID,
			TargetUserID:   &targetUserID,
			EventType:      homeScreenNotificationEvent,
			Priority:       notice.priority,
			Channel:        notification.ChannelUser,
			DeliveryStatus: notification.DeliveryStatusDelivered,
			MaxRetries:     3,
			Title:          notice.title,
			Message:        notice.message,
			Data:           map[string]any{"link": notice.link},
			Source:         homeScreenNotificationSource,
		})
	}
	if err = n.CreateMany(ctx, db, rows); err != nil {
		return fmt.Errorf("create the administrator's notifications: %w", err)
	}

	return nil
}
