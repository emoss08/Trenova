# Agent extensions

An extension is a capability an organization turns on for its agents only. It is not an
integration: integrations feed the whole system (maps, telematics, email), while an extension
contributes agent tools and nothing else. Each extension runs on the organization's own account
with its vendor, so its credentials, limits and cost belong to the organization.

An extension provides a **capability**, and the capability owns the tools. Web research
(`web_search`, `web_read`, and the web watch tools `watch_web`, `list_web_watches` and
`stop_web_watch`) is the first capability; Exa and Parallel each provide it, and
[agent-web-watching.md](agent-web-watching.md) describes the watches. A vendor that cannot
serve one of its capability's tools lists it in its spec's `Unsupported` (Parallel cannot
watch), and that tool is neither offered nor granted while the vendor serves the capability.
Portal
automation (`list_portal_sites`, `read_portal`, `act_on_portal`) is the second; Browser Use
provides it, and [agent-portal-automation.md](agent-portal-automation.md) describes it. Data
analysis (`analyze_data`) is the third; E2B provides it, and
[agent-data-analysis.md](agent-data-analysis.md) describes it. Agents,
the agent editor, the safety rules and the citation UI name the capability's tools, never a
vendor, so an organization can move from one vendor to another without touching an agent.
Only one extension may serve a capability at a time: turning on a second while the first is on
is refused on `enabled`, and the catalog item says who holds it (`capabilityHeldBy`). Should two
ever be on at once, `agentextension.ActiveFrom` picks the first in `AllTypes` order, so the gate
and every tool call agree on which vendor answers.

## Where things live

| Piece | Location |
|---|---|
| Capabilities and the tools they own, what is active | `services/tms/internal/modules/agent/domain/agentextension/capability.go` |
| Types, availability, each vendor's settings spec and validator | `services/tms/internal/modules/agent/domain/agentextension/` (`spec.go`, `webresearch.go`, one file per vendor) |
| Stored settings and daily usage | `agent_extensions`, `agent_extension_usage_daily` |
| Marketplace, settings, test, metering, web research, portal sites and tasks, analyses | `internal/modules/agent/app/agentextensionservice/` |
| REST API (`/agent-extensions/...`) | `internal/api/handlers/agentextensionhandler/` |
| Vendor connectors | `internal/infrastructure/agentextension/<vendor>connector/` over `shared/<vendor>`, registered with `connectorkit.Provide` |
| Tools | `internal/modules/agent/app/agentquerytoolservice/` (`web_tools.go`, `web_watch_tools.go`, `portal_tools.go`, `analysis_tool.go`) for reads, `internal/modules/agent/app/agenttools/portaltools/` for `act_on_portal`, `agenttools/webwatchtools/` for `watch_web` and `stop_web_watch` |
| Runtime gating and the approval rule | `internal/modules/agent/app/agentruntime/extensions.go` |
| Extension commands (`/browser`, `/search`) | `agentextension/command.go`, `agentruntime/command.go`, `assistantservice/commands.go`, `EXTENSION_COMMANDS` in the client's `composer-commands.ts` and `use-desk-extension-commands.ts` |
| Marketplace UI | `client/apps/web/src/routes/agent-control/_components/extensions/` |

Settings reuse the integration field spec (`domain/configspec`) and the encrypted secret codec
(`services/secretconfig`). Secrets are always bound to the tenant (`PurposeAgentExtensionSecret`).

## How an extension reaches an agent

An extension is **active** for an organization when it is enabled, its required settings are
saved and the plan allows its capability. `AgentExtensionGate.ActiveCapabilities` answers that
as an `agentextension.Active` (capability → serving type and availability), and every place a
tool is offered asks it through `Offers`, `Grants` and `Serving`:

- `permittedTools` drops an inactive extension's tools, so a turn is never offered one and
  `find_tools` never names one.
- With **Every agent** availability, `OpenTurn` adds the tools to what every agent holds (the
  assistant included) and to the prompt's tool list. With **Agents you choose**, an administrator
  adds them to an agent under **Choose tools**.
- The agent editor's catalog (`GET /agent-definitions/tools/`) lists an extension's tools only
  while it is active, marks them `extension`, and flags `grantedToEveryAgent`. Saving an agent
  refuses a newly added tool from an inactive extension; one the agent already held is kept, so
  turning an extension off does not make its agents unsaveable.
- `dispatch` refuses an inactive extension's tool even when the agent lists it, and accepts one
  an extension grants to every agent.

Tool use is authorized on the capability's own resource (`web_research` read; `portal_automation`
read to look and update to act; `data_analysis` read). Managing extensions, and the portal sites and task history,
needs `agent_extension` read and update.

## Extension commands: /browser, /search and /watch

A person can hold a Desk turn to one extension by starting the message with its slash
command: `/browser` works the request on the organization's portal sites (portal automation),
`/search` answers it from the public web and `/watch` sets a web watch on it (both web
research; see [agent-web-watching.md](agent-web-watching.md)). All three are entries in one
registry, and nothing on either side names a particular command outside it.

- **The registry.** `agentextension.AllCommands` lists the commands and `CommandSpecFor` gives
  each one's capability, the tools of which the turn needs at least one to honour it
  (`Requires`: `read_portal` or `act_on_portal` for `/browser`, `web_search` for `/search`,
  `watch_web` for `/watch`), the tools whose call honours it (`Honours`, left empty for any of
  the capability's tools; `/watch` names the watch tools, since a search answers now and the
  person asked to hear later), and the words a person is refused with (`MissingTaskReason`, `DirectedReason`,
  `UnavailableReason`, harvested for translation by `shared/cmd/i18n-extract`).
  `ParseCommand` reads a message the same way for every command: the command alone or
  followed by whitespace, in any case, at the start of the message.
- **Offered only where it can be honoured.** The composer asks
  `GET /assistant/agents/:agentID/commands/` (assistant read), which answers
  `{"commands": [...]}`: the commands whose extension is active, whose required tool the agent
  holds (or the extension gives every agent), and which the person may use.
  `agentruntime.Service.AvailableCommands` decides it from the same held tools, grants and
  permission checks a turn's tool set is built from, reading the organization's extensions once
  for all of them. The client keeps only the names it has a command for
  (`extensionCommandsNamed`), so a newer server never shows a command the client cannot fill.
  Turning an extension on or off invalidates the assistant queries, so a command appears or goes
  at once; with no extension on, no command is listed. A tool its vendor cannot serve is never
  offered (`Spec.Unsupported`), so `/watch` is not listed while Parallel serves web research.
- **Checked again when sent.** `PrepareTurn` refuses a command whose agent cannot honour it, a
  command with nothing after it, and one handed to another agent (`directedAgentId`): a directed
  task runs the other agent without the conversation's own model, so nothing would hold it.
- **Held to its extension.** The turn carries `RunRequest.Command`, kept on `TurnState.Command`
  so the workflow's turn sees it (a state from before carries `browser: true`, read as
  `/browser`). `OpenTurn` loads the command's tools the person may use even on a turn whose
  tools are narrowed, and sends the model the command's directive with the person's words in
  place of the command. A reply that ends the turn without calling a tool that honours the
  command (`CommandSpec.HonouredByCall`), and without asking the person a question, is sent back
  once with the command's correction (`holdToCommand`); a second
  such reply stands, so a request the extension cannot serve still ends in an answer. Another
  extension's tools do not count, nor do its own capability's tools that do not honour it.
- The words are sent as typed, command and all, so the conversation keeps them as written and a
  queued or retried message is still the same request. Typed while a reply is under way, a
  command is queued rather than steered (`isExtensionRequest`): a steer is more words for the
  turn already running, which is not held.

### Adding an extension command

1. Add the command to `AllCommands` and its spec to `commands` in
   `agentextension/command.go`, naming `Honours` when only some of its capability's tools
   carry it out. `TestEveryCommandIsServedByItsCapability` checks that its required and
   honouring tools belong to its capability and that each refusal is written.
2. Add its directive, correction and retry reason to `commandHolds` in
   `agentruntime/command.go`. `TestEveryCommandHasAHold` fails until it has one.
3. Add a case to `commandCases` in the runtime's and the assistant service's `command*_test.go`;
   `TestCommandCasesCoverEveryCommand` fails until both cover it.
4. On the client, add it to `EXTENSION_COMMANDS` in `composer-commands.ts` with
   `extensionCommand({ name, description, slots })`, give it an icon in `COMMAND_ICONS`
   (`desk-slash.tsx`), and add a case to `desk-composer-extension-commands.test.tsx`.
5. Run `task i18n`, translate the new strings, and describe the command in the Desk product
   guide (`docs/product-guide/desk/index.md`, then `guide:generate`).

## Outside content holds every later write for a person

A tool a capability lists in `CapabilitySpec.OutsideContentTools` returns text written outside
the organization, which may have been written to steer an agent. It is declared per tool, not per
capability: `read_portal` returns a portal's page, while `list_portal_sites` returns what
administrators wrote and `act_on_portal` returns only that the task started. A tool whose
result is outside content only when it read something a person brought in declares
`ReadsExternal: marked` instead and names what it read: `analyze_data` taints the run with
each attached file it analyzed and caps nothing, because a report run is the organization's
own data. Once a turn has a
successful result from such a tool, every write it asks for afterwards is capped at `Propose`, whatever the tier,
the ceiling or the private-write rule would allow, and the model is told why.

- The flag lives on the turn (`Turn.external`, `TurnState.ExternalContent`) and travels to the
  tool activity on `DispatchCall.AfterExternalContent`.
- It is set in the loop from the tool's name and whether the call failed, never from the tool's
  output, so a replay decides the same way.
- A delegate starts from the delegating turn's flag (`DelegateCall.AfterExternalContent`) and
  hands its own back (`DelegateRun.ExternalContent`).
- It adds only optional data and no command, so it took no `GetVersion` gate.
- The call's `execute_tool` span carries `trenova.ai.after_external_content`, and a write the
  flag held records `PolicyDefault` as its tier source and `tainted` among what held it, so the
  trace and the audit trail say the rule chose the tier, not a person or earned trust. See
  [ai-tracing.md](ai-tracing.md#execute_tool-tool-internal).
- A turn or run that picks up a portal task's answer from a wait opens with the flag already set
  (`RunRequest.AfterExternalContent`, from `Wait.CarriesOutsideText`), because the answer is
  quoted in its opening note rather than returned by a tool call.
- The web tools and `read_portal` declare `ReadsExternal: always` with the `web` taint source in their
  `ToolPolicy`, so a successful call also adds a `web` mark to the run's taint (see
  [agent-runtime.md](agent-runtime.md#taint-runs-that-have-read-outside-content)). While a turn
  carries the flag, dispatch hands `agenttoolpolicy.Decide` the turn's taint with a `web` mark
  added (`decisionTaint`), so the decision records `tainted` for anything that leaves the
  organization, and then caps what is left at `Propose`. The cap covers web content and the
  partner text and guides named below: tools that read inbound mail, documents or partner files
  already stored in Trenova keep the policy's own, narrower rule, which holds only what leaves the
  organization.

### Partner text and guides

`inspect_x12` and `draft_edi_connection` are not extensions, but they return text written outside
the organization the same way: X12 a person pasted, or an implementation guide a person attached,
both written by a trading partner. A successful call sets the same turn flag, so every later write
in the turn is capped at `Propose` and the model is told it read "content written outside the
organization (a web page, or a trading partner's text or guide it was given)". Both declare
`ReadsExternal: always`, `inspect_x12` with the `edi` source and `draft_edi_connection` with the
`attachment` source, so the run is tainted too. The guide's write, `apply_edi_connection_draft`,
has a `Propose` ceiling of its own and carries the guide's digest; see
[edi-spec-onboarding.md](edi-spec-onboarding.md).

Which tools set the flag is decided from the name alone (`agentruntime.ReadsExternalContent`: an
extension tool its capability lists as returning outside content
(`agentextension.ReturnsExternalContent`), or a name in `outsideTextTools`), because the loop
runs in workflow code and must decide the same way on replay. Each such tool's policy declares
`CapsLaterWrites`, which the safety table and the tool snapshot show, and
`TestEveryToolThatCapsLaterWritesIsOneTheRuntimeCapsOn` (`agentevalgate`) fails when a policy and
the runtime disagree. A tool that reads partner text already stored in Trenova (an inbound file,
a message) taints the run but does not set the flag: the narrower taint rule holds only what
leaves the organization.

## What leaves Trenova

Queries are checked before they are sent (`agentextension.CheckSearchQuery`): record IDs, email
addresses and phone numbers are refused. A portal task's instruction and details are checked by
`CheckPortalInstruction` and `NormalizePortalDetails`: record IDs are refused, because the person
reviewing a portal action sees the details that will be typed into the page. `web_read` opens only a page a search returned: each
result carries a `ref`, an HMAC of the tenant and the URL keyed from the organization's API key,
and a read with any other URL or ref is refused. The model cannot construct an address to send
data to.

## Sources in the reply

The client reads the pages a reply drew on from the saved `web_search` and `web_read` results,
not from the model's prose (`components/assistant/web-sources.ts`). The model is told to cite by
linking a page with its site as the link text and to add no titles, dates or source list of its
own (`webCitationGuidance`), because the client supplies them:

- A link in the answer whose address matches a result is drawn as a citation chip naming the
  site, with the title, publish date, whether it is official and the passage in a hover card.
  `AiMarkdown` asks `MarkdownLinkContext` how to draw a web link, so the renderer stays unaware
  of the assistant. A link to an address no search returned stays a plain link: it is not a
  source.
- The reply's answer carries a closed "N sources" line listing cited pages first, then what
  the search also found. A reply saved as several steps searches in one step and answers in a
  later one, so the pages are gathered across the reply (`replyWebSources`).
- The activity line says "Searched the web" with the query and "Read {site}" for a page, and a
  web call never folds into the lookups in Trenova beside it.

## Metering

Every request reserves one unit of the organization's daily limit before it is sent
(`Reserve`, an upsert that increments only below the limit) and records failure and the
vendor-reported cost after. The connection test counts but is never refused by the limit. The
marketplace card shows today's requests against the limit and the month's cost.

## Connectors

Each vendor is one `agentextensionports.ExtensionConnector`, provided into the
`agent_extension_connectors` fx group by `connectorkit.Provide`. A connector names its `Type`,
runs its own connection test (`Test`, returning the cost and an optional note), and `Classify`s
its errors into `ProviderFailure` kinds (unauthorized, out of credits, rate limited, invalid
request, not found). The service turns a kind into the message the model and the administrator
read, with the vendor's name and the capability's own words (`CapabilitySpec.Unchecked`,
`RequestNoun`), so no vendor's error type reaches the service. `connectorkit.ClassifyHTTP` covers
the HTTP status mapping most vendors need.

A connector for a capability also implements that capability's provider port (web research:
`WebSearchProvider`; portal automation: `BrowserProvider`; data analysis: `SandboxProvider`), receiving the decrypted settings and reading its own vendor fields from
them. `TestEveryExtensionHasAConnectorForItsCapability` fails when a type has no connector or its
connector cannot do what the capability needs.

## Adding a vendor for an existing capability

1. Add the `Type` to `domain/agentextension/enums.go` and a migration widening
   `ck_agent_extensions_type` (the enum constraint test fails otherwise).
2. Add a file for its `Spec`: the capability, its fields (a web research vendor builds them with
   `webResearchFields`, which brings the API key, results per search, daily limit and excluded
   sites; a portal vendor with `portalAutomationFields`), and a `Validate`. Every spec has `apiKey`
   and `dailyRequestLimit`; `DefaultDailyLimit` sets the limit's default when the capability's
   unit is not one request (50 portal tasks a day rather than 1,000 searches).
3. Write its client in `shared/<vendor>` and its connector in
   `internal/infrastructure/agentextension/<vendor>connector`, and register it in that module.
   A vendor that does not report cost computes it from its published prices.
4. Add its catalog `definition` in `agentextensionservice/catalog.go`.
5. Bundle the vendor's mark in `client/packages/shared/src/components/ui/logos/` from Brandfetch
   and register its domain, so the marketplace needs no network call to draw it. Use the SVG
   Brandfetch lists; a vendor it has only as an image (E2B's icon) is bundled as that image,
   unmodified, and drawn with an `img`.
6. Add its strings to the translation catalogs (`task i18n`) and describe it in the AI control
   product guide.

## Adding a capability

A new capability is a new kind of thing agents can do, so it needs more than a vendor:

1. Add it to `capability.go` with the tools it owns, whether they return outside content, and
   the words an agent uses when it is unavailable. A tool belongs to exactly one capability.
2. Give it a plan capability and label in `capabilityDefinitions` (`agentextensionservice`).
3. Declare its provider port next to `WebSearchProvider`, and write its tools in a tool
   package's `ToolProviders()`; `TestEveryExtensionToolIsRegistered` (`agenttoolpolicy`) checks
   each is registered as a read or a write, and that a tool listed as returning outside content
   declares `CapsLaterWrites` and is one the runtime caps on.
   Each tool's `ToolPolicy` declares what it reads and writes, as for any other tool.
4. If its tools act outside Trenova (a browser that submits a form, a phone call), they are
   writes: they need previews, write coverage and a `Propose` ceiling like any other write.
   `act_on_portal` is the example: an `external_recipient` write whose preview is a `Send`, and
   which runs only from an approval.
