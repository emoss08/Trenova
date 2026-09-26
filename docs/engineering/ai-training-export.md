# AI Training Export

A training export turns the AI corrections of the organizations that consented into an
anonymized dataset for fine-tuning a document-extraction model. Trenova operators run it. No
customer can start one, and nothing in a customer's API reaches the files. A customer sees only
their consent switch and a list of the exports that included their data.

Read [ai-corrections.md](ai-corrections.md) first: that doc covers what a correction is and how
consent is recorded.

## Running one

```bash
trenova ai training-export start --from 2026-01-01 --requested-by "Jordan Lee" --note "extraction v2"
trenova ai training-export list
trenova ai training-export status <export-id>
trenova ai training-export cancel <export-id>
trenova ai training-export withdrawn <export-id> --output withdrawn.tsv
```

- The window runs from `--from` (inclusive) to `--to` (exclusive), as UTC dates. `--to`
  defaults to now and cannot be in the future.
- `--max-per-org` (default 2,000, at most 20,000) stops one large customer from dominating the
  dataset.
- `--validation-percent` (default 10, at most 50) sets how much is held out. The split is a hash
  of the export and the correction, so a retry puts every example back in the same split.
- One export runs at a time (`uq_ai_training_exports_active`).
- The command builds only the graph it needs (`bootstrap.TrainingExportCommandOptions`):
  config, the database, object storage and a Temporal client. `AITrainingExportWorkflow` does
  the work on `system-queue`, so a worker must be running.

## What is read

For each organization whose `agent_controls.ai_training_consent` is on, in organization-id
order (`aitrainingjobs` pages through them and continues as new every 25):

1. Consent is read again when the organization's turn comes, and skipped if it is off.
2. Corrections are read oldest first. They must:
   - be captured inside the window;
   - have a document;
   - have at least one scored field;
   - not be the source of an extraction evaluation case. Training on the evaluation set would
     make every later evaluation run meaningless.
3. For each correction, the page text is read again from `document_content_pages` (at most
   100 pages of 20,000 characters). The target is the correction's `confirmed` snapshot, and
   the prediction is its `predicted` snapshot.
4. After the organization's files are written, consent is read a third time. If consent was
   withdrawn, or re-granted with a different `ai_training_consent_changed_at`, the files are
   deleted, the records removed, and every example is counted as `consentWithdrawn`.

A correction is left out, and counted in `dropped`, for any of these reasons:

| Reason | Meaning |
| --- | --- |
| `documentUnreadable` | The document is gone. |
| `noDocumentText` | No page has extracted text. |
| `nothingConfirmed` | The confirmed snapshot is empty. |
| `residualIdentifier` | A known value survived anonymization; see below. |
| `consentWithdrawn` | Consent changed while the organization was being exported. |

## Anonymization

`aitraining.Anonymize` runs inside the worker, before anything is written, and no model is
called. It replaces the values Trenova knows and the patterns below; an identifier it does not
know can survive in free text (see **Known limits**). Each example gets its own random
surrogates, from a fresh ChaCha8 seed. The same original becomes the same surrogate throughout one example, in the
page text, the target and the prediction alike. Across examples it does not, so examples cannot
be linked by a shared surrogate.

1. **Addresses on the web**:
   - emails become `first.last@example.com`;
   - URLs become `https://www.example.com`;
   - bare domains become `example.com`.
2. **Known values**, replaced consistently. Longest matches go first, with case-insensitive
   matching and 1–3 separator characters allowed between tokens.
   - **Parties**:
     - shipper, consignee, bill-to and carrier names;
     - the name on every stop;
     - the document's issuer;
     - the organization's and business unit's names.

     They become an invented company name. Legal suffixes (`Inc`, `LLC`, …) are matched too
     and kept.
   - **People**: the names of every member of the organization, and the carrier contact.
   - **Identifiers**:
     - reference, load, BOL, PO, PRO, pickup, appointment, container, trailer, tractor and
       seal numbers;
     - SCAC and DOT;
     - the organization's tax ID and login slug.

     The replacement keeps each character's class (a letter stays a letter, a digit stays a
     digit), and punctuation keeps its place.
   - **Streets, cities and ZIP codes** of every stop and of the organization. `Road` also
     matches `Rd`, `East` matches `E`, and so on.
   - **Money**: the rate and fuel surcharge are multiplied by one factor per example, drawn
     from 0.85–1.15 but never within 1% of 1. Every amount in the text is scaled by the same
     factor and rounded to the cent, so line items still add up to the total within a cent.
3. **Patterns**:
   - tax IDs and SSNs;
   - MC, DOT, MX and FF numbers;
   - phone numbers;
   - `$` and `USD` amounts;
   - PO boxes and street addresses;
   - a ZIP code after a state code;
   - runs of six or more digits (unless they read as a date like `20260314`);
   - mixed letter-and-digit tokens of six or more characters (unless they are a quantity like
     `48000LBS`).
4. **`piiscrub`**: card, routing, account and IBAN numbers are masked.
5. **Residual check**: every known value of four or more characters is searched for in the
   output, token by token, and identifiers of six or more characters are also searched with
   punctuation removed. If any survives, the whole example is dropped as `residualIdentifier`.

Kept as they are: states, dates and time windows, weights, piece counts, commodity,
equipment, and the document kind. The file name becomes `document.<ext>`.

**Known limits.** Replacement works from the values Trenova already knows, plus the patterns
above. A person's name that appears only in free text, and belongs to nobody in the
organization or on the shipment, is not caught. The same goes for an unlabelled free-text
street without a number. Review a sample of each export before it is used, and grow the rules
from what you find. Each new rule needs a case in `anonymizer_internal_test.go`.

## What is written

All objects go under `ai-training-exports/<export-id>/` in the platform bucket:

- `parts/<ordinal>-train.jsonl` and `parts/<ordinal>-validation.jsonl`, one per organization
  and split. The ordinal is the organization's position in the export, never its id.
- `manifest.json` (`trenova.extraction-training-manifest/v1`), which holds:
  - the window, the limits and the counts;
  - the drops by reason;
  - each part with its size and SHA-256;
  - the anonymization method.

  It never includes an organization's identity.

Each line of a part is one `trenova.extraction-training/v1` example:

```json
{
  "id": "5d0f…",
  "format": "trenova.extraction-training/v1",
  "task": "ShipmentDraftExtraction",
  "split": "train",
  "documentKind": "RateConfirmation",
  "input": { "fileName": "document.pdf", "pages": [{ "number": 1, "text": "…" }] },
  "target": { "fields": { "referenceNumber": "…", "rate": "…" }, "stops": [ … ] },
  "prediction": { "fields": { … }, "stops": [ … ] },
  "outcomes": { "referenceNumber": "Correct", "stops.pickup[0].city": "Corrected" },
  "quality": { "scored": 9, "correct": 7, "corrected": 1, "missed": 1, … }
}
```

- `target` is what a person confirmed, so it is the supervised answer. Only the fields in the
  correction's scope are there.
- `prediction` and `outcomes` let the same export feed preference training: a `Corrected`
  field is a rejected answer beside its confirmed one.
- The production prompt is not stored in the export. `trenova ai training-export render` builds
  it from the current production code when the datasets are made, so a prompt change does not
  require a re-export. See [extraction-fine-tuning.md](extraction-fine-tuning.md).

## What is recorded

- `ai_training_exports` is one row per export, and is platform data with no tenant key.
  - `progress` holds one entry per organization, keyed by ordinal, so a retried organization
    replaces its entry instead of adding to it.
  - `parts` and `dropped` are totals derived from `progress`.
- `ai_training_export_records` holds one row per example, keyed by tenant. Each row records:
  - the correction;
  - the `example_id` written to the files;
  - the split;
  - `consent_granted_at`, the consent grant in force when the example was read.

`aiTrainingExportHistory` (with `AgentControl` read permission) lists every export that kept any
of an organization's examples, whatever the export's final status: a canceled or failed export
still holds the files of the organizations it finished. It shows under the consent switch on AI Control's overview.

## Withdrawal

Withdrawing consent keeps an organization out of every export that starts after the change.
An organization that withdraws mid-export is removed from that export too; see "What is read".

Files already written cannot be recalled. Instead, before training on an older export, run
`trenova ai training-export withdrawn <export-id>` and remove the listed example ids. The
command lists every example whose organization:

- has consent off now;
- changed its consent after the export read it;
- or no longer exists.

The records are kept even after the corrections behind them are purged, so this list can
always be produced.
