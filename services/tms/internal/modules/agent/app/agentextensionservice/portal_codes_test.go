package agentextensionservice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/portaljobs/portalcontract"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/client"
)

const textedCodePrompt = "SIGN-IN CODE NEEDED: the portal texted a six-digit code to the " +
	"phone ending 42."

type recordingWorkflows struct {
	kernelports.WorkflowStarter

	mu  sync.Mutex
	ids []string
}

func (r *recordingWorkflows) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	_ ...any,
) (client.WorkflowRun, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.ids = append(r.ids, options.ID)

	return nil, nil //nolint:nilnil // the run handle is never read
}

func askedForCode(answer string) *agentextensionports.PortalTaskState {
	succeeded := false

	return &agentextensionports.PortalTaskState{
		Phase:       agentextensionports.PortalTaskFinished,
		Succeeded:   &succeeded,
		Answer:      answer,
		VisitedURLs: []string{"https://portal.acme.com/login"},
		Steps:       2,
		CostUSD:     decimal.RequireFromString("0.03"),
	}
}

func (f *fixture) person() agentextensionports.PortalTaskOwner {
	return agentextensionports.PortalTaskOwner{
		Actor: &permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    f.tenant.UserID,
			UserID:         f.tenant.UserID,
			OrganizationID: f.tenant.OrgID,
			BusinessUnitID: f.tenant.BuID,
		},
		ThreadID: pulid.MustNew("athr_"),
	}
}

// readAwaitingCode reads a portal that stops for a code the person was texted,
// from a conversation, and returns the outcome the agent sees.
func (f *fixture) readAwaitingCode(t *testing.T) *agentextensionports.PortalReadOutcome {
	t.Helper()

	f.enableBrowser(t)
	f.svc.workflows = f.workflows
	site := f.addSite(t, false)
	f.browser.finishWith(askedForCode(textedCodePrompt))

	outcome, err := f.svc.ReadPortal(t.Context(), &agentextensionports.PortalReadRequest{
		TenantInfo: f.tenant,
		SiteID:     site.ID,
		Question:   "Which dock appointments are open on October 20?",
		Owner:      f.person(),
	})
	require.NoError(t, err)

	return outcome
}

func newCodeFixture(t *testing.T) *fixture {
	t.Helper()

	f := newFixture(t)
	f.workflows = &recordingWorkflows{}

	return f
}

func (f *fixture) enterCode(
	t *testing.T,
	runID pulid.ID,
	code string,
) (*agentextensionports.PortalRunView, error) {
	t.Helper()

	return f.svc.ProvidePortalCode(t.Context(), &agentextensionports.ProvidePortalCodeRequest{
		TenantInfo: f.tenant,
		RunID:      runID,
		Code:       code,
	})
}

func TestAReadStoppedForATextedCodeWaitsWithItsBrowserOpen(t *testing.T) {
	t.Parallel()

	f := newCodeFixture(t)
	outcome := f.readAwaitingCode(t)

	assert.Equal(t, agentextension.PortalRunStatusAwaitingCode, outcome.Status)
	assert.Equal(t, textedCodePrompt, outcome.CodePrompt)
	assert.Empty(t, outcome.Answer)

	run := f.runs.only(t)
	assert.Equal(t, agentextension.PortalRunStatusAwaitingCode, run.Status)
	assert.Equal(t, textedCodePrompt, run.CodePrompt)
	require.NotNil(t, run.CodeRequestedAt)
	assert.Equal(t, testNow, *run.CodeRequestedAt)
	assert.Nil(t, run.FinishedAt)
	assert.Equal(t, 2, run.Steps)
	assert.Empty(t, f.browser.closed, "the browser stays on the code prompt")
	assert.Empty(t, f.browser.stopped)
	assert.Equal(t, []string{portalcontract.WorkflowIDFor(run.ID)}, f.workflows.ids,
		"the read is followed from the code prompt to its end")
	assert.Zero(t, f.usage.failures, "the read is settled when it ends, not while it waits")
	assert.True(t, f.usage.cost.IsZero())
}

func TestARelayedCodeContinuesTheTaskInTheSameBrowser(t *testing.T) {
	t.Parallel()

	f := newCodeFixture(t)
	outcome := f.readAwaitingCode(t)
	waiting := f.runs.only(t)

	view, err := f.enterCode(t, outcome.RunID, " 123-456 ")
	require.NoError(t, err)
	assert.Equal(t, agentextension.PortalRunStatusRunning, view.Status)
	assert.Equal(t, 1, view.CodeAttempts)
	assert.Nil(t, view.CodeExpiresAt)

	require.Len(t, f.browser.started, 2)
	followUp := f.browser.started[1]
	assert.Equal(t, "session-1", followUp.SessionID)
	assert.Empty(t, followUp.StartURL)
	assert.Equal(t, agentextension.ContinueWithCode(waiting.Instruction), followUp.Instruction)
	assert.Equal(t, "123456", followUp.Secrets[agentextension.PortalSecretSignInCode])
	assert.Equal(t, "s3cret-pass", followUp.Secrets[agentextension.PortalSecretPassword])
	assert.Equal(t, []string{"portal.acme.com"}, followUp.AllowedDomains)
	assert.NotContains(t, followUp.Instruction, "123456")

	stored := f.runs.only(t)
	assert.Equal(t, "task-2", stored.VendorTaskID)
	assert.NotContains(t, fmt.Sprintf("%+v", *stored), "123456",
		"the code is sent as a secret and never recorded")

	f.browser.finishWith(finished("Open slots: 8:00 AM.", "https://portal.acme.com/appointments"))
	poll, err := f.svc.PollPortalRun(t.Context(), f.tenant, outcome.RunID)
	require.NoError(t, err)
	assert.False(t, poll.CodeRequested)
	done := poll.Run
	assert.Equal(t, agentextension.PortalRunStatusSucceeded, done.Status)
	assert.Equal(t, "Open slots: 8:00 AM.", done.Answer)
	assert.Equal(t, []string{
		"https://portal.acme.com/login",
		"https://portal.acme.com/appointments",
	}, done.VisitedURLs, "the pages before the code are kept")
	assert.Equal(t, 3, done.Steps)
	assert.True(t, decimal.RequireFromString("0.07").Equal(done.CostUSD))
	assert.Equal(t, []string{"session-1"}, f.browser.closed)
}

func TestAPollAnnouncesACodeRequestOnce(t *testing.T) {
	t.Parallel()

	f := newCodeFixture(t)
	f.enableBrowser(t)
	site := f.addSite(t, true)
	started, err := f.svc.StartPortalAction(t.Context(), &agentextensionports.PortalActionRequest{
		TenantInfo:     f.tenant,
		SiteID:         site.ID,
		Instruction:    "Book the 8:00 AM dock appointment on October 20",
		IdempotencyKey: "apr_code",
		Owner:          f.person(),
	})
	require.NoError(t, err)

	f.browser.finishWith(askedForCode(textedCodePrompt))
	first, err := f.svc.PollPortalRun(t.Context(), f.tenant, started.RunID)
	require.NoError(t, err)
	assert.True(t, first.CodeRequested)
	assert.Equal(t, agentextension.PortalRunStatusAwaitingCode, first.Run.Status)

	second, err := f.svc.PollPortalRun(t.Context(), f.tenant, started.RunID)
	require.NoError(t, err)
	assert.False(t, second.CodeRequested)
	assert.Equal(t, agentextension.PortalRunStatusAwaitingCode, second.Run.Status)
	assert.Empty(t, f.browser.closed)
}

func TestACodeNotEnteredInTimeClosesTheBrowser(t *testing.T) {
	t.Parallel()

	f := newCodeFixture(t)
	outcome := f.readAwaitingCode(t)
	f.svc.now = func() int64 { return testNow + agentextension.PortalCodeWaitSeconds }

	_, err := f.enterCode(t, outcome.RunID, "123456")
	require.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "The time to enter this code has run out")

	poll, err := f.svc.PollPortalRun(t.Context(), f.tenant, outcome.RunID)
	require.NoError(t, err)
	assert.Equal(t, agentextension.PortalRunStatusFailed, poll.Run.Status)
	assert.Equal(t, portalCodeExpiredMessage, poll.Run.Failure)
	assert.Equal(t, 2, poll.Run.Steps, "what the task did before the code is kept")
	assert.Equal(t, []string{"session-1"}, f.browser.closed)
	assert.Len(t, f.browser.started, 1)

	_, err = f.enterCode(t, outcome.RunID, "123456")
	require.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "not waiting for a sign-in code")
}

func TestOnlyThePersonTheTaskWasForEntersItsCode(t *testing.T) {
	t.Parallel()

	f := newCodeFixture(t)
	outcome := f.readAwaitingCode(t)

	someoneElse := f.tenant
	someoneElse.UserID = pulid.MustNew("usr_")
	_, err := f.svc.ProvidePortalCode(t.Context(), &agentextensionports.ProvidePortalCodeRequest{
		TenantInfo: someoneElse,
		RunID:      outcome.RunID,
		Code:       "123456",
	})
	require.True(t, errortypes.IsNotFoundError(err))
	assert.Len(t, f.browser.started, 1)

	view, err := f.svc.ProvidePortalCode(t.Context(), &agentextensionports.ProvidePortalCodeRequest{
		TenantInfo:  someoneElse,
		RunID:       outcome.RunID,
		Code:        "123456",
		Administers: true,
	})
	require.NoError(t, err)
	assert.Equal(t, agentextension.PortalRunStatusRunning, view.Status)
}

func TestACodeThatIsNotLettersAndDigitsIsRefused(t *testing.T) {
	t.Parallel()

	f := newCodeFixture(t)
	outcome := f.readAwaitingCode(t)

	for _, code := range []string{"12", "12!456", "1234567890123", "123456; drop"} {
		_, err := f.enterCode(t, outcome.RunID, code)
		var fieldErr *errortypes.Error
		require.ErrorAs(t, err, &fieldErr, code)
		assert.Equal(t, "code", fieldErr.Field, code)
	}
	assert.Len(t, f.browser.started, 1)
	assert.Equal(t, agentextension.PortalRunStatusAwaitingCode, f.runs.only(t).Status)
}

func TestAPortalThatKeepsRefusingCodesStopsAfterThree(t *testing.T) {
	t.Parallel()

	f := newCodeFixture(t)
	outcome := f.readAwaitingCode(t)

	for attempt := 1; attempt <= agentextension.MaxPortalCodeAttempts; attempt++ {
		_, err := f.enterCode(t, outcome.RunID, "123456")
		require.NoError(t, err, "attempt %d", attempt)
		f.browser.finishWith(askedForCode(
			"SIGN-IN CODE NEEDED: the portal says the code is wrong."))
		poll, err := f.svc.PollPortalRun(t.Context(), f.tenant, outcome.RunID)
		require.NoError(t, err)
		if attempt < agentextension.MaxPortalCodeAttempts {
			assert.True(t, poll.CodeRequested, "attempt %d", attempt)
			continue
		}
		assert.False(t, poll.CodeRequested)
		assert.Equal(t, agentextension.PortalRunStatusFailed, poll.Run.Status)
		assert.Contains(t, poll.Run.Failure,
			"Acme DC still asked for a sign-in code after 3 codes were entered")
	}
	assert.Equal(t, []string{"session-1"}, f.browser.closed)
}

func TestACodeTheVendorWouldNotTakeEndsTheRun(t *testing.T) {
	t.Parallel()

	f := newCodeFixture(t)
	outcome := f.readAwaitingCode(t)
	f.browser.startErr = errors.New("session gone")

	_, err := f.enterCode(t, outcome.RunID, "123456")
	require.Error(t, err)

	run := f.runs.only(t)
	assert.Equal(t, agentextension.PortalRunStatusFailed, run.Status)
	assert.Equal(t, 2, run.Steps)
	assert.Equal(t, []string{"session-1"}, f.browser.closed)
}
