# AI Corrections

`ai_corrections` records what an AI extraction predicted beside what a person confirmed
when they turned the prediction into a record. It is the organization's own measure of
extraction accuracy and its evaluation set, and it is the only source a model-training
export may read.

## What is captured

One task today, `ShipmentDraftExtraction`: a document shipment draft (read from a rate
confirmation or a bill of lading) against the shipment a person created from it.

Capture runs in `documentservice.AttachLineageToResource`, after the draft is marked as
attached to the shipment. It is best effort: a failure is logged and never fails the
attach. `aicorrectionservice.CaptureShipmentDraft`:

1. reads the draft's `DraftData` (fields and stops, each with its source and confidence);
2. loads the shipment with its moves, stops, locations, states and commodities;
3. scores each field and each stop field, and stores one row per draft
   (`uq_ai_corrections_source`; attaching the draft to another shipment replaces it);
4. records the model and provider when the document's latest AI extraction completed.

A draft with nothing predicted is skipped with `aicorrection.ErrNothingPredicted`.

| Outcome | Meaning |
| --- | --- |
| `Correct` | Both sides present and equal after normalization |
| `Corrected` | Both present, and the person changed it |
| `Missed` | Not predicted, but the person entered it |
| `Unconfirmed` | Predicted, but left empty on the record: neither right nor wrong |
| `Unscored` | Both present, but the prediction could not be read (a date like "FCFS") |

`scored_count` is correct + corrected + missed, so accuracy is `correct_count / scored_count`.
The shipment form is pre-filled from the draft, so `Correct` means "not changed", not
"checked". Treat it as an upper bound.

Comparison rules live in `aicorrectionservice/compare.go`: identifiers ignore case and
punctuation, names also match when one contains the other (at least four characters), money
compares to the cent, weights and pieces compare their first integer, postal codes compare
five digits, and dates match the stop's scheduled day in the location's timezone or UTC.

## Consent

`agent_controls.ai_training_consent` is off by default. Only a signed-in person can change
it (an API key cannot), and each change records `ai_training_consent_changed_at` and
`ai_training_consent_changed_by_id` and is audited as "AI training consent granted" or
"withdrawn". It is set on AI Control's overview.

Corrections are captured whether or not consent is on: without it they stay inside the
tenant and serve only that organization's accuracy and evaluations.

Any export for training must:

- read consent **at export time**, so withdrawing it excludes every existing row from later
  exports;
- anonymize before anything leaves the tenant. `predicted`, `confirmed`, `field_results` and
  `document_fingerprint` hold names, addresses, rates and reference numbers;
- record what it exported, so a withdrawal can be shown to have been honoured.

## Retention

`data_retention.ai_correction_retention_period`: 730 days by default, at least 30, zero reads
as the default. Set on the Data Retention page. `AICorrectionRetentionWorkflow` runs nightly
at 02:40 UTC on the system queue and purges each organization in batches.
