# Portal automation

Portal automation is the second agent extension capability (see
[agent-extensions.md](agent-extensions.md)). It lets an agent read and act on the
third-party web portals freight runs on — warehouse dock-appointment portals,
shipper and carrier portals, insurer certificate lookups — through a cloud
browser on the organization's own vendor account. The first vendor is Browser
Use Cloud.

It has three tools:

| Tool | Kind | What it does |
|---|---|---|
| `list_portal_sites` | read | Lists the portal sites the organization set up: what each is for, the addresses it lives on, whether it signs in and whether agents may act on it. Written by administrators, so it is not outside content. |
| `read_portal` | read | Looks something up on one of the organization's portal sites and returns what the page says. Never submits anything. |
| `act_on_portal` | write | Does something on a portal site — books an appointment, uploads a document, accepts a tender — after a person approves it. |

The capability is the plan capability `agent.portal_automation` and the
permission resource `portal_automation` (`read` to look, `update` to act). Sites
and runs are managed under the `agent_extension` resource, like the extension
itself.

## Sites

An agent can only use a portal the organization has set up as a **site**
(`agent_portal_sites`, ids `apst_`): a name, the hostnames it lives on, the https
page to start on (which must be on those hostnames), an optional sign-in, an
optional vendor browser profile that holds a saved sign-in, a note on what it is
for, and whether agents may act on it or only read it. A new site is read-only.

The sign-in's username and password are encrypted with the tenant-bound secret
codec under `PurposeAgentExtensionSecret`, resource kind `agent_portal_site` and
the site's id as subject, so a value copied to another site or tenant does not
decrypt. A sign-in left blank on save keeps the stored one; clearing it removes
every part. The site is never returned with its sign-in, only `hasSignIn` and
`hasAuthenticator`.

Trenova itself cannot be a site. A site whose addresses cover the host of
`app.webBaseUrl` is refused when it is saved, and one saved before the rule is
refused when a task names it: an agent acts in Trenova with its own tools, as the
person it works for and within their permissions, while a browser signed in with
a stored sign-in would act as whoever that sign-in belongs to.

## Sign-in codes

A portal that asks for a second factor at sign-in is handled one of three ways:

- **An authenticator app's code.** The site can hold the portal's authenticator
  key (the setup key it shows when an authenticator app is added, not a six-digit
  code), encrypted beside the username and password. It is accepted as copied:
  grouped with spaces or dashes, in either case, or as an `otpauth://` link, and
  refused unless it is a base32 key of at least 80 bits
  (`agentextension.NormalizeAuthenticatorKey`). It is sent as the vendor secret
  `portal_bu_2fa_code`: Browser Use generates a fresh code from any secret named
  with the `bu_2fa_code` suffix whenever the browser types it, so no code is
  stored or shown, and no model sees the key. A key needs a sign-in beside it;
  clearing the sign-in clears it, and `clearAuthenticator` removes the key alone.
  The approver's preview says the task answers the authenticator code.
- **A code sent by text message or email.** The browser cannot read one, so the
  person relays it. Every task tells the browser to stop at a code it was not
  given and begin its answer with `SIGN-IN CODE NEEDED`
  (`agentextension.PortalSignInCodeMarker`); a run that does, or that did not
  succeed and talks about a sign-in code (`agentextension.AsksForSignInCode`),
  waits for the code when it belongs to a conversation (see
  [Relayed codes](#relayed-codes)). A run with no conversation (a background run)
  ends `Failed` with a message that names the site and says what an
  administrator can change, so the agent says so rather than trying again.
- **A browser profile already signed in.** A profile in the vendor's account
  keeps a signed-in session; a site that names one (`browserProfileId`) starts
  each task already past the sign-in until the portal ends that session.

### Relayed codes

Every task runs in a browser Trenova opens itself (`POST /sessions` with
`keepAlive` and `persistMemory`), not one the vendor closes when the task ends,
so a follow-up task can carry on from the page the first stopped at. Trenova
closes that browser (`StopSession`, the service's `release`) whenever a run ends,
however it ends; the connector closes one it opened for a task the vendor then
refused.

When the browser stops for a code and the run can relay one (`relays`: it has a
conversation, its browser is open, and fewer than `MaxPortalCodeAttempts` (3)
codes were entered), the run goes to `AwaitingCode` instead of ending
(`PortalRun.AwaitCode`). What it did so far (pages, steps, cost) stays on the
run, and `code_prompt` keeps how the portal asked, in the browser's words.

- A read waiting on a code returns with `Status: AwaitingCode` inside its tool
  call: the service starts `PortalRunWorkflow` for it, and `read_portal`
  registers a `PortalRun` wait on the conversation and tells the agent to have
  the person enter the code in the box above their message box, never in the
  conversation, and to end its turn. The answer comes back as a new turn when the
  wait is met.
- An action is already followed by `PortalRunWorkflow`; the poll that finds the
  code request moves the run and returns `CodeRequested`, and the activity calls
  `AnnouncePortalRun`, which republishes the conversation's waits so the box
  appears.

Listing a conversation's waits attaches `signInCode` (`portalRunId`, `prompt`,
`expiresAt`, `attempts`) to an open `PortalRun` wait whose run is waiting. The
Desk draws the code box on that wait's row, and AI control draws it on the run
for someone who can update extensions. The box posts to
`POST /agent-portals/runs/:runID/sign-in-code/`, which takes only the person the
run was asked for, or someone who can update agent extensions (anyone else is
told there is no such run), and only while the run waits. The code is read as a
person types it, spaces and dashes dropped, and refused unless it is 4 to 12
letters and digits (`agentextension.NormalizeSignInCode`). `ProvidePortalCode`
then starts a follow-up task in the same browser (`PortalTaskRequest.SessionID`)
whose text is `agentextension.ContinueWithCode` over the original task, with the
code sent only as the vendor secret `portal_sign_in_code`: it is never written to
the run, never reaches a model, and a detail named like it is refused. The run is
`Running` again with the follow-up's task id, and what the follow-up does adds to
what the run already held (`progressOf`). A portal that refuses the code asks
again (up to three codes in all); after the third, the run fails and says so.

No code within `PortalCodeWaitSeconds` (10 minutes) of the request ends the run:
the poll closes the browser and records it `Failed` with what it did before the
code. `PortalRunWorkflow` does not count time spent waiting on a code toward its
20-minute deadline (`portal-run-code-wait-outside-deadline`), since the code wait
bounds it.

Sites belong to the organization, not to a vendor, so moving to another vendor
keeps them. An agent finds them with `list_portal_sites` and names one by its id
in `read_portal` and `act_on_portal`; a disabled or unknown site is refused with
the sites that exist. A site with tasks recorded against it cannot be removed,
only turned off, so its history keeps its site.

## What keeps it safe

Every portal is a page written outside the organization, which may have been
written to steer an agent. The rules:

- **Navigation is confined to the site.** Each task is sent with the site's
  hostnames as the vendor's navigation allow-list (Browser Use `allowedDomains`)
  and a system prompt extension telling the browser agent to stay on them.
  Afterwards every page the browser visited (each step's URL) is checked against
  the same list (`agentextension.OffSite`); a task that strayed is recorded as
  `Strayed`, its answer is withheld from the model, and the run lists the pages
  it reached. This is why the Browser Use connector uses API V2: V4 limits where
  a credential may be typed, not where the browser may go. The connector
  contract (`BrowserProvider`) takes the allow-list, so a vendor that enforces
  one on a newer API can replace it.
- **Credentials never reach a model.** A sign-in is sent as vendor secrets
  (`portal_username`, `portal_password`); the task names the placeholders, the
  browser agent sees only those, and the vendor types the values into the
  page. Trenova's agent never sees them either: they are decrypted only to start
  the task, and a detail whose name contains a placeholder, the authenticator's
  included, is refused.
- **The browser agent does not see screenshots** unless an administrator turns
  that on, so a value on the page cannot reach the vendor's model as an image.
- **Reading a portal holds every later write for a person.** `read_portal`
  declares `ReadsExternal: always` with the `web` taint source and
  `CapsLaterWrites`, exactly as `web_read` does. Outside content is declared per
  tool (`CapabilitySpec.OutsideContentTools`), so `list_portal_sites` and
  `act_on_portal` do not taint the turn by themselves.
- **Acting on a portal is always a proposal.** `act_on_portal` is an
  `external_recipient` write with a `Propose` ceiling. Its preview is a `Send`
  on the `Portal` message channel: the site's addresses as the recipients, the
  instruction as the subject, and the task exactly as the browser will receive
  it as the body, with whether a sign-in is used. It runs only from a person's
  approval (`ApprovedFromProposal`). A site an administrator marked read-only
  refuses it before anything is filed, and the preview warns instead.
- **What leaves Trenova** is the instruction, the details and the sign-in, sent
  to the vendor and typed into that site. The instruction (8 to 2,000
  characters) and every detail value are refused when they hold a Trenova record
  id; at most 20 details are sent.

## How a task runs

A read must answer within its turn; a write is approved later and may take
minutes, so they run differently. Either way the run is recorded before the
vendor is called, so a task the vendor took is never one Trenova has no record
of, and starting a task is never retried: a request that failed after the vendor
began it would start a second browser doing the same thing.

- **`read_portal`** starts the vendor task and polls it every 3 seconds inside
  the tool call, for at most `ReadBudget` (4 minutes, under the 5-minute tool
  activity). A task still running at the budget is stopped, recorded as
  `TimedOut`, and the agent is told to narrow the question. A read the portal
  stopped for a code the person can relay is followed like an action from there
  (see [Relayed codes](#relayed-codes)).
- **`act_on_portal`** runs on approval. `StartPortalAction` records the run keyed
  by the proposal (a unique idempotency key, so an approval retried finds the
  task it started and does not act twice) and starts the vendor task. The tool
  then registers a `PortalRun` wait on the conversation or background run that
  filed it and starts `PortalRunWorkflow` (`agent-portal-run:<run id>`, system
  queue). The workflow polls every 10 seconds, stops the task at 20 minutes,
  records the outcome and calls `NotifyPortalRun`, which meets the wait with the
  portal's answer quoted as outside text. The work picks up as a new turn
  (`WaitResolved`) or run (trigger `Wait`) that opens with outside content
  already read (`RunRequest.AfterExternalContent`, from
  `Wait.CarriesOutsideText`), so whatever the agent does with the answer is held
  for a person. A wait that cannot be set does not undo the action: the task is
  followed to its end either way and its outcome is recorded.

The wait service does not depend on the extension service (that would close a
cycle through the runtime); the tool registers the wait and the workflow's
activity reports the end.

Every task is a row in `agent_portal_runs` (ids `aprn_`): the site, read or act,
the instruction as sent, the details, whether it signed in, the vendor's task
and session, every page visited and those off the site, the outcome and the
answer, the steps and cost, who asked and from which agent run, conversation and
proposal. AI control lists the history under the extension; a run's recording is
fetched from the vendor when someone opens it, because its address is signed and
short-lived. A run is the `portal_run` record kind: `act_on_portal`'s result names
it, and its record link (`/admin/agent-control?tab=extensions&portalRun=<id>`)
opens the extension serving the capability with that task expanded. Sites are the
`portal_site` kind, so a tool schema takes a site id by its prefix.

## Settings

Browser Use's settings are its API key, the browser agent's model
(`browser-use-2.0` by default), the proxy country (`us`, `ca`, `mx` or none), the
step limit per task (30 by default, 5 to 100), the daily task limit (50 by
default), whether sessions are recorded (on by default) and whether the browser
agent sees screenshots (off by default). The connection test reads the account's
credit balance and concurrency.

## Metering

Each task reserves one unit of the extension's daily limit before it is sent and
records the vendor-reported cost after, against the day the task started, as
every extension does.

## Where things live

| Piece | Location |
|---|---|
| Capability, tools, sites, runs and the task text | `internal/modules/agent/domain/agentextension/` (`capability.go`, `portal.go`, `portaltask.go`, `portalsettings.go`, `browseruse.go`) |
| Stored sites and runs | `agent_portal_sites`, `agent_portal_runs` |
| Site management, reads, starting and finishing actions | `internal/modules/agent/app/agentextensionservice/portal_sites.go`, `portal_tasks.go` |
| The workflow that finishes an action | `internal/modules/agent/jobs/portaljobs/` |
| The wait it sets | `agentwait.KindPortalRun`, `agentwaitservice/portal.go` |
| Vendor connector | `internal/infrastructure/agentextension/browseruseconnector/` over `shared/browseruse` |
| Tools | `agentquerytoolservice/portal_tools.go` (`list_portal_sites`, `read_portal`), `agenttools/portaltools/` (`act_on_portal`) |
| REST API | `/agent-portals/sites/` (list, create, `/:siteID/` update and delete), `/agent-portals/runs/` (list, `/:runID/recording/`, `/:runID/sign-in-code/`) |
| Relayed codes | `agentextension/portalsignin.go`, `agentextensionservice/portal_codes.go`, the code box in `client/apps/web/src/components/assistant/portal-code-form.tsx` |
| AI control | `client/apps/web/src/routes/agent-control/_components/extensions/portal-*.tsx` |
