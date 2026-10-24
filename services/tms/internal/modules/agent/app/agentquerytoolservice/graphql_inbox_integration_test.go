//go:build integration

package agentquerytoolservice_test

import (
	"context"
	"errors"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture/capturedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email/emaildb"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/agentread"
	"github.com/emoss08/trenova/internal/api/graphql/agentread/agentreadtest"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentquerytoolservice"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/capturerepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/inboundmessagerepository"
	"github.com/emoss08/trenova/internal/modules/documents/adapters/postgres/inboundreplyrepository"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureservice"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/organizationrepository"
	"github.com/emoss08/trenova/internal/modules/identity/adapters/postgres/userrepository"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/adapters/postgres/emailrepository"
	"github.com/emoss08/trenova/internal/modules/workspace/app/emailservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/email"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/readscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	inboxMessagesPerTenant = 5
	inboxWaitingPerTenant  = 2
	inboxLongBody          = 7000
	inboxEndOfDayMinute    = 1050
)

// inboxDirectory is the role directory the assignees are read from: the
// people of each organization who may work its inbox. Who holds which role
// is the permission module's to prove; the inbox reads the people it names.
type inboxDirectory struct {
	permissionports.RoleRepository

	people map[pulid.ID][]permissionports.PermittedUser
}

func (d *inboxDirectory) ListUsersWithPermission(
	_ context.Context,
	req permissionports.ListUsersWithPermissionRequest,
) ([]permissionports.PermittedUser, error) {
	return d.people[req.OrganizationID], nil
}

// organizationCacheMiss is an organization cache that holds nothing, so
// every read of the queue's time zone reads the organization.
type organizationCacheMiss struct{}

func (organizationCacheMiss) GetByID(context.Context, pulid.ID) (*tenant.Organization, error) {
	return nil, errOrganizationNotCached
}

var errOrganizationNotCached = errors.New("organization not cached")

// inboxEnv is the inbox's reads over two tenants of a fresh database: the
// services the screen's resolvers call over the real repositories, and the
// loaders its fields resolve through.
type inboxEnv struct {
	admin     *keel.DB
	engine    permissionports.PermissionEngine
	reader    agentports.GraphQLReader
	public    agentports.GraphQLReader
	directory *inboxDirectory
	attach    func(ctx context.Context, tenant *agentreadtest.Tenant) context.Context
}

func newInboxEnv(t *testing.T) *inboxEnv {
	t.Helper()

	_, admin, dsn, cleanup := seedtest.SetupTestDBWithDSN(t)
	t.Cleanup(cleanup)

	conn := dbtest.NewRLSConnection(t, admin, dsn)
	logger := zap.NewNop()
	params := inboundmessagerepository.Params{DB: conn, Logger: logger}
	messages := inboundmessagerepository.New(params)
	queue := inboundmessagerepository.NewQueueRepository(params)
	replies := inboundreplyrepository.New(inboundreplyrepository.Params{DB: conn})
	users := userrepository.New(userrepository.Params{DB: conn, Logger: logger})
	directory := &inboxDirectory{people: map[pulid.ID][]permissionports.PermittedUser{}}
	engine := &mocks.DenyResourcesPermissionEngine{}
	captures := capturerepository.Params{DB: conn, Logger: logger}

	resolvers := agentread.ResolverParams{
		InboundMessageService: inboundmessageservice.New(inboundmessageservice.Params{
			Logger: logger,
			MailboxRepo: inboundmessagerepository.NewMailboxRepository(
				inboundmessagerepository.MailboxParams{DB: conn, Logger: logger}),
			MessageRepo:  messages,
			DB:           conn,
			QueueRepo:    queue,
			SettingsRepo: inboundmessagerepository.NewSettingsRepository(params),
			Roles:        directory,
			Organizations: organizationrepository.New(organizationrepository.Params{
				DB: conn, Logger: logger, Cache: organizationCacheMiss{},
			}),
		}),
		CaptureService: captureservice.New(captureservice.Params{
			Logger:      logger,
			DB:          conn,
			Batches:     capturerepository.NewBatchRepository(captures),
			Items:       capturerepository.NewItemRepository(captures),
			Pages:       capturerepository.NewPageRepository(captures),
			Devices:     capturerepository.NewDeviceRepository(captures),
			Permissions: engine,
		}),
		EmailService: emailservice.New(emailservice.Params{
			Logger: logger,
			Repo:   emailrepository.New(emailrepository.Params{DB: conn, Logger: logger}),
		}),
	}

	people := loaders.NewUserByIDLoaderFactory(loaders.UserByIDLoaderFactoryParams{Repo: users})
	counts := loaders.NewInboundAttachmentCountLoaderFactory(
		loaders.InboundAttachmentCountLoaderFactoryParams{MessageRepo: messages})
	threads := loaders.NewInboundThreadCountLoaderFactory(
		loaders.InboundThreadCountLoaderFactoryParams{QueueRepo: queue})
	attachments := loaders.NewInboundAttachmentsByMessageIDLoaderFactory(
		loaders.InboundAttachmentsByMessageIDLoaderFactoryParams{MessageRepo: messages})
	drafts := loaders.NewInboundReplyLoaderFactory(
		loaders.InboundReplyLoaderFactoryParams{ReplyRepo: replies})

	return &inboxEnv{
		admin:     admin,
		engine:    engine,
		directory: directory,
		reader: agentreadtest.NewReader(t, agentreadtest.Options{
			Engine: engine, Resolvers: resolvers,
		}),
		public: agentreadtest.NewReader(t, agentreadtest.Options{
			Engine: &publicRole{}, Resolvers: resolvers,
		}),
		attach: func(ctx context.Context, tenant *agentreadtest.Tenant) context.Context {
			info := tenantInfoOf(tenant)
			ctx = people.Attach(ctx, info)
			ctx = counts.Attach(ctx, info)
			ctx = threads.Attach(ctx, info)
			ctx = attachments.Attach(ctx, info)

			return drafts.Attach(ctx, info)
		},
	}
}

// inboxTenant is one tenant's inbox: its mailbox, its messages (the first
// inboxWaitingPerTenant wait on a person), its sending profiles (the last
// is inactive) and its intake stacks (the last is filed).
type inboxTenant struct {
	ctx         context.Context
	tenant      *agentreadtest.Tenant
	mailbox     string
	address     string
	messages    []string
	waiting     []string
	long        string
	profiles    []string
	inactive    string
	batches     []string
	filedBatch  string
	batchItems  map[string][]string
	senderEmail string
}

func (e *inboxEnv) write(t *testing.T, tenant *agentreadtest.Tenant, label string) *inboxTenant {
	t.Helper()

	ctx := t.Context()
	org, bu, user := tenant.Actor.OrganizationID, tenant.Actor.BusinessUnitID, tenant.Actor.UserID
	writeCtx := tenant.Ctx(ctx)
	must := func(err error) {
		t.Helper()
		require.NoError(t, err)
	}
	out := &inboxTenant{
		ctx:        e.attach(readscope.With(dbscope.WithTenant(ctx, tenant.Scope()), nil), tenant),
		tenant:     tenant,
		batchItems: map[string][]string{},
	}

	mailbox := &inboundmessage.Mailbox{
		ID: pulid.MustNew("imbx_"), OrganizationID: org, BusinessUnitID: bu,
		Name: "Dispatch " + label, Address: "dispatch-" + strings.ToLower(label) + "@inbox.test",
		Provider: inboundmessage.ProviderPostmark, TokenHash: pulid.MustNew("tok_").String(),
		ReviewPolicy: inboundmessage.ReviewAlways, Status: inboundmessage.MailboxActive,
	}
	must(inboundmessagedb.Mailboxes.Create(writeCtx, e.admin, mailbox))
	out.mailbox, out.address = mailbox.ID.String(), mailbox.Address

	for idx := range inboxMessagesPerTenant {
		suffix := label + string(rune('1'+idx))
		message := &inboundmessage.InboundMessage{
			ID: pulid.MustNew("imsg_"), OrganizationID: org, BusinessUnitID: bu,
			MailboxID: mailbox.ID, ProviderMessageID: "pm-" + suffix,
			FromAddress: "broker" + suffix + "@example.com", FromName: "Broker " + suffix,
			ToAddresses: []string{mailbox.Address}, Subject: "Load " + suffix,
			TextBody: "Please confirm load " + suffix + ".", ReceivedAt: int64(1_790_000_000 + idx*60),
			Status: inboundmessage.StatusActioned, ThreadKey: "thread-" + suffix,
			SenderDomain: "example.com",
		}
		if idx < inboxWaitingPerTenant {
			message.Status = inboundmessage.StatusInReview
			message.OwnerID = user
		}
		if idx == inboxMessagesPerTenant-1 {
			message.TextBody = strings.Repeat("x", inboxLongBody)
			out.long = message.ID.String()
		}
		must(inboundmessagedb.InboundMessages.Create(writeCtx, e.admin, message))
		out.messages = append(out.messages, message.ID.String())
		if idx < inboxWaitingPerTenant {
			out.waiting = append(out.waiting, message.ID.String())
		}
	}

	for idx := range 3 {
		suffix := label + string(rune('1'+idx))
		profile := &email.Profile{
			ID: pulid.MustNew("emlprof_"), OrganizationID: org, BusinessUnitID: bu,
			Name: "Profile " + suffix, SenderName: "Ops " + suffix,
			SenderEmail: "ops" + suffix + "@carrier.test", ReplyToEmail: "reply" + suffix + "@carrier.test",
			Provider: email.ProviderResend, AuthType: email.AuthTypeAPIKey,
			EncryptionType: email.EncryptionNone, Status: email.ProfileStatusActive,
		}
		if idx == 2 {
			profile.Status = email.ProfileStatusInactive
			out.inactive = profile.ID.String()
		} else {
			out.profiles = append(out.profiles, profile.ID.String())
		}
		if idx == 0 {
			out.senderEmail = profile.SenderEmail
		}
		must(emaildb.Profiles.Create(writeCtx, e.admin, profile))
	}

	device := &capture.CaptureDevice{
		ID: pulid.MustNew("cdev_"), OrganizationID: org, BusinessUnitID: bu, UserID: user,
		Name: "Scanner " + label, MachineName: "DESK-" + label, AgentVersion: "1.0.0",
		Architecture: capture.ArchitectureX64, Status: capture.DeviceActive,
		RefreshTokenHash: pulid.MustNew("rt_").String(), AccessTokenHash: pulid.MustNew("at_").String(),
		AccessTokenExpiresAt: 10,
	}
	must(capturedb.CaptureDevices.Create(writeCtx, e.admin, device))
	for idx := range 4 {
		batch := &capture.CaptureBatch{
			ID: pulid.MustNew("cbat_"), OrganizationID: org, BusinessUnitID: bu, UserID: user,
			DeviceID: device.ID, ClientKey: pulid.MustNew("ck_").String(),
			Source: capture.SourceScan, Status: capture.BatchReady, RetainUntil: 1 << 40,
			ItemCount: 2, CreatedAt: int64(1_790_000_000 + idx*60),
		}
		if idx == 3 {
			batch.Status = capture.BatchFiled
			out.filedBatch = batch.ID.String()
		} else {
			out.batches = append(out.batches, batch.ID.String())
		}
		must(capturedb.CaptureBatches.Create(writeCtx, e.admin, batch))
		for position := 2; position >= 1; position-- {
			item := &capture.CaptureItem{
				ID: pulid.MustNew("citm_"), OrganizationID: org, BusinessUnitID: bu,
				BatchID: batch.ID, Position: position, Status: capture.ItemProposed,
				PageIDs: []pulid.ID{pulid.MustNew("cpag_")},
			}
			must(capturedb.CaptureItems.Create(writeCtx, e.admin, item))
			out.batchItems[batch.ID.String()] = append(
				[]string{item.ID.String()}, out.batchItems[batch.ID.String()]...)
		}
	}

	e.directory.people[org] = []permissionports.PermittedUser{{UserID: user, Name: "Reader " + label}}
	settings := triage.DefaultSettings(org, bu)
	settings.OnCallUserID, settings.EndOfDayMinute = user, inboxEndOfDayMinute
	must(triagedb.Settingses.Create(writeCtx, e.admin, settings))

	return out
}

// inboxCall is a call as the tenant's person at a data access ceiling.
func inboxCall(
	t *testing.T,
	tool agentports.AgentQueryTool,
	tenant *inboxTenant,
	ceiling permission.FieldSensitivity,
	args map[string]any,
) (map[string]any, error) {
	t.Helper()

	actor := tenant.tenant.Actor
	data, err := tool.Query(tenant.ctx, &agentports.QueryToolParams{
		OrganizationID: actor.OrganizationID, BusinessUnitID: actor.BusinessUnitID,
		Actor: actor, Timezone: "America/Chicago", Params: args,
		DataAccessCeiling: ceiling, RunID: pulid.MustNew("ar_"),
	})
	if err != nil {
		return nil, err
	}

	return asMap(t, data), nil
}

func withheldPaths(page map[string]any) []string {
	raw, _ := page["withheldByAccess"].([]any)
	paths := make([]string, 0, len(raw))
	for _, path := range raw {
		paths = append(paths, path.(string))
	}

	return paths
}

// Every inbox read reads, through the screen's resolvers under row-level
// security, the tenant's own records and none of another's, pages through a
// list, narrows by a typed filter, shows at the agent's default data access
// of Internal what the repository tools showed, and shows a Public role
// nothing but ids.
func TestConvertedLists_InboxReadsUnderRowLevelSecurity(t *testing.T) {
	env := newInboxEnv(t)
	ours := env.write(t, agentreadtest.SeedTenant(t, env.admin, "A"), "A")
	other := env.write(t, agentreadtest.SeedTenant(t, env.admin, "B"), "B")
	actor := ours.tenant.Actor

	list := func(t *testing.T, name string, reader agentports.GraphQLReader) agentports.AgentQueryTool {
		t.Helper()
		tool, err := agentquerytoolservice.ConvertedListTool(name, reader, env.engine)
		require.NoError(t, err)

		return tool
	}

	t.Run("inbox_list_inbound_messages", func(t *testing.T) {
		tool := list(t, "inbox_list_inbound_messages", env.reader)

		ids, pages := walkByKey(t, ours.ctx, tool, actor, nil, "id")
		assert.Greater(t, pages, 1, "the walk must cross a page")
		assert.ElementsMatch(t, ours.messages, ids)
		for _, id := range other.messages {
			assert.NotContains(t, ids, id)
		}

		waiting, _ := walkByKey(t, ours.ctx, tool, actor,
			map[string]any{"status": []any{"InReview"}}, "id")
		assert.ElementsMatch(t, ours.waiting, waiting)

		mine, _ := walkByKey(t, ours.ctx, tool, actor,
			map[string]any{"mailboxId": ours.mailbox}, "id")
		assert.ElementsMatch(t, ours.messages, mine)
		theirs, _ := walkByKey(t, ours.ctx, tool, actor,
			map[string]any{"mailboxId": other.mailbox}, "id")
		assert.Empty(t, theirs, "another tenant's mailbox lists nothing")

		page, err := inboxCall(t, tool, ours, permission.SensitivityInternal,
			map[string]any{"limit": 10})
		require.NoError(t, err)
		assert.Empty(t, withheldPaths(page), "a mailbox's name and address are its "+
			"reference view, Internal, as the repository tool showed the address")
		for _, item := range page["items"].([]any) {
			row := item.(map[string]any)
			assert.Equal(t, ours.mailbox, row["mailboxId"])
			assert.NotContains(t, row, "fromAddress", "the sender is written once, as from")
			assert.Contains(t, row["from"], "@example.com")
			mailbox, _ := row["mailbox"].(map[string]any)
			assert.Equal(t, ours.address, mailbox["address"])
		}

		hidden := list(t, "inbox_list_inbound_messages", env.public)
		page = callTool(t, ours.ctx, hidden, actor, map[string]any{"limit": 10})
		assert.Contains(t, withheldPaths(page), "subject")
		for _, item := range page["items"].([]any) {
			row := item.(map[string]any)
			assert.Contains(t, row, "id")
			assert.NotContains(t, row, "subject")
			assert.NotContains(t, row, "preview")
		}
	})

	t.Run("inbox_list_email_profiles", func(t *testing.T) {
		tool := list(t, "inbox_list_email_profiles", env.reader)

		active, pages := walkByKey(t, ours.ctx, tool, actor, nil, "id")
		assert.Equal(t, 1, pages)
		assert.ElementsMatch(t, ours.profiles, active, "the active ones unless asked")
		inactive, _ := walkByKey(t, ours.ctx, tool, actor,
			map[string]any{"status": []any{"Inactive"}}, "id")
		assert.Equal(t, []string{ours.inactive}, inactive)
		for _, id := range append(slices.Clone(other.profiles), other.inactive) {
			assert.NotContains(t, append(active, inactive...), id)
		}

		page, err := inboxCall(t, tool, ours, permission.SensitivityInternal, map[string]any{})
		require.NoError(t, err)
		assert.Empty(t, withheldPaths(page),
			"who mail comes from is Internal: the profile is how an agent chooses a sender")
		senders := make([]any, 0, 2)
		for _, item := range page["items"].([]any) {
			row := item.(map[string]any)
			senders = append(senders, row["senderEmail"])
			assert.NotEmpty(t, row["name"])
			assert.NotEmpty(t, row["replyToEmail"])
			assert.NotContains(t, row, "provider", "how the profile connects is never read")
		}
		assert.Contains(t, senders, ours.senderEmail)

		hidden := list(t, "inbox_list_email_profiles", env.public)
		page = callTool(t, ours.ctx, hidden, actor, map[string]any{})
		assert.Contains(t, withheldPaths(page), "senderEmail")
		for _, item := range page["items"].([]any) {
			assert.NotContains(t, item.(map[string]any), "senderEmail")
		}
	})

	t.Run("inbox_list_capture_batches", func(t *testing.T) {
		tool := list(t, "inbox_list_capture_batches", env.reader)

		ids, pages := walkByKey(t, ours.ctx, tool, actor, nil, "id")
		assert.Greater(t, pages, 1, "the walk must cross a page")
		assert.ElementsMatch(t, ours.batches, ids, "the open stacks unless asked")
		for _, id := range append(slices.Clone(other.batches), other.filedBatch, ours.filedBatch) {
			assert.NotContains(t, ids, id)
		}
		filed, _ := walkByKey(t, ours.ctx, tool, actor,
			map[string]any{"status": []any{"Filed"}}, "id")
		assert.Equal(t, []string{ours.filedBatch}, filed)

		page := callTool(t, ours.ctx, tool, actor, map[string]any{"limit": 10})
		items, _ := page["items"].([]any)
		require.Len(t, items, len(ours.batches))
		for _, item := range items {
			row := item.(map[string]any)
			documents, _ := row["items"].([]any)
			got := make([]string, 0, len(documents))
			for _, document := range documents {
				got = append(got, document.(map[string]any)["id"].(string))
			}
			assert.Equal(t, ours.batchItems[row["id"].(string)], got,
				"each stack's documents are read with the page, in position order")
		}
		assert.Empty(t, page["withheldByAccess"])

		one := callTool(t, ours.ctx, tool, actor, map[string]any{"captureBatchId": ours.filedBatch})
		batch, _ := one["batch"].(map[string]any)
		require.NotNil(t, batch)
		assert.Equal(t, ours.filedBatch, batch["id"])
		assert.Len(t, batch["items"], 2)

		_, err := inboxCall(t, tool, ours, permission.SensitivityInternal,
			map[string]any{"captureBatchId": other.batches[0]})
		require.Error(t, err, "another tenant's stack is not found")

		hidden := list(t, "inbox_list_capture_batches", env.public)
		page = callTool(t, ours.ctx, hidden, actor, map[string]any{"limit": 10})
		assert.Contains(t, withheldPaths(page), "status")
		for _, item := range page["items"].([]any) {
			assert.NotContains(t, item.(map[string]any), "status")
		}
	})

	t.Run("inbox_list_triage_assignees", func(t *testing.T) {
		tool := list(t, "inbox_list_triage_assignees", env.reader)

		page := callTool(t, ours.ctx, tool, actor, map[string]any{})
		items, _ := page["items"].([]any)
		require.Len(t, items, 1)
		person := items[0].(map[string]any)
		user, _ := person["user"].(map[string]any)
		assert.Equal(t, actor.UserID.String(), user["id"])
		assert.EqualValues(t, inboxWaitingPerTenant, person["waiting"],
			"only this tenant's waiting messages are counted")
		assert.Equal(t, true, person["onCall"])
		settings, _ := page["settings"].(map[string]any)
		require.NotNil(t, settings)
		assert.Equal(t, actor.UserID.String(), settings["onCallUserId"])
		assert.Equal(t, "17:30", settings["endOfDay"])

		hidden := list(t, "inbox_list_triage_assignees", env.public)
		page = callTool(t, ours.ctx, hidden, actor, map[string]any{})
		assert.Contains(t, withheldPaths(page), "waiting")
		for _, item := range page["items"].([]any) {
			assert.NotContains(t, item.(map[string]any), "waiting")
		}
	})
}

// One message reads whole for the tenant and is not found for another; the
// body is bounded and says so, the sender is written once, the mailbox's
// address shows at Internal as it did, and a Public role sees no body.
func TestConvertedGets_InboundMessageReadsUnderRowLevelSecurity(t *testing.T) {
	env := newInboxEnv(t)
	ours := env.write(t, agentreadtest.SeedTenant(t, env.admin, "A"), "A")
	other := env.write(t, agentreadtest.SeedTenant(t, env.admin, "B"), "B")

	get, err := agentquerytoolservice.ConvertedGetTool("inbox_get_inbound_message", env.reader, env.engine)
	require.NoError(t, err)

	record, err := inboxCall(t, get, ours, permission.SensitivityInternal,
		map[string]any{"messageId": ours.waiting[0]})
	require.NoError(t, err)
	assert.Equal(t, ours.waiting[0], record["id"])
	assert.Equal(t, "InReview", record["status"])
	assert.Equal(t, ours.tenant.Actor.UserID.String(), record["ownerId"])
	assert.Contains(t, record["from"], "@example.com")
	assert.NotContains(t, record, "fromName")
	assert.Empty(t, withheldPaths(record))
	mailbox, _ := record["mailbox"].(map[string]any)
	assert.Equal(t, ours.address, mailbox["address"])

	hidden, err := agentquerytoolservice.ConvertedGetTool("inbox_get_inbound_message", env.public, env.engine)
	require.NoError(t, err)
	record, err = inboxCall(t, hidden, ours, permission.SensitivityInternal,
		map[string]any{"messageId": ours.waiting[0]})
	require.NoError(t, err)
	assert.Equal(t, ours.waiting[0], record["id"])
	assert.Contains(t, withheldPaths(record), "textBody")
	assert.NotContains(t, record, "textBody")

	long, err := inboxCall(t, get, ours, permission.SensitivityInternal,
		map[string]any{"messageId": ours.long})
	require.NoError(t, err)
	assert.Equal(t, true, long["bodyTruncated"])
	assert.Less(t, len(long["textBody"].(string)), inboxLongBody)

	_, err = inboxCall(t, get, ours, permission.SensitivityInternal,
		map[string]any{"messageId": other.waiting[0]})
	require.Error(t, err, "another tenant's message is not found")
}
