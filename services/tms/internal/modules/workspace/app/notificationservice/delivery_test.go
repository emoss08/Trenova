package notificationservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type memberDirectory struct {
	tenantports.UserRepository

	members map[pulid.ID]*tenant.User
}

func (d *memberDirectory) GetTenantMember(
	_ context.Context,
	req tenantports.GetTenantMemberRequest,
) (*tenant.User, error) {
	user, ok := d.members[req.UserID]
	if !ok {
		return nil, errortypes.NewNotFoundError("User not found")
	}
	return user, nil
}

type storedPreferences struct {
	notificationports.PreferenceRepository

	byUser map[pulid.ID]*notification.Preference
}

func (s *storedPreferences) Get(
	_ context.Context,
	req notificationports.GetPreferenceRequest,
) (*notification.Preference, bool, error) {
	pref, ok := s.byUser[req.TenantInfo.UserID]
	return pref, ok, nil
}

type sentMail struct {
	sent chan *emailports.SendEmailRequest
}

func (m *sentMail) Send(_ context.Context, req *emailports.SendEmailRequest) (*email.Message, error) {
	m.sent <- req
	return &email.Message{}, nil
}

func (m *sentMail) SendPersisted(
	context.Context,
	*emailports.SendPersistedEmailRequest,
) (*email.Message, error) {
	return &email.Message{}, nil
}

type dispatchFixture struct {
	repo  *recordingNotificationRepo
	prefs *storedPreferences
	users *memberDirectory
	mail  *sentMail
	svc   *Service
	orgID pulid.ID
	buID  pulid.ID
	now   time.Time
}

func newDispatchFixture(t *testing.T) *dispatchFixture {
	t.Helper()
	f := &dispatchFixture{
		repo:  &recordingNotificationRepo{},
		prefs: &storedPreferences{byUser: map[pulid.ID]*notification.Preference{}},
		users: &memberDirectory{members: map[pulid.ID]*tenant.User{}},
		mail:  &sentMail{sent: make(chan *emailports.SendEmailRequest, 4)},
		orgID: pulid.MustNew("org_"),
		buID:  pulid.MustNew("bu_"),
		now:   time.Date(2026, time.October, 14, 17, 0, 0, 0, time.UTC),
	}
	f.svc = New(Params{
		Logger:      zap.NewNop(),
		Repo:        f.repo,
		Realtime:    silentRealtime{},
		Users:       f.users,
		Preferences: f.prefs,
		Email:       f.mail,
	})
	f.svc.now = func() time.Time { return f.now }
	f.svc.webBaseURL = "https://tms.example.com"
	return f
}

func (f *dispatchFixture) member(name string) *tenant.User {
	user := &tenant.User{
		ID:                 pulid.MustNew("usr_"),
		Name:               name,
		EmailAddress:       name + "@example.com",
		Timezone:           "America/Chicago",
		AvailabilityStatus: tenant.AvailabilityAvailable,
	}
	f.users.members[user.ID] = user
	return user
}

func (f *dispatchFixture) notice(target pulid.ID, eventType string) *notification.Notification {
	buID := f.buID
	return &notification.Notification{
		OrganizationID: f.orgID,
		BusinessUnitID: &buID,
		TargetUserID:   &target,
		Channel:        notification.ChannelUser,
		EventType:      eventType,
		Priority:       notification.PriorityMedium,
		Title:          "A tender needs your review",
		Message:        "S-48219 needs review",
		Source:         "tender",
		Data:           map[string]any{"link": "/dispatch/console?move=smv_1"},
	}
}

func deliveryOf(t *testing.T, n *notification.Notification) map[string]any {
	t.Helper()
	delivery, ok := n.Data[deliveryDataKey].(map[string]any)
	require.True(t, ok, "the notification carries the delivery decision")
	return delivery
}

func TestCreateStampsTheRecipientsDecisionForTheBrowser(t *testing.T) {
	t.Parallel()
	f := newDispatchFixture(t)
	maya := f.member("maya")
	pref := notification.DefaultPreference(f.orgID, f.buID, maya.ID)
	pref.DesktopMode = notification.DeliveryModeAll
	pref.EmailMode = notification.DeliveryModeOff
	pref.Sound = false
	f.prefs.byUser[maya.ID] = pref

	created, err := f.svc.Create(t.Context(), f.notice(maya.ID, "tender_accepted"))

	require.NoError(t, err)
	delivery := deliveryOf(t, created)
	assert.Equal(t, true, delivery["desktop"])
	assert.Equal(t, false, delivery["sound"])
	assert.Equal(t, string(notification.EmailDeliveryNone), delivery["email"])
	assert.Equal(t, "/dispatch/console?move=smv_1", created.Data["link"], "the producer's data is kept")
}

func TestCreateWithoutStoredPreferencesUsesTheDefaults(t *testing.T) {
	t.Parallel()
	f := newDispatchFixture(t)
	maya := f.member("maya")

	routine, err := f.svc.Create(t.Context(), f.notice(maya.ID, "tender_accepted"))
	require.NoError(t, err)
	assert.Equal(t, false, deliveryOf(t, routine)["desktop"], "urgent only by default")

	urgent, err := f.svc.Create(t.Context(), f.notice(maya.ID, "tender_waterfall_exhausted"))
	require.NoError(t, err)
	assert.Equal(t, true, deliveryOf(t, urgent)["desktop"])
	assert.Equal(t, string(notification.EmailDeliveryNow), deliveryOf(t, urgent)["email"])
}

func TestCreateEmailsAnUrgentNotificationRightAway(t *testing.T) {
	t.Parallel()
	f := newDispatchFixture(t)
	maya := f.member("maya")

	created, err := f.svc.Create(t.Context(), f.notice(maya.ID, "carrier_intel_block"))
	require.NoError(t, err)

	select {
	case sent := <-f.mail.sent:
		assert.Equal(t, []string{"maya@example.com"}, sent.To)
		assert.Equal(t, email.PurposeNotifications, sent.Purpose)
		assert.Equal(t, "notification-email-"+created.ID.String(), sent.IdempotencyKey)
		assert.Contains(t, sent.HTML, "https://tms.example.com/dispatch/console?move=smv_1")
		assert.Contains(t, sent.Text, "S-48219 needs review")
		assert.Contains(
			t,
			sent.Text,
			"Change what reaches you: https://tms.example.com/settings?section=notifications\n",
			"the email links to the notifications section of Settings",
		)
	case <-time.After(2 * time.Second):
		t.Fatal("the urgent notification was not emailed")
	}
}

func TestCreateHoldsEmailForADigestWhenAsked(t *testing.T) {
	t.Parallel()
	f := newDispatchFixture(t)
	maya := f.member("maya")
	pref := notification.DefaultPreference(f.orgID, f.buID, maya.ID)
	pref.EmailMode = notification.DeliveryModeAll
	pref.EmailDigest = true
	f.prefs.byUser[maya.ID] = pref

	created, err := f.svc.Create(t.Context(), f.notice(maya.ID, "tender_accepted"))

	require.NoError(t, err)
	assert.Equal(t, string(notification.EmailDeliveryDigest), deliveryOf(t, created)["email"])
	select {
	case <-f.mail.sent:
		t.Fatal("a digest notification must not be emailed on its own")
	case <-time.After(100 * time.Millisecond):
	}
}

func TestCreateHandsAReviewToTheDelegateOfSomeoneAway(t *testing.T) {
	t.Parallel()
	f := newDispatchFixture(t)
	jordan := f.member("jordan")
	maya := f.member("maya")
	maya.AvailabilityStatus = tenant.AvailabilityAway
	maya.AwayDelegateUserID = &jordan.ID

	_, err := f.svc.Create(t.Context(), f.notice(maya.ID, "tender_needs_review"))

	require.NoError(t, err)
	require.Len(t, f.repo.created, 2)
	handedOff := f.repo.created[1]
	assert.Equal(t, jordan.ID, *handedOff.TargetUserID)
	assert.Equal(t, maya.ID.String(), handedOff.Data[onBehalfOfIDKey])
	assert.Equal(t, "maya", handedOff.Data[onBehalfOfNameKey])
	assert.Equal(t, false, deliveryOf(t, f.repo.created[0])["desktop"], "the person away is not pinged")
}

func TestCreateDoesNotHandOffRoutineNewsOrOnceItIsBack(t *testing.T) {
	t.Parallel()
	f := newDispatchFixture(t)
	jordan := f.member("jordan")
	maya := f.member("maya")
	maya.AvailabilityStatus = tenant.AvailabilityAway
	maya.AwayDelegateUserID = &jordan.ID

	_, err := f.svc.Create(t.Context(), f.notice(maya.ID, "tender_accepted"))
	require.NoError(t, err)
	assert.Len(t, f.repo.created, 1, "only approvals, tender reviews and mentions are handed off")

	backOn := f.now.Add(-time.Hour).Unix()
	maya.AwayBackOn = &backOn
	_, err = f.svc.Create(t.Context(), f.notice(maya.ID, "tender_needs_review"))
	require.NoError(t, err)
	assert.Len(t, f.repo.created, 2, "no hand-off once the return day has come")
}

func TestCreateDoesNotHandOffAHandOff(t *testing.T) {
	t.Parallel()
	f := newDispatchFixture(t)
	maya := f.member("maya")
	jordan := f.member("jordan")
	maya.AvailabilityStatus = tenant.AvailabilityAway
	maya.AwayDelegateUserID = &jordan.ID
	jordan.AvailabilityStatus = tenant.AvailabilityAway
	jordan.AwayDelegateUserID = &maya.ID

	_, err := f.svc.Create(t.Context(), f.notice(maya.ID, notification.MentionEventType))

	require.NoError(t, err)
	assert.Len(t, f.repo.created, 2)
}

func TestDigestDueAtEightInThePersonsZoneOncePerDay(t *testing.T) {
	t.Parallel()
	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	pref := notification.DefaultPreference(pulid.MustNew("org_"), pulid.MustNew("bu_"), pulid.MustNew("usr_"))
	pref.EmailMode = notification.DeliveryModeAll
	pref.EmailDigest = true

	eight := time.Date(2026, time.October, 14, 8, 5, 0, 0, loc)
	assert.True(t, DigestDue(pref, eight, loc))
	assert.False(t, DigestDue(pref, eight.Add(time.Hour), loc))

	sent := eight.Add(-time.Hour).Unix()
	pref.LastDigestAt = &sent
	assert.False(t, DigestDue(pref, eight, loc), "not twice in one morning")

	yesterday := eight.Add(-24 * time.Hour).Unix()
	pref.LastDigestAt = &yesterday
	assert.True(t, DigestDue(pref, eight, loc))

	pref.EmailMode = notification.DeliveryModeUrgent
	assert.False(t, DigestDue(pref, eight, loc))
}
