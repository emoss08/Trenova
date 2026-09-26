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

Comparison rules live in `aicorrection.Score` (`domain/aicorrection/score.go`), shared by capture and
by evaluation runs: identifiers ignore case and
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

The training export that does this is described in
[ai-training-export.md](ai-training-export.md). Any export for training must:

- read consent **at export time**, so withdrawing it excludes every existing row from later
  exports;
- anonymize before anything leaves the tenant. `predicted`, `confirmed`, `field_results` and
  `document_fingerprint` hold names, addresses, rates and reference numbers;
- record what it exported, so a withdrawal can be shown to have been honoured.

## Retention

`data_retention.ai_correction_retention_period`: 730 days by default, at least 30, zero reads
as the default. Set on the Data Retention page. `AICorrectionRetentionWorkflow` runs nightly
at 02:40 UTC on the system queue and purges each organization in batches.

## Production accuracy

`extractionAccuracy(windowDays)` reads the corrections captured in the last 1–365 days and
reports overall accuracy, accuracy per field (stop fields grouped across stops, so
`stops.pickup.city`), and accuracy grouped by model and by document kind. An empty model means
the draft was read by rules alone. When the window holds more corrections than one report reads,
the report says `sampled`. These numbers inherit the upper-bound caveat above.

## Evaluation set

Production accuracy tells you how the current setup did on last month's documents. The
evaluation set answers a different question: how would *this* model do on the *same* documents?
It is stored in three tables (migration `20261231006840_extraction_eval`).

### Cases (`extraction_eval_cases`)

A case is a frozen test: the page text the extractor reads, and the snapshot a person confirmed.
`PromoteCorrection` builds one from a correction:

- the correction must still have its document, and a correction becomes at most one case;
- the document's extracted page text is copied into the case (at most 100 pages of 20,000
  characters each), so a later re-OCR, edit or deletion of the document never changes the test;
- `expected` is the correction's `confirmed` snapshot;
- `input_hash` covers the task, file name and pages, so identical inputs can be recognised.

A case is `Candidate` (added, not yet run), `Active` (run by every evaluation) or `Retired`
(kept for history, no longer run). Review a candidate's expected values before activating it:
it is only as good as what the person confirmed. Deleting a case never changes past results,
which keep the case's title.

### Runs (`extraction_eval_runs`, `extraction_eval_results`)

A run evaluates one AI provider over up to `case_limit` active cases (50 by default, at most
500). The provider must be enabled and allowed to serve `DocumentExtraction`. One organization
has at most one queued or running run at a time.

`ExtractionEvalRunWorkflow` runs on the document-intelligence queue. For each pending case, it:

1. checks the run is still active and the organization's evaluation budget, which is the same
   budget agent evaluation suites draw on (`agentqualityservice.CheckEvaluationBudget`). A spent
   budget ends the run as `BudgetStopped`;
2. calls `aidocumentservice.ExtractRateConfirmationForEvaluation` with the case's frozen pages.
   The call is pinned to the chosen provider (`completionrouter` `RequireProvider`): if the
   provider cannot serve the call, the case fails rather than quietly falling back to another
   model. Usage is recorded with the evaluation purpose;
3. scores the prediction against the case's `expected` snapshot with `aicorrection.Score`, and
   records the model served, latency, tokens and cost.

A failure that can be retried is retried by Temporal; the last attempt records the case as
`Failed` and moves on. The workflow continues as new after a fixed number of cases, so a long run keeps
a bounded history. Cancelling stops the run before its next case.

The run's accuracy is correct over scored across every completed case, with per-field accuracy
alongside. Unlike production accuracy it is not an upper bound: the model never saw the
person's answer.

### Permissions and retention

Reading the set and runs needs `AgentEvalSuite` read; promoting a correction and starting a run
need create; editing, retiring, deleting a case and cancelling a run need update. Every change
is audited.

Runs and their results older than the correction retention period are purged by the same nightly
`AICorrectionRetentionWorkflow`. Cases are not purged: they are the organization's own test set
and stay until someone deletes them. The case holds document text, so the same export rules as
corrections apply to it: consent at export time, and anonymization before anything leaves the
tenant.
