EDI (X12) Utilities for Trenova

Overview

- Separate, asynchronous EDI parsing utilities kept outside the core app.
- Initial focus: inbound 204 (Motor Carrier Load Tender) in X12 004010 style.

Quick Start

- Build CLI: `GOWORK=off GOCACHE=$(pwd)/.gocache go build ./cmd/edi-cli`
- Parse into generic segments: `./edi-cli testdata/204/sample1.edi`
- Emit minimal typed 204 JSON: `./edi-cli --format 204 testdata/204/sample1.edi`
- Emit Shipment DTO JSON: `./edi-cli --format shipment testdata/204/sample1.edi`
- Emit multiple shipments (NDJSON): `./edi-cli --format shipment --multi ndjson testdata/204/multi_tx.edi`
- Include validation issues: add `--validate` to either command
- Emit a 997 ACK (004010): add `--validate --ack` to print a simple 997 EDI acknowledgment instead of JSON
- Override delimiters (if partner uses custom):
  - Combined: `--delims elem,comp,seg,rep` (rep optional)
  - Individual: `--element "`" --component "<" --segment "~"`

Partner Profiles

- Define partner-specific defaults in a JSON profile:
  - Path to a JSON schema for rules.
  - Default delimiters (element, component, segment, repetition).
  - Validation toggles (strict/lenient, SE count enforcement, pickup/delivery requirement, etc.).
- Reference mapping: map DTO reference keys to L11 qualifiers.
  - Party role mapping: map DTO roles (shipper/consignee/bill_to) to N1 codes.
  - Stop type mapping: map S5 type codes to pickup/delivery/other.
  - Shipment ID mapping: prefer certain L11 qualifiers before B2-03.
  - Shipment ID mode: `ref_first|b2_first|ref_only|b2_only`.
  - Carrier SCAC fallback: fixed SCAC if B2-02 is blank.
  - Raw L11 audit: include all (or filtered) L11 qualifiers/values in shipment output.
  - Equipment type normalization: map raw equipment type to normalized values.
- Example: `testdata/profiles/meritor-4010.json`
- Use in CLI: `./edi-cli --profile testdata/profiles/meritor-4010.json --format 204 --validate <file.edi>`
- Use in scanner: `./edi-scan -dir <dir> -profile testdata/profiles/meritor-4010.json`
- Emit NDJSON shipments while scanning: `./edi-scan -dir <dir> -profile testdata/profiles/meritor-4010.json -out shipments.ndjson`
- Emit per-transaction NDJSON (scanner): `./edi-scan -dir <dir> -out shipments.ndjson -per-tx`
- Precedence of delimiters: detect from ISA, then apply profile overrides, then apply CLI flags.

Generic Schemas

- A generic 204 (004010) schema with code sets is included at `testdata/schema/generic-204-4010.json`.
- If `--validate` is set and no schema is provided via `--schema` or profile, the CLI uses the generic 004010 schema when GS08=004010.
- Includes common code sets: B2A-01 (00,01,05), N1-01 (BT,SH,ST,CN,SF), S5-02 (LD,UL,CL,CU), presence of SH/ST and LD/UL.

Streaming Parser

- You can iterate segments from an `io.Reader` without loading the whole file:
  - Example:
    - `r, _ := os.Open("file.edi")`
    - `d, _ := x12.DetectDelimiters(firstBytes)`
    - `sc := x12.NewSegmentScanner(r, d)`
    - `for sc.Next() { s := sc.Segment(); /* process s */ }`

Reference Mapping

- In a partner profile, add a `references` object mapping DTO fields to L11 qualifier(s):
  - Example:
    - `"references": { "customer_po": ["PO"], "bill_of_lading": ["BM"], "shipment_ref": ["SI", "CR"] }`
- When emitting `--format shipment`, the mapper picks the first available value for each key from `L11` segments.

Mapping Behavior (Configurable)

- `party_roles`: prioritize which N1 codes populate DTO roles.
  - Example: `{ "shipper": ["SH", "SF"], "consignee": ["ST", "CN"], "bill_to": ["BT"] }`
- `stop_type_map`: normalize S5 types.
  - Example: `{ "LD": "pickup", "CL": "pickup", "UL": "delivery", "CU": "delivery" }`
- `shipment_id_quals`: qualifiers to source `shipment_id` from L11 before falling back to B2-03.
  - Example: `["CR", "SI"]`
- `shipment_id_mode`: how to choose `shipment_id`.
  - One of: `ref_first` (default), `b2_first`, `ref_only`, `b2_only`.
- `carrier_scac_fallback`: when `B2-02` is blank, set this fixed value.
- `include_raw_l11`: if true, emit `references_raw` with qualifier→values.
- `raw_l11_filter`: optional list of qualifiers to include; empty means all.
- `equipment_type_map`: map raw equipment types to normalized strings (e.g., `{"VEH":"trailer","53_FT_DRY":"dry_van"}`).
- `emit_iso_datetime`: when true, emit ISO-8601 `datetime` on appointments.
- `timezone`: IANA timezone for datetime normalization (default `UTC`).
- `service_level_quals`: list of L11 qualifiers to pick service level (first present wins).
- `service_level_map`: normalize raw service level values to canonical names.
- `accessorial_quals`: list of L11 qualifiers; all values become accessorial codes.
- `accessorial_map`: normalize accessorial codes to canonical names (used for DTO `accessorials[].name`).

What’s Implemented (MVP)

- Delimiter detection (element, component, segment) from ISA.
- Streaming-friendly segment splitter (current impl materializes into memory; stream later).
- Minimal 204 typed model (B2/B2A/L11, header N1s, S5+DTM stops, N7 equipment).
- Shipment DTO + mapper from 204 for TMS ingestion (SCAC, ShipmentID, parties, stops, equipment, notes, common refs).
- Totals (weight/unit/pieces from AT8/L3) and commodities (L5) mapped into DTO `totals` and `goods`.
- Sample 204 fixture and a smoke test.
- Basic 204 validator with envelope checks, B2 required fields, S5 type/sequence, DTM format, and counts.
- JSON schema-driven validation for partner-specific rules (segment/element/presence/conditional).

Roadmap (Short Term)

- Formal 204 schema + validation (required segments, code sets).
- Partner profiles (version, mapping, tolerances) and config loader.
- Mapping to Shipment DTO with code normalization.
- Error model with segment index and hints.
- 997/999 acknowledgements (optional).

Repo Layout

- `cmd/edi-cli/` – simple CLI for parsing and JSON output.
- `internal/x12/` – delimiters + segment parsing.
- `internal/specs/x12/004010/` – 204 schema scaffolding.
- `internal/tx/tx204/` – typed 204 model + builder.
- `internal/dto/` – Shipment DTO for TMS ingestion.
- `internal/mapper/` – mappers from typed 204 to DTOs.
- `testdata/204/` – sample EDI files.
- `testdata/schema/generic-204-4010.json` – generic 004010 schema with common codes and presence rules.

Notes

- Keep this module autonomous to decouple ingestion and scale independently of the core app.
- Parser favors correctness and debuggability; performance tuning will follow.

gRPC Config Service

- Service: The TMS hosts a gRPC service to serve partner config profiles to the EDI parser.
- Proto: `proto/config/v1/config.proto` (generate with `buf generate`).
- Client: `client/configclient` provides a Go client with optional TLS/mTLS and auth headers.
- Example:
  - Dial: `c, _ := configclient.Dial(ctx, configclient.DialOptions{ Address: ":9090", Insecure: true })`
  - Fetch: `cfg, _ := c.Get(ctx, "", buID, orgID, "partner-name")`
