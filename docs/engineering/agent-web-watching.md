# Web watching

An agent keeps watching the web for a person: "tell me when FMCSA finalizes the broker
transparency rule", "let me know if Acme Freight is in the news". The organization's web
research vendor runs the search on a schedule and hands back only pages earlier runs did not
find, and the conversation (or background agent) that asked picks the work up with them.
Exa is the vendor that can do it, through its Monitors API; Parallel serves web research
without it.

## Tools

| Tool | Kind | What it does |
|---|---|---|
| `watch_web` | action, `AutoExecute`, `web_research:read` | Sets a watch: `purpose` (what the person wants to hear about), `query` (the search run each time), optional `sites` (hostnames, at most 20), `cadence` (`SixHours`, `Daily`, `Weekly`; daily by default) and `days` (1 to 90; 30 by default) |
| `list_web_watches` | query | The watches set for the person (or, from a background run, by its agent), active ones unless asked for all, each marked when it came from this conversation |
| `stop_web_watch` | action, `AutoExecute`, `web_research:read` | Stops a watch the person no longer needs |

A person can also start a Desk message with `/watch` and what to keep an eye on, which holds
the turn to setting a watch: the extension command registry lists it only where `watch_web` can
be used, and a reply that ends without a watch tool (a search alone does not count) is sent
back once (see "Extension commands" in [agent-extensions.md](agent-extensions.md)).

The tools belong to the web research capability (`agentextension.WebWatchTools`), so they
reach an agent exactly as `web_search` does. A vendor that cannot watch lists them in its
spec's `Unsupported`; `Active.Serving` and `Active.Grants` leave them out while that vendor
serves the capability, and the catalog lists only the tools the vendor serves. A watch is web
research that goes on: it changes no record, reaches nobody and spends on the organization's
own account, so it runs on its own, and like every write it is proposed after the turn read
outside content. A task another agent handed over cannot set one
(`webwatchtools.ErrDelegatedWatch`).

The preview (`PlanWebWatch`, which never reaches the vendor) says what will be searched,
where, how often and until when, and the cost at Exa's published price
(`ExaMonitorRunCostUSD`, $0.015 a run, times `WebWatchChecks`). The same plan refuses a watch
before it is set when Trenova has no public https address (`app.webBaseUrl`: the vendor
reports to it, and a loopback, private, `.local` or `.internal` host is refused), when the
search would send a record id, an email address or a phone number (`CheckSearchQuery`, as for
`web_search`), and when the organization already has as many watches running as Exa's
"Web watches at once" setting allows (10 by default, 1 to 100).

## Setting a watch

`WatchWeb` records the watch (`agent_web_watches`, ids `awwt_`) before the vendor is called,
so a monitor Exa made is never one Trenova has no record of. The monitor is created once and
never retried (the connector's client has no retry): a request that failed after Exa took it
would make a second monitor billing for the same search. It carries the query, the sites (a
watch that names its sites sends none of the organization's excluded sites; one over the whole
web sends them all), the cadence as Exa's period, the watch id as metadata and a webhook
address with the tenant in its path
(`/api/v1/webhooks/agent-extensions/web-watches/:orgID/:buID/:watchID/`). Exa returns the
webhook's signing secret, which is stored encrypted on the watch (`secretconfig`, bound to the
watch). A watch the vendor refused ends `Failed` at once; one the vendor took but Trenova could
not keep track of has its monitor removed and ends `Failed`.

`watch_web` then parks the work on it: `RegisterWebWatchWait` sets a `WebWatch` wait (see
[agent-waits.md](agent-waits.md)) on the conversation or background run, lasting as long as
the watch plus an hour for its last check (`agentwait.MaxWebWatchLifetimeSeconds`, up to 90
days). A watch whose work could not be parked is stopped at once: what it found would reach no
one.

## Following a watch

`WebWatchWorkflow` (`agent-web-watch:<watch id>`, system queue) checks the watch every half
hour, or at once on the `check` signal, until a check says it ended, continuing as new after a
run of checks so its history stays small. A check that keeps failing waits for the next one; a
watch ends only when its time runs out, it is stopped, or Exa can no longer run it.

Each check (`CheckWebWatch`):

1. Ends a watch whose time ran out, removing its monitor. A watch whose extension can no longer
   be read ends saying where its monitor may still be running.
2. Ends `Failed` a monitor Exa turned off on its own (after its key kept failing), removing it.
3. Reads the latest runs (`WatchRuns`, told when the last counted run was made so a run already
   counted is not fetched in full again), oldest first, stopping at the first that has not
   settled so it is counted once it has. Each settled run is billed to the extension's usage
   for the day it settled; a failed run's reason is kept on the watch in words an administrator
   can act on (a rejected key, no credits, rate limits, a site Exa cannot search). A completed
   run with pages at an absolute http or https address becomes a finding
   (`agent_web_watch_findings`, `awwf_`, unique per watch and vendor run, so a run is recorded
   once).
4. Returns the findings not yet handed on. The activity meets the waits with them
   (`NotifyWebWatch`), then marks them delivered.

`NotifyWebWatch` meets each wait set before the newest finding with what the watch found:
the vendor's summary and pages, quoted and marked as written outside the organization, newest
first, within a wait's outcome (`webWatchOutcome`). The work picks up as a new turn or run that
opens with outside content already read, so whatever it then asks to write is held for a
person. It then sets the next wait for the same work while the watch goes on. A wait as new as
the newest finding is the one set after an earlier delivery, so a delivery tried again neither
picks the work up twice nor sets two waits.

Exa's webhook only speeds this up. `ReceiveWebWatchDelivery` reads the watch under the tenant
in its path, checks the delivery's `Exa-Signature` (`t=<unix>,v1=<hex>`, HMAC-SHA256 of
`t.body`, five minutes either way) against the watch's own secret, and signals the workflow to
check now. What the delivery says is never used: the check reads the runs from Exa. The route
is public, rate-limited by client address, and refuses a body over its limit; a bad signature
is 401 and an unknown watch 404.

## Stopping and ending

`stop_web_watch`, **Stop watching** on the Desk's wait, and **Stop watching** under the
extension in AI control (`POST /agent-web-watches/:watchID/stop/`) remove the monitor, record
the watch `Stopped` with who stopped it, close its waits quietly (`EndWebWatchWaits`) and
signal the workflow, which then ends. Only the person a watch was set for, the agent whose
background work set one for no person, or someone who can update agent extensions may stop it;
anyone else is told it does not exist. A monitor Exa will not remove leaves the watch running
rather than recorded as stopped. A `WebWatch` wait cannot be cancelled on its own
(`cancel_wait` and the wait's cancel refuse it): it ends with its watch.

A watch that fails meets its waits with the reason, so the person hears. One whose time ran out
lets its wait run out an hour later, and the work hears the watch ended.

## Where people see it

- The Desk lists the wait above the message box ("A web watch", how often it looks and until
  when, how many findings so far) from `Wait.WebWatch`, attached when the conversation's waits
  are listed.
- AI control lists every watch under the extension that offers watches (`webWatchServer`, so
  the history stays with Exa while another vendor serves web research): what it searches for,
  where, how often, until when, its checks and cost, and each finding with its pages, opened
  from the `web_watch` record link (`?tab=extensions&webWatch=<id>`).

## Where things live

| Piece | Location |
|---|---|
| Watch, finding, cadence, bounds | `internal/modules/agent/domain/agentextension/webwatch.go` |
| Ports | `ports/agentextensionports/webwatch.go` |
| Service | `app/agentextensionservice/web_watches.go` |
| Repositories | `adapters/postgres/agentextensionrepository/webwatch.go` |
| Workflow and activity | `jobs/webwatchjobs/`, `webwatchcontract` |
| Waits | `agentwait.KindWebWatch`, `app/agentwaitservice/webwatch.go` |
| Tools | `app/agenttools/webwatchtools/` (`watch_web`, `stop_web_watch`), `app/agentquerytoolservice/web_watch_tools.go` (`list_web_watches`) |
| Vendor | `shared/exa/monitors.go`, `internal/infrastructure/agentextension/exaconnector/monitors.go` |
| REST | `GET /agent-web-watches/`, `GET /agent-web-watches/:watchID/findings/`, `POST /agent-web-watches/:watchID/stop/`, the public webhook |
| Client | `routes/agent-control/_components/extensions/web-watches.tsx`, `components/desk-chat/composer/desk-waits.tsx`, `lib/web-watch.ts` |
