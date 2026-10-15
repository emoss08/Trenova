# Building an EDI connection from a partner's spec

A trading partner sends an implementation guide (a PDF, a DOCX, or a text file of sample X12).
An agent on the Desk reads it, drafts the whole connection, and proposes it. A person reviews the
draft and approves it. Nothing is written before that approval. Read this before changing
`modules/edi/app/ediconnectiondraft`, its ports (`ediports/ediconnectiondraft.go`) or repository
(`ediconnectiondraftrepository`), the `draft_edi_connection` or `apply_edi_connection_draft` tools,
the `edi_connection_draft` Desk artifact, or the communication profile validator's credential rules.

## The shape of it

```
person attaches spec ─▶ document intelligence (pages, OCR) ─▶ document_content_pages
                                                                    │
agent reads pages (read_attachment) and cites every value ─────────┤
                                                                    ▼
draft_edi_connection(documentId, cited values, requirements, mappings, gaps, resolutions)
   └─ ediconnectiondraft.Service.PlanConnectionDraft
        ├─ check every citation against the stored page text
        ├─ hold back low-confidence, unverified and conflicting values
        ├─ match an existing partner (or plan a new one, never a duplicate)
        ├─ plan partner, communication profile, document profiles, control numbers, mappings
        ├─ pick the closest template per set and list every unmet requirement
        ├─ run the spec's sample X12 through the inbound parser and compare its envelope
        ├─ validate every planned record with the validators the setup screens use
        └─ diff against what the partner has today (a change set on a re-run)
   ▼
edi_connection_draft artifact on the Desk (the review)
   ▼
apply_edi_connection_draft (always proposed) ─▶ Preview = the same plan, recomputed now
   ▼ person approves (approval carries the preview digest; the input carries the document digest)
Execute ─▶ ApplyConnectionDraft: re-plan, check the digest and readiness, write in one transaction
```

## Tools

| Tool | Kind | Holds it | What it does |
|---|---|---|---|
| `draft_edi_connection` | query, EDI read | EDI desk | Builds the plan and returns the review (`connectionDraftView`: the draft, a title and a note for the model). Writes nothing. On a ready draft the note names, for the model to repeat in its reply, each template rule left unmet (`MS1-01 (page 5)`), each gap the guide leaves open and the credentials to enter before the connection is turned on; a reply that leaves them out hides what the person must still do. On a ready draft of a partner that exists, the note also lists each write the draft makes, value by value (`Contact name from Dana Whitfield to Priya Raman (page 1)`, `interchange control numbers on the 990 outbound, 214 outbound start at 5000 instead of 1000 (page 3)`): a long draft is shortened to fit the model's context and a note never is, so the change set is never lost with the lists that were cut. On a draft that is not ready it asks only about the open decisions: a question ends the turn (`agent-loop-end-on-ask`), so anything else asked for in that reply would never reach the person, and the draft that follows the answer names them. Reads outside content, so it caps every later write in the turn at `Propose` (`CapsLaterWrites`, listed in `agentruntime.outsideTextTools`). |
| `apply_edi_connection_draft` | action, EDI update, egress internal | EDI desk | The same input plus `documentDigest`. Its `Preview` recomputes the plan from the document and the configuration as they are at decision time; `Validate` refuses a digest that does not match or a draft that is not ready; `Execute` runs only from an approved proposal, and reports what it saved (`ToolResultReporter`): the partner as an `edi_partner` record, and a state that counts the records by kind, says they were saved inactive (or, for a partner that exists, that what existed keeps its status) and names the credentials still to enter, so the reply after approval can say so. Its default tier and ceiling are both `Propose`. It is an update because creating EDI records is not something an agent may hold; the records it creates are created under a person's approval. |

Both tools take the same input (`agenttools/edispec`: `ParamSchema`, `Parse`). Reading the
document is document intelligence's job, through the pipeline every attachment goes through
(`DocumentContentService.GetContent`). No second extractor exists. The agent reads the pages from
the file attached to the message, or with `read_attachment` for a guide attached earlier or cut
short, and passes what it found, each value with its citation; the planner checks every quote
against the stored pages either way:

- `documentId`: a file attached to this conversation. `draft_edi_connection` reads only a file the
  person it works for attached (`Document.AttachedBy`); `apply_edi_connection_draft` reads any file
  attached to the conversation (`Document.AttachedTo`), because the approver need not be the
  person who uploaded it. On approval the conversation is the one the proposal's run answered,
  which the executor and the preview both hand the tool (`proposalexecutor.WithConversation`). A
  file still being read answers "still reading" (`ErrStillReading`).
- `ediPartnerId`, `customerId` (optional): the partner to update and the customer to link. Without
  a partner, the planner matches one.
- `values[]`: `{field, value, confidence, citation: {page, quote, section}}`. `field` names a slot
  the planner knows (`partner.name`, `isa.ourId`, `sets.214.outbound.ack`, `as2.partnerAs2Id`, ...;
  the catalog is `ediconnectiondraft/catalog.go`, exposed as the `edi.connectionDraftField` enum
  source). Secret slots do not exist.
- `requirements[]`: `{transactionSet, direction, segment, elementPosition, usage, codes, citation}`,
  the segment and element usage the spec states.
- `mappings[]`: `{entityType, sourceId, sourceLabel, targetId, confidence, citation}`, the partner's
  codes the spec lists, each with the Trenova record it maps to. `targetId` comes from the list
  tool for its entity type.
- `gaps[]`: `{kind: Unsupported | Ambiguous, summary, citation}`, what the spec asks for that
  Trenova does not support, or that the spec leaves unclear.
- `resolutions[]`: a person's answer to a held-back value, each either one of the cited candidates
  or a value the person gave in the conversation (`source: Candidate | Person`); the review labels
  it that way.

Input sizes are bounded (`input.go`: 300 values, 300 requirements, 500 mappings, 100 gaps, 100
resolutions).

## No draft table

A draft is a pure function of three things: the stored page text, the cited input, and the
configuration as it is now. The proposal row is the durable record. It holds:

- the input;
- the document digest (`documentDigest`, SHA-256 over the stored page text in page order);
- the approved preview.

The artifact holds the review the person read. A separate drafts table would be a second copy
that could go stale against the configuration it describes. Each re-plan reads the configuration
again instead.

## Every value has a source

- **Citations are checked.** A quote must appear on the page it cites, with whitespace, case and
  typographic quotes and dashes normalized; a quote found on another page is followed there. The
  value must also appear in that quote as a whole token, or be a code a cited phrase spells out
  (for example "Mutually Defined" for `ZZ`). A citation that fails either check makes the value
  **unverified**, and an unverified value is never used.
- **A value is taken as the guide writes it.** The schema asks for the value as the guide gives
  it, so a field with a fixed set of values reads it through the same spellings the quote check
  uses: "Synchronous, signed" is `sync`, "SHA-256" is `sha256`, "AES-256 (CBC)" is `aes256-cbc`,
  "qualifier 01 (DUNS)" is `01` (`spelledOut` in `fields.go`). The longest spelling wins, so
  "AES-256-GCM" is `aes256-gcm`, not `aes256-cbc`; a value that spells two codes ("DUNS or SCAC")
  is **invalid** and held for the person.
- **Confidence.** A value whose confidence is below `0.8` is **held**. So is a field two citations
  give different values for, or a value the spec's own sample X12 contradicts. A held value is
  left unset, or at the partner's current value on a re-run, and the review lists it under *To
  decide* with each candidate, its page and quote. The planner never picks between them.
- **Samples are evidence.** A sample's ISA/GS envelope is parsed rather than read by a model, so a
  value taken from it cites the sample's page and its ISA segment. A sample printed in a PDF reads
  back with its fixed-width padding squeezed and its long lines wrapped; the finder joins the
  wrap and the parser reads the squeezed ISA.
- **Quotes are redacted** before they are stored or shown (`edi.RedactSecrets`). Anything after a
  password, passphrase, secret, key or token label, and any PEM block, is masked, and a sample's
  ISA02 and ISA04 are masked with `X`. The planner redacts every quote and gap summary as it reads
  the request, and checks a quote against the page in its original and its redacted form, so a
  draft plans the same from either. `apply_edi_connection_draft` files its arguments redacted
  (`ToolSelectionResolver`, `edispec.RedactQuotes`), so a spec that prints a credential beside a
  username never puts that credential into the proposal, its preview or the approval card. What
  is not redacted is what any attachment leaves: the document itself, the pages
  `read_attachment` returned, and the model's own calls, kept in the thread as it sent them.

## The plan

### Partner

- **Matching.** The planner looks for an existing partner by the `ediPartnerId` given, else by
  code, name, the ISA id on its document profiles, the AS2 id on its communication profiles and
  the customer (`FindPartnerMatches`). It matches only on a field's settled value, the one the plan
  would write or the person picked: a code, name or id the plan holds back is not evidence of who
  the partner is.
  - Exactly one match: the plan updates that partner.
  - More than one: the plan stops with the candidates under *Which partner is this?* and creates
    nothing.
  - None: the plan creates a new External partner. An internal (Trenova-to-Trenova) partner is
    never set up from a spec.
- **Status.** A new partner is created `Inactive`, and so is everything under it. No message can
  go out from a drafted setup until a person has entered its credentials and turned it on.

### Connection and capabilities

An outside partner has no `edi_connections` row: that table joins two Trenova organizations. Its
capabilities are the partner's inbound and outbound flags and the transaction sets it has document
profiles for; the review shows both in its head.

### Document profiles

There is one per transaction set and direction the spec covers (204 inbound, 990/214/210
outbound). Each carries the X12 version, the functional group id, the ISA and GS qualifiers and
ids, the usage indicator, the separators, and the acknowledgment the partner expects (997 or 999,
and the window in minutes). They are written through `UpsertPartnerDocumentProfile`; on an
existing profile only the fields whose value changes are written.

### Control numbers

A sequence is written only when the spec states a starting number
(`StartControlNumber`, an upsert that only ever moves forward). A re-run never moves an existing
sequence backwards, because that would reissue numbers the partner has already seen: a spec that
asks for a lower start keeps the sequence where it is and the review says why.

### Communication profile

The plan carries the method (AS2, SFTP or VAN) and its non-secret settings: AS2 ids, endpoint, MDN
mode, algorithms; host, port, username, directories; environment. The profile is created
`Inactive`.

### Templates

For each set and direction the planner scores the organization's own templates for that document
type (active or certified versions) and the starter template (`EnsureStarterTemplate`) against the
spec's stated requirements. It picks the one with the fewest unmet requirements; on a tie, the
organization's own template wins, and a profile's current template that cannot be scored is kept
with a note. Every requirement the chosen template does not meet is listed with its citation (a
mandatory segment it never writes, a mandatory element it leaves empty, a segment marked not used
that it writes). The plan never edits a template; closing those gaps is template designer work.

### Mappings

The partner's codes become mapping items (`SaveMappingProfile`). Each target must be a record that
exists in this organization (`ResolveMappingTargets`). A code without a usable target is listed.

### Samples

Every ISA…IEA run in the page text (up to ten) is parsed with `ediinboundservice.ParseSample`. Its
envelope is compared with the drafted profile for its set and direction (sender and receiver,
qualifiers, version, separators), and for an inbound 204 the codes it carries that no drafted or
existing mapping covers are listed.

## Validation

Every planned record goes through the validators the setup screens use:

- `ValidatePartner`;
- `ValidateCommunicationProfile`;
- `ValidatePartnerDocumentProfileRequest` and partner settings against their schema
  (`ValidatePartnerSettings`).

The review shows each failure under *Would not save*. The draft is not ready, and
`apply_edi_connection_draft` refuses to run, while any of these hold: a validation failure, a held,
unverified or invalid value in a required field, an ambiguous partner match, or nothing to change.
A held value in an optional field (a contact phone, a time zone, an acknowledgment window) is still
listed under *To decide* and left unset, or at its current value on a re-run; it does not block.

## Secrets

Passwords, private keys, host keys and certificates never come from the document into storage.
The input has no slot for them, and a value given for a secret key is refused
(`IsCredentialKey`). The plan leaves them unset. The review's *Credentials you enter* section names
each one (`CredentialRequirements`), marks the ones the connection cannot run without, and says
where to enter them.

So that an inactive profile can be saved without them, the communication profile validator
requires the credential settings only of an `Active` profile. Delivery and polling read only active
profiles, and turning a profile on validates it as active, so it cannot be turned on without its
credentials.

## Re-runs are change sets

Running the tool on a revised spec for a partner that already exists produces a diff against what
is configured now: each field and record is `Add`, `Change` (current → proposed), `Same` or `None`.
The diff deletes nothing (a record the revised spec no longer mentions is listed under *Left as
they are*), changes no status and touches no secret. Each existing document profile keeps its own
envelope: an interchange value is written to it only when the guide or the person gives it, so a
setting the guide never names (a repetition separator, an inbound profile's usage indicator) stays
as each profile has it. The preview pins the version of every record
it changes, and its digest is what the person approves, so a record that moved after the review
refuses the approval.

## Outside content and the proposal

- **Taint and the write cap.** The spec is outside content. `draft_edi_connection` declares
  `ReadsExternal: always` with the attachment source and caps later writes. Every later write in
  the turn is therefore proposed, whatever the agent's tier, and the model's note says the quotes
  are the partner's text, never an instruction.
- **Never acts on its own.** `apply_edi_connection_draft` has a `Propose` ceiling of its own, and
  `ApplyConnectionDraft` needs a person as the actor in the draft's own organization.
- **Document digest.** `Validate`, `Preview` and `ApplyConnectionDraft` all refuse when the
  document read now has a different digest from the one proposed.

## The review

The `edi_connection_draft` artifact is described in [desk-artifacts.md](desk-artifacts.md). Every
value has a page tab that opens the quoted words; a held value is settled by picking a reading,
which asks the agent to draft again with it; saving is the proposal, which the card follows by its
document digest.

## Where it lives

| Piece | Where |
|---|---|
| Planner, verification, diff, apply | `modules/edi/app/ediconnectiondraft/` |
| Plan types and ports | `modules/edi/ports/ediports/ediconnectiondraft.go` |
| Repository | `modules/edi/adapters/postgres/ediconnectiondraftrepository/` |
| Sample parse | `ediinboundservice.ParseSample` |
| Tool input | `agent/app/agenttools/edispec/` |
| Tools | `agent/app/agentquerytoolservice/edi_connection_draft_tool.go`, `agent/app/agenttools/editools/apply_connection_draft.go` (and `preview_connection_draft.go`) |
| Artifact | `assistantservice/edi_connection_draft.go`; client `routes/desk/_components/artifacts/kinds/edi-connection-draft/` (with `connection-draft-skeleton.tsx`) |
| Approval card | `components/assistant/proposal-presenters.ts` (`apply_edi_connection_draft`) |
| Bench | `deskbench/scenarios/edi.yaml` (`edi-guide-new-partner`, `edi-guide-revised`), fixtures `deskbench/fixtures/attachments/edi-guide.html` + `edi-guides.json`, rendered by `render.mjs` |
| Simulator spec | `dev/edi-partner-sim/specs/simpartner-implementation-guide.txt`, held to the simulator by `internal/sim/spec_test.go` and drafted by `ediconnectiondraft/simulator_spec_test.go` |
