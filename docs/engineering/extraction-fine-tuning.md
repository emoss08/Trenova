# Extraction Fine-Tuning

How an open, instruction-tuned model is fine-tuned for Trenova's document extraction on our own
GPUs, checked against the production model, and put in front of customers. The data comes from
an [AI training export](ai-training-export.md); read that first.

```
trenova ai training-export start            anonymized JSONL in object storage
trenova ai training-export render           prompt-completion datasets on disk
trenova-finetune run                        SFT → merge → DPO → merge → predict   (GPU)
trenova ai fine-tune score                  model vs. production on the validation set
scripts/serve.sh + AI provider              vLLM behind an OpenAIChat provider
AI Control → Quality → Document extraction  evaluation run on the golden set
```

The Go side owns everything that must match production exactly: the prompt, the reply schema,
reading a reply, and scoring it. The Python side in `ml/extraction-finetune` only trains and
generates. Neither re-implements the other.

## 1. Render the export

```bash
trenova ai training-export render aitx_01J... --out ./datasets/aitx_01J
```

`aitrainingservice.Renderer` checks, then writes. Before it writes anything, it:

- verifies the manifest against the checksum the export recorded;
- verifies every part against the manifest;
- leaves out every example that `withdrawn` would list. A dataset rendered today therefore
  honours every withdrawal made since the export ran.

The output directory is staged beside its destination and moved into place only once every file
is written. A failed render leaves nothing behind.

| File | What it holds |
| --- | --- |
| `sft-train.jsonl`, `sft-validation.jsonl` | `{id, prompt: [system, user], completion: [assistant]}` |
| `preference-train.jsonl` | `{id, prompt, chosen, rejected}` for training examples a person corrected |
| `eval-validation.jsonl` | `{id, prompt, expected, baseline}`, which the scorer reads |
| `schema.json` | The reply schema a served model is held to |
| `dataset-manifest.json` | Size, SHA-256 and record count of every file. Also the export lineage, the prompt fingerprint, the structured output mode, and production's sampling settings |

**The prompt is production's prompt.**

- The system and user messages come from the same code production calls:
  `aidocumentservice.Contract.CompletionRequest` builds the extraction request, with its
  2,500-byte page limit.
- `completionrouter.PromptRenderer` renders it through the router's own `structuredRun` and
  `structuredSystem`. That includes the untrusted-data fencing, and the schema instruction a
  `Prompted` or `JSONMode` provider is given.
- `TestRenderedPromptMatchesWhatTheRouterSends` holds the two paths together.

**Choose the structured output mode now.** `--structured-output` must be the mode the model will
be registered with:

- `JSONSchema` (the default) is right for vLLM, which enforces the schema with guided decoding.
- A model trained on `JSONSchema` prompts and then served `Prompted` has never seen the schema
  instruction it is given.

**The target is the confirmed answer**, formatted by `Contract.FormatReply` into the reply schema:

- Fields a person confirmed carry confidence 0.95. Other fields the production model predicted
  carry 0.7; `--drop-unverified` leaves them out. They are included by default because a target
  without them would teach the model to omit those fields.
- Each value's evidence excerpt and page number are found in the page text the model actually
  sees. Money is matched with and without thousands separators.
- Stop dates are the confirmed calendar day in ISO form, and stops are numbered in route order.

**Preference pairs** are built only for training examples with a `Corrected` or `Missed` field.
`chosen` is the confirmed answer; `rejected` is what the production model said, rendered the
same way.

## 2. Train

On the GPU machine:

```bash
cd ml/extraction-finetune
uv sync --extra train --extra predict
uv run trenova-finetune verify --dataset ./datasets/aitx_01J
uv run trenova-finetune run --config configs/qwen2.5-7b-instruct.yaml \
  --dataset ./datasets/aitx_01J --out ./runs/qwen-2026-10
```

`run` works through these stages and records each one in `run.json`:

1. **verify**: every dataset file is checked against the manifest.
2. **sft**: LoRA on the base model, with loss on the completion only. That is TRL's default for
   prompt-completion data, set explicitly here.
3. **merge-sft**: the adapter is folded into the base model.
4. **dpo**: preference training on top of the SFT model. It is skipped when the configuration
   disables it or there are fewer than `dpo.min_pairs` pairs.
5. **merge-dpo**: the preference adapter is folded into the SFT model.
6. **predict**: vLLM answers the validation prompts, asking for JSON the way the provider will.

`run.json` records:

- the configuration, the export and dataset checksums, and the prompt fingerprint;
- the package versions used;
- each stage's outcome and metrics.

It is rewritten atomically after every stage. `--resume` continues a run that stopped, and
refuses if the configuration or dataset has changed. The final model directory also gets
`trenova-model.json`, which says how to register the model.

Examples longer than `max_length` are dropped, never truncated, because a cut-off target teaches
the model to stop mid-JSON. The run stops if more than `sft.max_dropped_fraction` of them are
dropped.

`configs/` ships two configurations for Qwen2.5-7B-Instruct: full-precision LoRA (one 48 GB+
GPU), and 4-bit QLoRA with preference training off (a 24 GB GPU). Any instruction-tuned model
with a chat template works; pin `revision` to the exact weights you evaluated.

The `train` and `predict` extras pin the major versions whose APIs this code was checked
against: trl 1, transformers 5, peft 0.x, vLLM 0.x. Raise a pin only together with a check of
`training.py`, `models.py` and `predict.py`. The pipeline's own tests need no GPU, and run in CI
(`test-ml.yml`).

## 3. Score

```bash
trenova ai fine-tune score \
  --eval ./datasets/aitx_01J/eval-validation.jsonl \
  --predictions ./runs/qwen-2026-10/predictions.jsonl \
  --max-regression 0 --min-accuracy 0.85
```

- **How replies are scored.** Each reply goes through `extractionevaljobs.ReplyReader`: the
  same parsing, draft conversion and `aicorrection.ReadPrediction` that production uses.
  `aicorrection.Score` then scores it against the confirmed answer.
- **The baseline.** Beside it, the production model's original prediction for the same
  document (the baseline) is scored the same way. Both sides see the same anonymized document.
- **Failures count.** A prediction that is missing, errored, truncated or unreadable counts as
  every field missed.
- **Gates.** The command exits non-zero when accuracy is below `--min-accuracy`, or more than
  `--max-regression` below the baseline. `--json` prints the full report, with per-field
  accuracy on both sides.

Writing this scorer turned up a bug in how AI corrections were scored. An accepted AI extraction
stores its field keys lowercased (`referencenumber`, `pickupwindow`), and the scorer only looked
up camelCase keys. As a result, those fields read as `Missed` in production corrections and in
evaluation runs. `aicorrection.CanonicalFieldKey` now maps both spellings onto one key. Accuracy
captured before this change undercounts reference numbers and pickup and delivery windows.

## 4. Serve and compare in Trenova

```bash
VLLM_API_KEY=... scripts/serve.sh ./runs/qwen-2026-10/dpo/model trenova-extract-2026-10
```

Register an AI provider for the served model with these settings:

- **Kind:** `OpenAIChat`.
- **Base URL:** `http://<host>:8000/v1`.
- **Model:** the served name.
- **API key:** the `VLLM_API_KEY`.
- **Structured output mode:** the one in `trenova-model.json`.
- **Tasks:** only `DocumentExtraction`.
- **Private network:** a GPU box on a private network also needs `allowPrivateNetwork` on the
  provider, and private-network providers enabled in `ai` config.

Give it a priority after the current extraction provider, so nothing routes to it yet.

Then start an evaluation run on the organization's golden set, from AI Control → Quality →
Document extraction, pinned to the new provider. That scores it on real, unanonymized documents
the model never trained on, since promoted corrections are never exported. Move it ahead of the
current provider only when both the offline score and the evaluation run beat production.

## Handling the data

A rendered dataset is anonymized, but it is still customer data. Keep it on the training machine:

- Delete it, and the `runs/` directory's copies, once the model is chosen.
- Before training on an old export again, render it again rather than reusing an old directory,
  so withdrawals made since are applied.
