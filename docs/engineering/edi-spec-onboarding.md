# Building an EDI connection from a partner's spec

A trading partner sends an implementation guide (a PDF, a DOCX, or a text file of sample X12).
An agent on the Desk reads it, drafts the whole connection, and proposes it. A person reviews the
draft and approves it. Nothing is written before that approval. Read this before changing
`ediservice/onboarding*.go`, the `draft_edi_connection` or `apply_edi_connection_draft` tools, the
`edi_connection_draft` Desk card, or the communication profile validator's credential rules.

## The shape of it

```
person attaches spec ─▶ document intelligence (pages, OCR) ─▶ document_content_pages
                                                                    │
agent reads pages (read_attachment) and cites every value ─────────┤
                                                                    ▼
draft_edi_connection(documentId, cited values, requirements, mappings, gaps)
   └─ ediservice.PlanConnectionDraft
        ├─ verify every citation against the stored page text
        ├─ hold back low-confidence, unverified and conflicting values
        ├─ match an existing partner (or plan a new one, never a duplicate)
        ├─ plan partner, communication profile, document profiles, control numbers, mappings
        ├─ pick the closest template per set and list every unmet requirement
        ├─ run the spec's sample X12 through the inspector and the inbound parser
        ├─ validate every planned record with the validators the setup screens use
        └─ diff against what the partner has today (a change set on a re-run)
   ▼
edi_connection_draft Desk card (the review)
   ▼
apply_edi_connection_draft (always proposed) ─▶ Preview = the same plan, recomputed now
   ▼ person approves (approval carries the preview digest; the plan carries the document digest)
Execute ─▶ one transaction through the ediservice create and update paths
```

## Tools

| Tool | Kind | Holds it | What it does |
|---|---|---|---|
| `draft_edi_connection` | query, EDI read | EDI desk | Builds the plan and returns the review. Writes nothing. Reads outside content, so it caps every later write in the turn at `Propose` (`CapsLaterWrites`, `outsideTextTools`). |
| `apply_edi_connection_draft` | action, EDI create and update | EDI desk | The same input. Its `Preview` recomputes the plan from the document and the configuration as they are at decision time. Its ceiling is `Propose`: a person always decides. |

Both tools take the same input. Reading the document is document intelligence's job, through the
pipeline every attachment goes through (`DocumentContentService.GetContent`). No second extractor
exists. The agent reads the pages with `read_attachment` and passes what it found, each value with
its citation:

- `documentId`: a file the person attached to this conversation, or a document they may read on a
  record.
- `partnerId` (optional): the partner to update. Without it, the planner matches one.
- `values[]`: `{field, value, page, quote, confidence}`. `field` names a slot the planner knows
  (`partner.name`, `isa.receiverId`, `204.inbound.ack.type`, `as2.partnerAs2Id`, ...).
  Secret slots do not exist.
- `requirements[]`: the segment and element usage the spec states, per set and direction, each cited.
- `mappings[]`: the partner's codes the spec lists, with the Trenova record each should map to, each
  cited.
- `gaps[]`: what the spec asks for that Trenova does not support, or that the spec leaves
  ambiguous, each with a page and section.
- `resolutions[]`: a person's answer to a held-back value. Each one is either one of the cited
  candidates or a value the person gave in the conversation, and the review labels it that way.

## No draft table

A draft is a pure function of three things: the stored page text, the cited input, and the
configuration as it is now. The proposal row is the durable record. It holds:

- the input;
- the document digest;
- the approved preview.

The tool result holds the review the person read. A separate drafts table would be a second copy
that could go stale against the configuration it describes. Each re-plan reads the configuration
again instead.

## Every value has a source

- **Citations are checked.** A quote must appear on the page it cites, with whitespace and case
  normalized. The value must also appear in that quote, or be a code a cited phrase spells out
  (for example "Mutually Defined" for `ZZ`). A citation that fails either check makes the value
  **unverified**. An unverified value is never used.
- **Confidence.** A value whose confidence is below `0.8` is **held back**. So is a field that two
  citations give different values for, or a value the spec's own sample X12 contradicts. A
  held-back value is left unset, or left at the partner's current value on a re-run. The review
  lists it under *Needs your decision*, showing each candidate with its page and quote.
- **Samples are evidence.** A sample's ISA/GS envelope is parsed rather than read by a model, so a
  value taken from it has confidence 1 and cites the sample's page and ISA segment.
- **Quotes are redacted** before they are stored or shown. Anything after a password, passphrase,
  secret or key label, and any PEM block, is masked. A spec that prints a credential beside a
  username therefore never puts that credential into the proposal.

## The plan

### Partner

- **Matching.** The planner looks for an existing partner by, in order: the `partnerId` given; the
  code; the ISA ids on the partner's document profiles; the AS2 ids on its communication profiles;
  the customer.
  - Exactly one match: the plan updates that partner.
  - More than one: the plan stops at *Needs your decision* and creates nothing.
  - None: the plan creates a new External partner, but only when no partner with that code or
    name exists. Those are the unique indexes the create path enforces.
- **Status.** A new partner is created `Inactive`, and so is everything under it. No message can go
  out from a drafted setup until a person has entered its credentials, run a test case, and turned
  it on.

### Connection and capabilities

An outside partner has no `edi_connections` row: that table joins two Trenova organizations.
Its capabilities are:

- the partner's inbound and outbound flags;
- the transaction sets it has document profiles for.

The review shows both under *Connection*.

### Document profiles

There is one per transaction set and direction the spec covers (204 inbound, 990/214/210 outbound,
and so on). Each carries:

- the X12 version;
- the ISA and GS qualifiers and ids;
- the usage indicator;
- the four separators;
- the functional acknowledgment the partner expects: 997 or 999, and the window in minutes;
- the validation mode.

They are written through `UpsertPartnerDocumentProfile`.

### Control numbers

A sequence is written only when the spec states a range or a starting number. A re-run never
moves an existing sequence's next value backwards, because that would reissue numbers the partner
has already seen. A spec that asks for a lower start is listed as a decision instead.

### Communication profile

The plan carries the method (AS2, SFTP or VAN) and its non-secret settings:

- host, port, username, directories, file naming;
- AS2 ids, endpoint, MDN mode and URL, algorithms;
- ISA/GS ids, version, environment.

The profile is created `Inactive`.

### Templates

For each set and direction, the planner scores two kinds of candidate against the spec's stated
requirements:

- the organization's own templates for that document type (active or certified versions);
- the starter template (`templates.StarterSegments`).

It picks the one with the fewest unmet requirements; on a tie, the organization's own template
wins. Every requirement the chosen template does not meet is listed with its citation:

- a mandatory segment it never writes;
- a mandatory element it leaves empty;
- a code it cannot produce.

The plan never edits a template. Closing those gaps is template designer work, and the review
says so.

### Mappings

The partner's codes become mapping items (`SaveMappingItems`). Each target must be a record that
exists in this organization. A code whose target the agent could not name is listed as a gap.

### Samples

Every ISA…IEA run in the page text is checked in three ways:

- `edix12inspect.InspectX12` checks its structure.
- Its envelope is compared with the drafted profile for its set and direction: sender and
  receiver, qualifiers, version, functional group, separators.
- An inbound 204 sample is also parsed into a tender (`ediinboundservice.ParseSample`). The
  drafted mapping items are then run against it (`BuildInboundMappingPreview`, unsaved items), so
  the review shows which of the partner's codes still map to nothing.

## Validation

Every planned record goes through the validators the setup screens use:

- `ValidatePartner` and the partner's unique code and name;
- `ValidateCommunicationProfile`;
- `ValidatePartnerDocumentProfileRequest`, the template version check, and partner settings
  against their schema;
- `ValidateMappingItems`;
- each control number sequence's domain `Validate`.

The review shows each failure beside the record it belongs to. `apply_edi_connection_draft`
refuses to run while any of these hold:

- a validation failure;
- an unverified or held-back value in a field the write needs;
- an ambiguous partner match.

## Secrets

Passwords, private keys, host keys and certificates never come from the document into storage.
The input has no slot for them, and a value given for a secret key is refused. The plan leaves
them unset. The review's *Enter before turning this on* section names each one, with the setting
it belongs to and where to enter it:

- SFTP/VAN password or private key;
- known host key;
- AS2 private key, needed when a local certificate is set;
- partner signing and encryption certificates;
- basic auth password.

So that an inactive profile can be saved without them, the communication profile validator
requires the credential settings only of an `Active` profile:

- `password` or `privateKey`;
- `knownHostKey`;
- the AS2 private key when a local certificate is set.

Delivery and polling read only active profiles. Turning a profile on through the
communication profile screen validates it as active, so the profile cannot be turned on without
its credentials.

## Re-runs are change sets

Running the tool on a revised spec for a partner that already exists produces a diff against
what is configured now:

- each field is `add`, `change` (current → proposed) or `same`;
- each document profile, mapping item and sequence is new, changed or unchanged.

What the diff leaves alone:

- It deletes nothing. A mapping or profile the revised spec no longer mentions is listed as *not
  in this spec, kept*.
- It changes no status.
- It does not touch secrets.

The preview pins the version of every record it changes. The digest of that preview is what the
person approves, so a record that moved after the review refuses the approval.

## Outside content and the proposal

- **Taint and the write cap.** The spec is outside content. `draft_edi_connection` declares
  `ReadsExternal: always` with the attachment or document source, and it is listed in
  `outsideTextTools`. Every later write in the turn is therefore proposed, whatever the agent's
  tier.
- **Never acts on its own.** `apply_edi_connection_draft` has a `Propose` ceiling of its own.
- **Document digest.** The proposal records the document's digest: SHA-256 over the stored page
  text, in page order. `Preview` refuses when the document read now has a different digest.

## Where it lives

| Piece | Where |
|---|---|
| Planner, verification, diff | `modules/edi/app/ediservice/onboarding*.go` |
| Plan types | `modules/edi/ports/ediports/edionboarding.go` |
| Sample parse | `ediinboundservice.ParseSample` |
| Tools | `agent/app/agentquerytoolservice/edi_onboarding_tool.go`, `agent/app/agenttools/editools/apply_connection_draft.go` |
| Desk card | `routes/desk/_components/artifacts/cards/edi-connection-draft/` (with `edi-connection-draft-skeleton.tsx`) |
| Bench | `deskbench/scenarios/edi.yaml` (spec onboarding), fixture `deskbench/fixtures/edi/acme-retail-spec.pdf` |
| Simulator spec | `dev/edi-partner-sim/specs/` |
