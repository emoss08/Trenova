# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

The web app (`client/apps/web`) and the driver portal (`client/apps/dash`, an installable PWA) are both web. The Windows scanning and printing companion (`desktop/capture`) is a separate tray app.

## Users

Trenova is for small and mid-sized over-the-road carriers in the United States.

**Primary user: the dispatcher.** They run live freight under time pressure: assigning drivers and equipment, sequencing stops, and handling exceptions as they happen. When roles' needs conflict in a design decision, the dispatcher's working day wins.

**Other audiences**, served without overriding the dispatcher:

- Billers, collectors, accounts payable, accounting and payroll clerks, who carry work from delivery through invoice, settlement and close.
- Customer service, safety, fleet, pricing and carrier managers, who each own a set of records.
- Managers and owners, who work the approval queue and watch the operation.
- Drivers, who use the driver portal for loads, hours of service, pay and settlements. Their situation is the cab and the dock, on a phone, often in glare, with gloves, on poor signal and with one hand.

## Product Purpose

Trenova is a transportation management system covering the full order-to-cash cycle: shipment entry, dispatch, driver assignment, rating, billing, settlement, accounting and compliance.

It exists because enterprise TMS platforms often cost more than $50,000 a year, which puts them out of reach of the carriers that make up most of the industry. It closes that gap with a flat-priced, self-hostable, source-available product.

Success means a dispatcher can run the day from Trenova, with agents taking on work and nothing changing in the operation without a person's say-so.

## Positioning

**AI agents do the work, and a person approves every change.** This is the lead position.

Agents read the operation, draft the work and propose changes to records. A proposal shows what it would do, worked out by the same code that would make the change and read from the records as they are now, along with a dry run and whether it can be undone. It then waits on a person. Nothing an agent proposes changes the operation until someone approves it.

Flat pricing, self-hosting, public source and one system across order-to-cash are real and support this position. They do not lead it.

## Operating Context

- **Workflows:** shipment lifecycle and stop sequencing, exception management, rating and accessorials, billing readiness and invoicing, cash application, general ledger and close, driver and carrier settlement, detention, EDI partner work (204, 210, 214, 990, 997), FMCSA-oriented compliance.
- **The Desk:** where people talk to agents about the work in front of them. Conversations sit beside a workspace of artifacts (tables, drafts, documents, plans, record cards). Today opens the day, Watchtower and Decisions sit beside it, and a conversation about a shipment, invoice or dispute becomes a case.
- **Decisions:** the queue of proposed changes waiting on a person, worked by dispatchers, billing staff and managers, with keyboard shortcuts to approve and decline.
- **Paper:** BOLs, PODs, rate confirmations, invoices and weight tickets are extracted and generated. Trenova Capture scans and prints from Windows computers in the office.
- **Outside tools:** Samsara telematics, PC*MILER and AS2 trading partners.
- **Public pages:** tender offers and carrier tracking are reached without an account.

## Capabilities and Constraints

- **Status:** pre-release and not yet suitable for production use. The waitlist is open at trenova.app.
- **Editions:** the public repository is the self-hostable product. Trenova Cloud adds its own UI (self-serve signup, trial banner, plan page) through an overlay built from a private repository. How far the two editions must stay equal is not a design commitment recorded here.
- **Licence:** source-available under the Functional Source License (FSL-1.1-ALv2).
- **Languages:** English is the source. Spanish targets Latin American usage, for US cross-border freight. Traditional Chinese follows Taiwan usage and is not a character conversion of Simplified Chinese. Translated freight terms are fixed in `i18n/glossary.json`.
- **Terminology:** Trenova is the product, Nova is its AI, the Desk is where agents are worked with, and Decisions is the approval queue. Dash is the code name of the driver portal.
- **Scale:** the product is built for hundreds of thousands of users across thousands of tenants, so a surface cannot assume a small tenant's data.
- **Permissions:** what a person sees depends on their role and on their organization's settings, so a surface cannot assume every action or module is present.

## Brand Commitments

- The name is Trenova. The logo is at `.github/logo.webp`.
- Labels are sentence case.
- Product language is plain and specific, as in `docs/product-guide/`. It says what happened and what to do next.
- Agents never show people code or how Trenova is built.
- The visual system is recorded in `docs/engineering/design-system.md` and `client/packages/shared/src/styles/tokens.css`, not here.

## Evidence on Hand

- Capability and task descriptions: `README.md` and `docs/product-guide/`.
- Public site: trenova.app, with features, pricing, self-hosting and waitlist pages. Its source is not in this repository.
- Community: GitHub Discussions and Discord.
- The repository holds no customer names, testimonials, case studies, press, usage figures or benchmarks. Trenova is pre-release, so none may be implied or invented.
- The $50,000 figure is the README's statement about enterprise TMS cost. Treat it as a general claim, not a measured comparison.

## Product Principles

1. **The dispatcher's day decides.** When roles compete for a surface, design for the person running live freight under time pressure, then serve the others without slowing that person down.
2. **People approve, agents propose.** Any change an agent wants to make is shown in full before it runs, and the person can see what it does, whether it would go through and whether it can be undone.
3. **Show where a fact came from.** Agent output is tied to the records and steps behind it, so a person can check it instead of trusting it.
4. **Say the true state.** A value that moved, a change that would be refused and a step that failed are all stated plainly, including how many records are affected and why.
5. **Work where the work happens.** A dispatcher at a desk with several screens and a driver at a dock on a phone are both first-class, and neither gets a reduced version of the other's product.

## Accessibility & Inclusion

- **Standard:** WCAG 2.2 AA across the web app and the driver portal. This is the target; conformance has not been measured.
- **Keyboard:** high-frequency work must be fully operable from the keyboard.
- **Motion:** `prefers-reduced-motion` is honoured throughout.
- **Drivers:** the driver portal must hold up on a phone in the cab or at a dock: readable in glare, targets usable with gloves and one hand, and work that survives a poor connection.
- **Language:** every surface must work in all four locales, including longer Spanish strings and CJK text.
