# Agent extensions

An extension is a capability an organization turns on for its agents only. It is not an
integration: integrations feed the whole system (maps, telematics, email), while an extension
contributes agent tools and nothing else. Each extension runs on the organization's own account
with its vendor, so its credentials, limits and cost belong to the organization.

The first extension is web search through Exa (`web_search`, `web_read`).

## Where things live

| Piece | Location |
|---|---|
| Types, availability, settings spec, tool ownership | `services/tms/internal/core/domain/agentextension/` |
| Stored settings and daily usage | `agent_extensions`, `agent_extension_usage_daily` |
| Marketplace, settings, test, metering, web research | `internal/core/services/agentextensionservice/` |
| REST API (`/agent-extensions/...`) | `internal/api/handlers/agentextensionhandler/` |
| Vendor adapter | `internal/infrastructure/agentextension/exaconnector/` over `shared/exa` |
| Tools | `internal/core/services/agentquerytoolservice/web_tools.go` |
| Runtime gating and the approval rule | `internal/core/services/agentruntime/extensions.go` |
| Marketplace UI | `client/apps/web/src/routes/agent-control/_components/extensions/` |

Settings reuse the integration field spec (`domain/configspec`) and the encrypted secret codec
(`services/secretconfig`). Secrets are always bound to the tenant (`PurposeAgentExtensionSecret`).

## How an extension reaches an agent

An extension is **active** for an organization when it is enabled and its required settings are
saved. `AgentExtensionGate.ActiveExtensions` answers that, and every place a tool is offered asks
it:

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

Tool use is authorized on its own resource (`web_research`, read). Managing extensions needs
`agent_extension` read and update.

## Outside content holds every later write for a person

An extension whose spec sets `ReturnsExternalContent` returns text written outside the
organization, which may have been written to steer an agent. Once a turn has a successful result
from such a tool, every write it asks for afterwards is capped at `Propose`, whatever the tier,
the ceiling or the private-write rule would allow, and the model is told why.

- The flag lives on the turn (`Turn.external`, `TurnState.ExternalContent`) and travels to the
  tool activity on `DispatchCall.AfterExternalContent`.
- It is set in the loop from the tool's name and whether the call failed, never from the tool's
  output, so a replay decides the same way.
- A delegate starts from the delegating turn's flag (`DelegateCall.AfterExternalContent`) and
  hands its own back (`DelegateRun.ExternalContent`).
- It adds only optional data and no command, so it took no `GetVersion` gate.

## What leaves Trenova

Queries are checked before they are sent (`agentextension.CheckSearchQuery`): record IDs, email
addresses and phone numbers are refused. `web_read` opens only a page a search returned: each
result carries a `ref`, an HMAC of the tenant and the URL keyed from the organization's API key,
and a read with any other URL or ref is refused. The model cannot construct an address to send
data to.

## Metering

Every request reserves one unit of the organization's daily limit before it is sent
(`Reserve`, an upsert that increments only below the limit) and records failure and the
vendor-reported cost after. The connection test counts but is never refused by the limit. The
marketplace card shows today's requests against the limit and the month's cost.

## Adding an extension

1. Add the `Type` to `domain/agentextension/enums.go` and a migration widening
   `ck_agent_extensions_type` (the enum constraint test fails otherwise).
2. Add its `Spec` in `spec.go`: fields, the tools it owns, whether it returns outside content,
   and a settings parser. Include `dailyRequestLimit` so metering applies.
3. Add its catalog `definition` in `agentextensionservice/catalog.go` and its tools to a tool
   package's `ToolProviders()`; `TestEveryExtensionToolIsRegistered` checks the two agree.
4. Bundle the vendor's mark in `client/packages/shared/src/components/ui/logos/` and register its
   domain, so the marketplace needs no network call to draw it.
5. Add its strings to the translation catalogs (`task i18n`) and describe it in the AI control
   product guide.
