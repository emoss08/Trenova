# AI extensions roadmap

Researched October 2026. What belongs in the AI extensions marketplace next, what does not, and
why. The mechanics of adding one are in
[agent-extensions.md](../engineering/agent-extensions.md).

## The line between an extension and an integration

An integration feeds the whole system (maps, telematics, fuel, accounting, carrier data). An
extension adds agent tools and nothing else, on the organization's own vendor account. That
line decides several of the obvious candidates:

- **Carrier identity and fraud (Highway, RMIS, MyCarrierPortal), market rates (DAT RateView,
  SONAR TRAC)** are integrations. Their data belongs on carrier and rate records that people,
  rating and compliance read too, not in an agent's turn only. They are the highest-value
  freight data sources researched, so they go on the integrations roadmap.
- **Weather, fuel, FMCSA, routing, SMS** are already integrations, and agents already reach them
  through tools (`vet_carrier`, the monitoring tools). A missing agent tool over an existing
  integration is a tool, not an extension.
- **Document parsing (Reducto, LlamaParse, Mistral OCR)** overlaps the in-house document AI.

## Done

- **Capabilities.** An extension provides a capability and the capability owns the tools, so
  vendors are interchangeable without touching an agent. One vendor serves a capability at a
  time.
- **Parallel** as a second web research vendor: SOC 2 on every API, $1–5 per 1k searches, reads
  JavaScript pages and PDFs.
- **Portal automation** (`list_portal_sites`, `read_portal`, `act_on_portal`) on Browser Use
  Cloud; see [agent-portal-automation.md](../engineering/agent-portal-automation.md). Browser Use
  was chosen because its V2 task API enforces a navigation allow-list (`allowedDomains`), which
  is the rule the capability cannot do without, and types sign-ins from secrets the browser agent
  only sees as placeholders. Administrators set up each site; reads taint the turn; every action
  is a proposal whose preview is the exact task the browser gets, run from the approval and
  followed by a workflow and a wait; every task is recorded with the pages it reached and its
  session recording.
- **Data analysis** (`analyze_data`) on E2B; see
  [agent-data-analysis.md](../engineering/agent-data-analysis.md). Python over the report runs a
  person ran and the files they attached, in a fresh sandbox with all egress denied, stopped when
  it ends; charts are kept with the run and drawn on the Desk. Chosen over Daytona because E2B
  closes egress at sandbox creation and runs a code interpreter that returns charts as images.

- **Web watching** (`watch_web`, `list_web_watches`, `stop_web_watch`) on Exa Monitors; see
  [agent-web-watching.md](../engineering/agent-web-watching.md). A scheduled search that hands
  back only what earlier runs did not find, followed by a workflow that reads the runs from Exa
  (the signed webhook only asks for a check now) and a wait that picks the work up with what was
  found, quoted as outside text. Watches last at most 90 days, are capped per organization, and
  are billed at Exa's published $0.015 a run. Parallel serves web research without watches.

## Next, in order

1. **A second portal vendor** once one enforces where the browser may go, not only where a
   credential may be typed: Anchor or Kernel (signed-in sessions, zero data retention),
   Browserbase (largest ecosystem). `BrowserProvider` already takes the allow-list. TinyFish
   still needs SOC 2 and retention terms before it is considered.
2. **Baselayer business verification through Exa Connect** on the web research capability, which
   fits carrier vetting conversations without becoming an integration.
3. **Voice calling (Retell, Vapi)** for check calls and appointment calls, the core of HappyRobot
   and FleetWorks. Every call is a write outside the organization and every transcript is outside
   content; it needs a consent ledger (TCPA treats AI voices as artificial, and some states
   require two-party recording consent) before anything else.
4. **Bring your own remote MCP server**, last. No TMS researched offers an organization-run
   extension marketplace; Alvys and HappyRobot expose MCP inbound only. When it comes: remote
   servers only, OAuth through client ID metadata documents, an administrator approves each tool,
   each tool definition pinned by hash and disabled on any change until approved again, every
   tool `ReturnsExternalContent`, no write tools, and per-tool daily metering.

## Not recommended

- **Tavily**: changed hands (Nebius, 2026) and its MCP passes the key in the URL.
- **More web search vendors** (Brave, Perplexity, Firecrawl search) beyond Parallel and Exa: each
  is surface area to support for the same capability. Firecrawl is worth a look only if crawl or
  site monitoring is wanted, and then as its own capability.
- **Carrier411**: SOAP-only and available only through certified vendors.
