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

## Next, in order

1. **Browser capability (portal work).** The largest time saving available to agents: appointment
   portals, shipper tender portals, carrier tracking pages, insurer certificate checks. Vendors:
   Browserbase (largest ecosystem), Anchor or Kernel (strongest on signed-in sessions and zero
   data retention), TinyFish (per-step pricing meters cleanly, but no SOC 2 or retention terms
   published yet — diligence first). It cannot reuse web research's safety model, because
   `web_read` is safe only since the model never chooses an address. It needs:
   - an administrator's allow-list of sites per organization; no navigation outside it;
   - credentials held in the vendor's vault, never in the prompt or the turn;
   - reads (look up an appointment, read a tracking page) as `ReturnsExternalContent` tools;
   - every submit, booking or upload as a write with a preview, write coverage and a `Propose`
     ceiling;
   - session recordings kept as the evidence a person approves against.
2. **Exa's newer endpoints on the web research capability:** Monitors (watch a regulation page or
   a carrier's site) and Baselayer business verification through Exa Connect, which fits carrier
   vetting conversations without becoming an integration.
3. **Voice calling (Retell, Vapi)** for check calls and appointment calls, the core of HappyRobot
   and FleetWorks. Every call is a write outside the organization and every transcript is outside
   content; it needs a consent ledger (TCPA treats AI voices as artificial, and some states
   require two-party recording consent) before anything else.
4. **Code sandbox (E2B, Daytona)** for lane, margin and reconciliation analysis over exports, only
   with network egress denied; with egress open it is an exfiltration path.
5. **Bring your own remote MCP server**, last. No TMS researched offers an organization-run
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
