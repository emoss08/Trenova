# EDI Designer Feature-Complete Roadmap

This document defines the target architecture and staged implementation path for
feature-complete, self-service EDI in Trenova. It is not an MVP or V1 roadmap.
The goal is a production-grade EDI designer and execution platform that lets
operations, implementation, and partner-facing teams configure, certify,
operate, and troubleshoot partner EDI without engineering changes for routine
partner variation.

## Product Goal

The EDI Designer should make Trenova capable of full external and internal EDI
exchange across the transportation lifecycle. A trained implementation user
should be able to configure a partner, select supported transaction sets, design
or adapt templates, map Trenova source data into outbound documents, parse
inbound documents into normalized business events, test the configuration,
certify with the partner, promote versions safely, and operate the live
integration with complete auditability.

The finished product must support:

- Standards-based X12 envelope, segment, element, and acknowledgment behavior.
- Partner-specific implementation-guide rules without custom deployments.
- Safe Starlark scripting for partner mapping logic.
- Declarative transform pipelines for common non-code mapping operations.
- Conditional rendering for segments, loops, and elements.
- Versioned templates and script libraries with draft, certification, active,
  deprecated, and rollback lifecycle states.
- Transport execution over internal exchange, AS2, SFTP, and VAN workflows.
- Inbound parsing, validation, acknowledgment generation, exception review, and
  replay.
- A message archive suitable for audit, support, certification evidence, and
  operations.

## Target Architecture

The target architecture separates design-time configuration from runtime
execution while keeping a single domain model for templates, partner schemas,
validation, transport, messages, acknowledgments, and exceptions.

### Major Components

- **EDI Designer UI**: React designer for templates, source context browsing,
  transforms, Starlark scripts, conditions, validation rules, test fixtures, and
  partner certification artifacts.
- **Template Service**: Backend service for template versions, transaction-set
  metadata, loops, segments, elements, validation rules, source bindings,
  transforms, conditions, and promotion workflow.
- **Script Library Service**: Backend service for reusable Starlark libraries,
  approved helper surfaces, static analysis, dependency tracking, versioning,
  and usage references.
- **Rendering Runtime**: Backend runtime that renders outbound documents from
  domain payloads using template versions, Starlark, transform pipelines,
  conditions, partner settings, and envelope/control-number services.
- **Parsing Runtime**: Backend runtime that parses inbound documents into
  envelope, group, transaction, segment, loop, and normalized business payload
  structures.
- **Validation Runtime**: Shared runtime for syntax validation, implementation
  guide rules, partner rules, source data requirements, duplicate detection, and
  business validation.
- **Transport Runtime**: Worker-backed execution for AS2, SFTP, VAN, and
  internal sends and receives, including retries, polling, receipts, MDNs,
  mailbox state, and dead-letter behavior.
- **Acknowledgment Runtime**: Generation and processing of TA1, 997, and 999
  acknowledgments, with reconciliation back to archived messages and transport
  attempts.
- **Message Archive**: Immutable operational record of inbound/outbound payloads,
  normalized payloads, rendered payloads, validation diagnostics,
  acknowledgments, transport attempts, replay metadata, and audit events.
- **Exception Workbench**: Operational surface for mapping misses, validation
  failures, transport errors, rejected acknowledgments, duplicate messages, and
  controlled replay.

### Runtime Flow

Outbound flow:

1. A business event or user action creates an EDI send request.
2. The runtime resolves partner capability, active profile, template version,
   partner settings, source context, envelope settings, and control numbers.
3. Conditions decide which loops, segments, and elements render.
4. Element values resolve through constants, field paths, partner settings,
   mappings, runtime values, transform pipelines, or Starlark.
5. Validation runs before archive finalization and transport submission.
6. The message archive stores the canonical request, rendered document,
   diagnostics, and outbound transport job.
7. Transport sends the payload and records delivery receipts, MDNs, VAN
   responses, or polling results.
8. Acknowledgment tracking reconciles TA1/997/999 and raises exceptions for
   missing, rejected, duplicate, or late acknowledgments.

Inbound flow:

1. Transport receives or polls for an EDI payload.
2. The archive stores the raw payload before parsing.
3. Envelope and partner resolution identify the sender, receiver, transaction
   set, document type, and expected parser/template.
4. Syntax and duplicate-control-number validation run.
5. The parser maps X12 segments into normalized business payloads.
6. Partner and business validation run.
7. Acknowledgments are generated when required.
8. Valid documents create domain commands, review records, or workflow events.
9. Invalid or ambiguous documents enter the exception workbench with replay and
   correction options.

## Completed Stages

These stages describe the current foundation already available or recently
hardened. They are not the end state.

### Stage 0: Internal EDI Foundation

- Partner management, internal partner connections, communication profile
  metadata, mapping profiles, load tender transfers, review flows, shipment
  links, audit logging, and transfer lifecycle visibility exist.
- Current production behavior is strongest for internal organization-to-
  organization load tender exchange.
- External transport configuration data exists, but full external send/receive
  execution is not complete.

### Stage 1: Template and Rendering Foundation

- Backend template structures support X12 204-style segments and elements.
- Runtime values and envelope separators can be resolved for rendered documents.
- Basic validation modes exist for strict, warn-only, and disabled rendering.
- The first rendering path targets outbound 204 generation from load tender
  payloads.

### Stage 2: Starlark Element Rendering

- Element-level Starlark source rendering is supported for scalar values.
- Scripts receive source context and can use approved helper functions.
- Runtime diagnostics are converted into renderer diagnostics with safe paths.
- Disabled validation still preserves rendering and Starlark diagnostics.

### Stage 3.5: Starlark Rendering Hardening

- Repeat rendering supports scripts that read repeat data from either the second
  function argument or `ctx["item"]` / `ctx["repeat"]`.
- Starlark `None` results fall through to normal renderer validation.
- Max-length validation, truncation, separator sanitization, missing function
  diagnostics, and repeat aliases have focused backend tests.

### Stage 3.6: Backend Transform Pipeline Rendering

- Backend `TemplateElementSourceTransform` rendering is supported for outbound
  204 element values.
- Transform base sources can read constants, field paths, partner settings,
  mappings, runtime values, and repeat values through the shared direct-source
  resolver.
- Transform pipelines support common scalar mapping operations including trim,
  upper/lower, default, coalesce, date/time and numeric formatting,
  normalization, qualifier lookup, substring, padding, punctuation removal,
  replace, string predicates, concatenation, required, empty-if-none, and
  conditional selection.
- Transform diagnostics use `transform_error` with segment, element position,
  source path, message, and suggested fix metadata.
- Disabled validation preserves transform diagnostics, warn-only downgrades
  transform errors, and strict mode treats transform errors as blocking.
- Normal renderer post-processing still applies after transform output:
  element defaults, required validation, max-length truncation, and X12
  separator sanitization.

### Stage 3.7: Backend Condition Rendering

- Backend segment and element conditions are supported for outbound 204
  rendering.
- Declarative conditions support path truthiness, negation, string equality, and
  string inequality against shipment, repeat, partner, mapping, and runtime
  context roots.
- Repeated segment conditions can filter individual repeat items through the
  repeat aliases used by value rendering.
- Element conditions blank skipped element values and bypass required validation
  for skipped elements.
- Starlark-backed conditions use the same safe Starlark evaluator, source
  context, repeat aliases, timeout, and step-limit controls as value rendering.
- Condition diagnostics use `condition_error` with segment, element position,
  condition path, message, suggested fix metadata, and validation-mode handling.
- Starlark condition diagnostics preserve the underlying Starlark diagnostic
  code in the condition error message, such as `starlark_runtime_error` or
  `starlark_step_limit`.

### Stage 3.8: Frontend Template Designer Foundation

- `/edi/designer` now defaults to a Templates workspace instead of the document
  preview surface.
- The frontend has backend-aligned DTOs and service/query helpers for template
  creation, version metadata, draft cloning, segment replacement, script
  library replacement, validation, certification, activation, archive, source
  context field search, and partner setting field search.
- Draft segment, element, transform, condition, and script-library edits use an
  explicit Save Draft flow against backend draft endpoints.
- Certified, active, superseded, deprecated, and archived versions are presented
  as read-only with disabled editing controls.
- Validation diagnostics are grouped by severity, can select related
  segment/element rows when identifiable, and decorate outline/element rows.
- The designer includes source and partner setting path pickers, transform
  operation metadata, condition-string construction, and CodeMirror Starlark
  script editing with discovered function names.
- The existing outbound document profile, preview, generate, and archive
  workflow remains available as the Document Preview & Archive tab.
- The frontend designer foundation has been modularized under focused
  designer, archive, script, transform, element, profile, hook, and utility
  folders while preserving the existing workspace import path.
- Template, archive, and message inspector state that should survive reloads is
  backed by `nuqs` URL parameters; drafts, editor contents, and payload bodies
  remain local component state.
- Template and archive server state is routed through React Query hooks, heavy
  workspace sections are lazy-loaded, and shared designer/message/date helpers
  are isolated in tested utility modules.
- Designer select-like controls now use the shared Trenova field UX: static EDI
  enums are rendered through controlled `SelectField` adapters, backend-owned
  values use autocomplete-backed EDI select-options endpoints, and source /
  partner path browsing keeps manual path editing while inserting through
  autocomplete popovers.
- The EDI Partners workspace supports direct External partner setup alongside
  the existing Internal partner-pair connection workflow; direct Internal
  partner creation remains pair controlled.
- The outbound message inspector is now modularized under the designer
  inspector workspace with URL-backed tab and selected-segment state,
  backend-backed X12 inspection, formatted and raw inspection views,
  dictionary-backed segment/element labels, diagnostics navigation, payload,
  control number, and provenance views.
- Backend X12 inspection now detects ISA/envelope/fallback separators, preserves
  exact raw payloads and segment offsets, returns segment/element/component
  introspection, formats human-readable X12, and emits structural diagnostics
  for envelope order, control-number mismatches, and trailer counts.
- The outbound renderer preserves template element positions, fixed-width ISA
  element padding, configured delimiter fallbacks, separator sanitization, and
  generated SE/GE/IEA trailer counts/control-number consistency.

## Remaining Stages

The stages below are ordered by dependency and operational value. A stage is
complete only when domain model, API, runtime behavior, UI, tests, and
operational support are present where relevant.

### Stage 4: Template Versioning and Transaction-Set Model

- [x] Add first-class backend transaction sets plus loop, segment, element, and
  code-list dictionary tables for the outbound 204 foundation.
- [x] Support backend draft, certified, active, deprecated, archived,
  superseded, and rollback lifecycle states for template versions.
- [x] Keep active/certified template versions immutable in service workflows and
  allow edits only through cloned draft versions.
- [x] Pin partner document profiles to stable template versions by default and
  prevent active profiles from pinning draft or archived versions.
- [x] Add backend APIs for template creation, draft cloning, version update,
  segment replacement, validation, certification, activation, archive, and
  rollback.
- [x] Require backend validation and certification before normal activation,
  while allowing rollback to validated certified, superseded, or deprecated
  versions.
- [ ] Add frontend designer workflows for version diffing, promotion review, and
  certification evidence.
- [ ] Track source context schema and script library compatibility once those
  first-class models exist.

### Stage 5: Transform Pipelines

- [x] Implement backend transform pipeline rendering for common mapping operations that do
  not require Starlark.
- [x] Support trim, uppercase, lowercase, default, coalesce, format date/time,
  format decimal, normalize phone, normalize postal, qualifier lookup, substring,
  padding, punctuation removal, concatenation, conditional default, and code-map
  transforms.
- [x] Allow transform steps to read from base sources, previous step output,
  partner settings, mapping profiles, runtime values, and literal arguments.
- [x] Validate transform configuration at render time.
- [ ] Validate transform configuration at design time.
- [ ] Expose per-step preview and diagnostics in the designer.

### Stage 6: Condition Rendering

- [ ] Add condition rendering for loops and transform branches.
- [x] Add condition rendering for segments and elements.
- [x] Support declarative path, negated path, equality, and inequality
  predicates.
- [x] Support Starlark predicates for complex partner rules.
- [x] Conditions must run with the same source context and safety controls as
  value rendering.
- [x] Diagnostics must identify failed and invalid condition paths.
- [ ] Diagnostics and previews must identify skipped condition paths when needed
  for designer explainability.
- [ ] Add designer previews showing why a segment or element rendered or did not
  render for a selected fixture.

### Stage 7: Script Libraries

- [x] Add backend template-version-owned reusable Starlark libraries for draft,
  certified, active, superseded, and archived template versions.
- [x] Clone script libraries into new draft versions and keep library statuses
  aligned with template version lifecycle transitions.
- [x] Support library-backed element Starlark functions, inline scripts calling
  library helpers, and library-backed Starlark conditions.
- [x] Validate template-version libraries for required fields, language,
  duplicate names, duplicate functions, syntax, missing function references, and
  callable references.
- [x] Cover runtime, renderer, and service behavior for version-owned script
  libraries with backend tests.
- [ ] Add reusable Starlark libraries scoped by tenant, transaction set, partner,
  and global system templates.
- Support library versions, imports from approved library references, dependency
  graphs, static validation, promotion workflow, and usage impact analysis.
- Prevent circular dependencies and unsafe dynamic imports.
- Allow templates to pin exact script library versions.
- Add library-level tests and fixture-based certification evidence.

### Stage 8: Source Context Browser

- [x] Add backend source context schema models and seeded outbound X12 204
  metadata for shipment, repeat, partner, runtime, and future mapping paths.
- [x] Provide backend list/search APIs for source context schemas and fields.
- [x] Validate supported template source paths against active source context
  metadata, including repeat context mismatches and future/deprecated fields.
- [ ] Show field paths, data types, nullability, examples, domain descriptions,
  repeat boundaries, and source ownership in the frontend browser.
- [ ] Support direct field insertion into elements, transforms, conditions, and
  Starlark editor snippets.
- [ ] Include source context preview, explainability, repeat aliases, envelope
  values, and resolved mapping context in the browser.
- [ ] Add schema versioning pins so template versions know exactly which context
  shape they were certified against.

### Stage 9: Partner Settings Schema

- [x] Add backend versioned partner settings schema models and metadata fields.
- [x] Seed global outbound X12 204 partner settings metadata for carrier,
  contact, bill-to, shipper, consignee, default equipment/payment/weight/
  packaging values, reference qualifiers, stop mappings, accessorial codes,
  commodity defaults, and envelope bridge metadata.
- [x] Expose backend listing, search, and validation APIs for partner setting
  schemas and fields.
- [x] Validate profile partner settings on active profile create/update while
  allowing inactive profiles to retain invalid draft settings.
- [x] Validate template `partner.*` references against partner setting schema
  metadata without changing render or Starlark value semantics.
- [ ] Add frontend structured settings editor and environment-specific settings
  management.
- [ ] Add secret-management UI and encrypted sensitive partner setting storage.
- [ ] Add designer insertion/snippet integration for partner setting paths.

### Stage 10: Frontend Designer

- [x] Build the initial template designer workspace for outbound 204 templates.
- [x] Provide template outline navigation by version, segment, and element.
- [x] Add template list filters and create-template flow.
- [x] Add draft-only segment and element editing with explicit Save Draft.
- [x] Add source selector and partner setting selector backed by backend field
  search APIs.
- [x] Add transform builder for backend-supported operations.
- [x] Add condition editor for persisted backend condition strings.
- [x] Add template-version script library editing with CodeMirror.
- [x] Add validation diagnostics panel and lifecycle actions.
- [x] Preserve existing document preview, generate, and archive workflow as a
  secondary designer tab.
- [x] Refresh designer headers, sticky rails, read-only status, selected
  segment/element affordances, and document preview/generate hierarchy while
  keeping React Query server state and shared editor utilities.
- [ ] Provide template outline navigation by envelope, transaction set, loop,
  segment, and element.
- Support drag/reorder where X12 ordering rules allow it and prevent invalid
  structural edits.
- Provide validation rule editor, code list selector, rendered preview,
  diagnostics panel, version diff, and promotion workflow.
- Include fixture management and side-by-side comparison between rendered output,
  expected output, and partner implementation-guide examples.
- Surface safety warnings for scripts, unsafe truncation, required data gaps,
  unmapped codes, missing partner settings, and acknowledgment risks.

### Stage 11: Message Archive

- Completed for generated outbound messages:
  - Outbound message detail inspector.
  - Backend-backed X12 inspection endpoints for archived messages and raw X12
    payloads.
  - Raw X12 viewer.
  - Formatted X12 view with segment labels, element labels, empty markers, and
    composite breakdowns.
  - Payload snapshot viewer.
  - Diagnostics detail grouping.
  - Control number display and copy.
  - Segment tree and segment detail with dictionary labels, control badges, and
    element diagnostics.
  - Download and copy support.
- Remaining:
  - Transport attempts.
  - Acknowledgment history.
  - Replay.
  - Inbound archive.
  - Immutable payload hash and tamper evidence.
  - Advanced archive search beyond partner, transaction set, direction, status,
    generated date range, and ID/control-number query filters.

### Stage 12: Test and Certification Workbench

- Add a workbench for partner onboarding and certification.
- Support fixture libraries, generated sample payloads, imported partner sample
  files, expected output assertions, syntax validation reports, partner rule
  checklists, acknowledgment simulations, and certification sign-off.
- Allow certification packs to be exported for partner review.
- Require passing certification checks before activating a partner template in
  production mode.
- Track certification evidence by partner, transaction set, template version,
  script library version, transport profile, and environment.

### Stage 13: Transport Execution

- Implement send and receive execution for internal, AS2, SFTP, and VAN
  profiles.
- Add transport workers, retry policy, idempotency keys, delivery attempt
  records, dead-letter queues, mailbox checkpoints, file naming rules, and
  environment separation.
- AS2 must support signing, encryption, certificate rotation, synchronous and
  asynchronous MDNs, compression when configured, and partner-specific headers.
- SFTP must support key-based auth, password auth where allowed, directory
  polling, archive folders, duplicate file detection, atomic download/upload,
  and partner file naming conventions.
- VAN support must model mailbox credentials, polling windows, delivery receipts,
  partner-specific routing IDs, and provider-specific response metadata.

### Stage 14: Acknowledgments

- Generate TA1 where configured for interchange-level acceptance or rejection.
- Generate 997 and 999 acknowledgments for supported inbound transaction sets.
- Process inbound TA1, 997, and 999 for outbound messages.
- Reconcile acknowledgments to interchange, group, transaction, and business
  records.
- Track expected acknowledgment windows and raise exceptions for missing, late,
  duplicate, rejected, or structurally invalid acknowledgments.
- Expose acknowledgment state in message archive, partner dashboards, and
  exception workbench.

### Stage 15: Inbound EDI

- Implement inbound parsing and business application for 204, 990, 214, 210,
  and acknowledgment transaction sets according to partner capability.
- Normalize inbound documents into reviewable business payloads before mutating
  core shipment or billing state.
- Support duplicate detection, replay protection, mapping-required states,
  manual review, automated acceptance where configured, and audit logging.
- Inbound 204 should create or update tender review records.
- Inbound 990 should update tender response state.
- Inbound 214 should create shipment status events.
- Inbound 210 should create invoice review or vendor invoice workflows as
  accounting capabilities allow.

### Stage 16: Exception Workbench

- Centralize operational exceptions across rendering, parsing, validation,
  mapping, transport, acknowledgments, duplicates, and domain application.
- Provide queue filters, ownership, severity, SLA timers, root-cause grouping,
  notes, audit history, and resolution workflow.
- Support safe replay from archived messages after configuration changes.
- Distinguish retryable technical failures from business validation failures.
- Provide suggested fixes from diagnostics where available.

### Stage 17: Production Operations and Observability

- Add dashboards for throughput, failures, acknowledgment latency, transport
  latency, retry rates, partner error rates, duplicate detections, and stale
  messages.
- Add alerts for missing acknowledgments, transport outages, credential expiry,
  certificate expiry, mailbox polling failures, repeated partner rejects, and
  dead-letter growth.
- Add runbooks for partner outage, replay, credential rotation, certificate
  rotation, template rollback, and emergency disablement.

## Domain Model

The feature-complete domain model should include these aggregate areas.

### Partner and Capability Model

- `edi_partners`: partner identity, operational status, direction flags, linked
  customer/internal organization, default profiles, and ownership.
- `edi_partner_capabilities`: enabled transaction sets, direction, transport
  modes, acknowledgment requirements, automation policy, certification state,
  and effective dates.
- `edi_partner_settings_schemas`: versioned setting definitions used by
  templates and transports.
- `edi_partner_settings_values`: typed setting values by partner, environment,
  and schema version, with secret-value handling.

### Template Model

- `edi_template_families`: reusable base by standard, document type, and
  transaction set.
- `edi_template_versions`: immutable versioned definitions with lifecycle,
  source context schema version, script library pins, certification state, and
  activation metadata.
- `edi_template_drafts`: mutable working copies before promotion.
- `edi_template_loops`: loop structure, repeat source, max use, conditions, and
  parent-child relationships.
- `edi_template_segments`: segment definitions, sequence, required state,
  conditions, max use, and loop ownership.
- `edi_template_elements`: element source configuration, validation rules,
  transforms, Starlark function references, code lists, defaults, and conditions.
- `edi_template_version_events`: audit trail for create, edit, certify,
  activate, rollback, deprecate, and archive actions.

### Runtime Model

- `edi_messages`: archive root for inbound and outbound messages.
- `edi_message_payloads`: raw, rendered, normalized, and redacted payload
  variants.
- `edi_message_diagnostics`: validation, rendering, parsing, transport, and
  acknowledgment diagnostics.
- `edi_transport_attempts`: send/receive attempts, method, status, request and
  response metadata, retry state, and failure details.
- `edi_acknowledgments`: generated and received TA1/997/999 records and
  reconciliation state.
- `edi_control_numbers`: allocated, consumed, received, and duplicate-detected
  ISA, GS, and ST control numbers.
- `edi_exceptions`: operational exception records with queue, ownership,
  severity, source message, and resolution workflow.

### Design Support Model

- `edi_script_libraries`: reusable Starlark library records.
- `edi_script_library_versions`: immutable library versions with static
  validation result and dependency graph.
- `edi_test_fixtures`: source payloads, partner settings, expected X12, expected
  diagnostics, and certification tags.
- `edi_certification_runs`: workbench executions and sign-off evidence.
- `edi_source_context_schemas`: versioned schema for outbound and inbound source
  data.

## Starlark Runtime

Starlark remains the controlled extension point for partner-specific mapping
logic that cannot be expressed cleanly through field paths or transform
pipelines.

Requirements:

- Scripts execute with deterministic timeouts and execution step limits.
- Context is immutable and contains `shipment`, `partner`, `runtime`, `mapping`,
  and repeat aliases where applicable.
- Repeat functions may read repeat data through a second function argument,
  `ctx["item"]`, or `ctx["repeat"]`.
- Return values are restricted to scalar strings, numbers, booleans, or `None`.
- Approved helpers cover common formatting, qualifier, date/time, defaulting,
  and normalization behavior.
- Imports are disabled except for approved version-pinned script libraries.
- Diagnostics must include code, severity, segment, element position, safe path,
  message, and suggested fix.
- Runtime must never expose secrets unless a field is explicitly allowed and
  redacted in diagnostics.
- Static validation should catch missing required functions, unsupported imports,
  helper misuse, and obvious arity errors before activation.

## Transform Pipelines

Transform pipelines are the preferred tool for routine mapping behavior.
Starlark should be reserved for logic that cannot be represented declaratively.

Pipeline requirements:

- A pipeline has a base source and ordered transform steps.
- Steps are typed, validated, previewable, and serializable.
- Each step receives the previous step output plus safe access to configured
  arguments.
- Steps must produce diagnostics without panics or swallowed errors.
- Pipelines must support design-time preview against fixtures.
- Pipelines must be reusable through template cloning and version diffs.

## Condition Rendering

Conditions determine whether loops, segments, elements, and transform branches
render.

Requirements:

- Conditions support declarative predicates for common checks such as exists,
  empty, equals, in-list, greater-than, partner setting enabled, and mapping
  resolved.
- Starlark conditions are supported for complex partner rules.
- Conditions must be evaluated before required validation for skipped elements.
- Diagnostics must distinguish an invalid condition from a false condition.
- Preview must show condition inputs, output, and skipped structure.

## Template Versioning

Template versioning must protect live partner integrations from accidental
behavior changes.

Requirements:

- Active versions are immutable.
- Drafts are editable and can be cloned from active versions.
- Certification runs attach to exact template, script library, source context,
  partner settings schema, and transport profile versions.
- Activation requires passing configured validation and certification gates.
- Rollback activates a previous certified version without mutating history.
- Diff views show structural, source, transform, condition, validation, script,
  and partner setting changes.
- Runtime messages store exact version references for audit and replay.

## Script Libraries

Script libraries should reduce duplication while preserving safety.

Requirements:

- Libraries are versioned and immutable after promotion.
- Templates pin exact library versions.
- Library imports use stable logical names, not arbitrary file paths.
- Dependency graphs are visible and validated.
- Library changes require impact analysis across dependent templates.
- Tests can run at the library level and template level.

## Frontend Designer

The designer is the primary self-service surface for implementation teams.

Requirements:

- Template outline with loop, segment, and element hierarchy.
- Element editor for source type, base source, transforms, Starlark, conditions,
  defaults, validation, code lists, and notes.
- Starlark editor with syntax highlighting, helper documentation, static
  diagnostics, fixture execution, and output preview.
- Transform builder with per-step previews.
- Condition editor with human-readable predicates and Starlark fallback.
- Source context browser embedded beside the editor.
- Rendered X12 preview with segment and element highlighting.
- Diagnostics panel connected to the selected template node.
- Version diff and promotion workflow.
- Certification workbench entry points.
- Clear separation between draft, certified, active, deprecated, and archived
  versions.

## Source Context Browser

The source context browser is the bridge between Trenova domain data and EDI
configuration.

Requirements:

- Browse outbound payloads such as load tender, tender response, shipment
  status, invoice, and acknowledgments.
- Browse inbound normalized payloads for business application.
- Show paths, types, examples, repeat context, nullability, and descriptions.
- Include partner settings, runtime values, mapping outputs, and envelope data.
- Provide copy/insert actions for field paths and Starlark snippets.
- Support schema version pinning and compatibility warnings.

## Partner Settings Schema

Partner settings must become typed where they affect rendering, parsing,
transport, or acknowledgments.

Requirements:

- Define schema fields with type, label, description, required flag, default,
  environment scope, secret flag, enum options, and validation rules.
- Use schemas to drive UI forms and design-time validation.
- Allow templates and scripts to reference settings by stable keys.
- Redact secret values in diagnostics, archive views, logs, and previews.
- Version schemas and track compatibility with active templates.

## Message Archive

The archive is the operational source of truth for EDI traffic.

Requirements:

- Store immutable inbound and outbound message records.
- Store raw payloads, rendered payloads, normalized payloads, redacted display
  payloads, hashes, metadata, diagnostics, acknowledgments, and attempts.
- Support replay with explicit user action, audit trail, and idempotency checks.
- Support retention policies and legal/audit export.
- Link messages to shipments, invoices, transfer records, partners, profiles,
  template versions, and exceptions.

## Test and Certification Workbench

The workbench turns EDI onboarding into a repeatable process.

Requirements:

- Fixture-based render, parse, validate, acknowledge, and transport simulation.
- Partner sample file import and expected-output comparison.
- Certification checklists by transaction set and partner.
- Evidence export containing rendered files, diagnostics, version references,
  test results, and approval metadata.
- Promotion gates based on successful certification runs.
- Regression runs when templates, script libraries, partner settings schemas, or
  source context schemas change.

## Transport Execution

Transport execution must be durable, observable, idempotent, and partner-aware.

Requirements:

- Worker-backed send/receive jobs with retry, backoff, timeout, and dead-letter
  behavior.
- Profile-specific validation for internal, AS2, SFTP, and VAN.
- Secret handling through encrypted storage and redacted logs.
- Idempotency keys for sends and receives.
- Mailbox checkpoints for polling transports.
- Transport attempt records with request metadata, response metadata, and
  failure classification.
- Environment isolation for test, certification, and production traffic.

## Acknowledgments

Acknowledgments are first-class records, not incidental files.

Requirements:

- Generate and process TA1, 997, and 999 according to partner configuration.
- Reconcile acknowledgments to interchange, group, transaction, and business
  entity.
- Track expected acknowledgment windows.
- Raise exceptions for missing, late, rejected, duplicate, or invalid
  acknowledgments.
- Include acknowledgment status in partner dashboards, archive search, message
  detail, and exception queues.

## Inbound EDI

Inbound EDI must be safe by default. Parsing a document should not immediately
mutate core business state unless the partner and transaction policy explicitly
allow automation.

Requirements:

- Parse inbound X12 into envelope, group, transaction, loop, segment, and element
  structures.
- Validate syntax, control numbers, partner identity, duplicate documents,
  implementation-guide rules, partner rules, mappings, and business constraints.
- Normalize valid payloads into reviewable business commands.
- Support manual review and automated application policies by partner and
  transaction set.
- Generate acknowledgments even when business application requires manual
  review, where configured.
- Preserve raw payload and diagnostics for every inbound document.

## Exception Workbench

The exception workbench is the operational control plane for failed or risky EDI
work.

Requirements:

- Unified queues for rendering errors, Starlark errors, transform errors,
  condition errors, validation failures, mapping misses, transport failures,
  duplicate documents, rejected acknowledgments, missing acknowledgments, and
  business application failures.
- Severity, ownership, status, due date, comments, resolution reason, and audit
  history.
- Suggested fixes from diagnostics.
- Safe retry and replay actions with idempotency protection.
- Links to message archive, partner profile, template version, fixture, source
  payload, and transport attempts.

## Safety and Security Requirements

Security and safety are design constraints for every stage.

- Never expose raw secrets in scripts, diagnostics, logs, archive views, previews,
  or exports.
- Encrypt communication profile secrets and sensitive partner settings.
- Enforce role-based permissions for viewing payloads, editing templates,
  editing scripts, activating versions, replaying messages, and managing
  transport secrets.
- Require audit events for template edits, activation, rollback, replay,
  exception resolution, partner setting changes, and transport credential
  changes.
- Keep active runtime configuration immutable through version pins.
- Limit Starlark execution by timeout, step count, memory-conscious conversion,
  approved helpers, frozen context, and restricted imports.
- Validate all external inbound payloads before domain application.
- Detect duplicate control numbers and duplicate payload hashes.
- Separate test, certification, and production environments.
- Support emergency disablement at partner, capability, transaction set, and
  transport profile levels.

## Testing Expectations

Each stage must ship with tests that cover behavior, safety, and regression
surface. Tests should be scoped to the change but broad enough for shared
runtime contracts.

Required test categories:

- Unit tests for renderers, parsers, validators, transforms, conditions,
  Starlark helpers, and domain services.
- Table-driven tests for transaction-set-specific segment and element behavior.
- Fixture tests for partner-specific templates and certification examples.
- Golden-file tests for rendered X12 where stable output matters.
- Parser round-trip tests where applicable.
- Integration tests for persistence, archive writes, control numbers, transport
  attempts, acknowledgments, exceptions, and replay.
- Security tests for secret redaction, Starlark restrictions, unsafe imports,
  timeouts, step limits, and context immutability.
- UI tests for designer workflows, preview diagnostics, version diff, promotion,
  and workbench flows.
- Regression tests for every bug fixed in rendering, parsing, transport, or
  acknowledgment reconciliation.
- Load and soak tests for high-volume message archive queries, polling jobs,
  retries, and acknowledgment reconciliation.

No stage should be considered complete until its runtime behavior is covered by
automated tests, its operational failure modes are visible, and its user-facing
designer or workbench flows have validation and diagnostic coverage.
