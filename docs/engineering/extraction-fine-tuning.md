# Extraction Fine-Tuning

How an open, instruction-tuned model is fine-tuned for Trenova's document extraction on our own
GPUs, checked against the production model, and put in front of customers. The data comes from
an [AI training export](ai-training-export.md); read that first.

```
trenova ai training-export start            anonymized JSONL in object storage
trenova ai training-export render           raw examples with the production prompt, on disk
trenova-finetune run                        targets → SFT → merge → DPO → merge → predict (GPU)
trenova ai fine-tune score                  model vs. production on the validation set
trenova-finetune serve / bench              vLLM tuned and timed on production-shaped requests
AI provider                                 the served model behind an OpenAIChat provider
AI Control → Quality → Document extraction  evaluation run on the golden set
```

The Go side owns everything that must match production exactly: the prompt, the reply schema,
reading a reply, and scoring it. The Python side in `ml/extraction-finetune` owns the training
recipe: how a confirmed answer becomes the reply the model learns, which examples become
preference pairs, and the training itself. Neither re-implements the other.

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
| `examples-train.jsonl`, `examples-validation.jsonl` | `{id, split, documentKind, prompt: [system, user], visiblePages, target, prediction, outcomes}` |
| `eval-validation.jsonl` | `{id, prompt, expected, baseline}`, which the scorer reads |
| `schema.json` | The reply schema a served model is held to |
| `dataset-manifest.json` | Size, SHA-256 and record count of every file. Also the export lineage, the prompt fingerprint, the structured output mode, production's sampling settings, the page limit, and the field keys the reply may use |

`visiblePages` is the page text cut where production cuts it, so evidence is only ever found in
text the model was shown. `target` is what a person confirmed, `prediction` what the production
model said, and `outcomes` how each field scored. Nothing here decides what the model is trained
to say; that is the recipe's job.

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

## 2. The training recipe

`trenova_finetune/targets.py` turns examples into training data. The `targets` section of the
configuration sets every choice it makes:

| Setting | Default | What it does |
| --- | --- | --- |
| `keep_unverified` | `true` | Also train on fields the production model predicted but nobody confirmed. Without them, the model would learn to omit those fields |
| `verified_confidence` | `0.95` | Confidence given to a confirmed field or stop |
| `unverified_confidence` | `0.7` | Confidence given to an unconfirmed field, and to every value in a rejected reply |
| `overall_confidence`, `review_status` | `0.9`, `Ready` | The reply's top-level confidence and status |
| `evidence_context_chars` | `60` | How much page text surrounds a stop name in its evidence excerpt |
| `preference_outcomes` | `[Corrected, Missed]` | A training example becomes a preference pair when a field has one of these outcomes |

A reply is built in the production wire format:

- Keys are ordered as the schema's `required` lists, with field keys in schema order.
- Every string is held to its `maxLength`, and the lists to their `maxItems`.
- Evidence is found in the visible page text, and money is matched with and without thousands
  separators.
- Stop dates are the confirmed calendar day in ISO form.
- Nothing Go derives after parsing is built (see [Keeping replies short](#keeping-replies-short)).

Every reply is validated against `schema.json` before it is written, so a recipe change that
breaks the wire format stops the build instead of reaching training. For each preference pair,
`chosen` is the confirmed reply; `rejected` is the production model's answer, built the same way.

`trenova-finetune targets --config … --dataset … --out …` builds the data on its own for
inspection. `run` builds it as its first stage, into the run directory, with its own checksummed
manifest. The recipe is part of the configuration, so `--resume` refuses a changed recipe.

The builder has two cross-checks. `tests/fixtures/extraction-schema.json` is the production reply
schema, and a Go test (`TestFineTuningPipelineSchemaFixtureIsCurrent`) fails when it goes stale.
Regenerate it with `TRENOVA_UPDATE_FIXTURES=1 go test ./internal/core/services/aidocumentservice/`.
In the other direction, `trenova ai fine-tune score` reads the recipe's replies with production's
reply reader.

## 3. Train

On the GPU machine:

```bash
cd ml/extraction-finetune
uv sync --extra train --extra predict
uv run trenova-finetune verify --dataset ./datasets/aitx_01J
uv run trenova-finetune run --config configs/qwen2.5-7b-instruct.yaml \
  --dataset ./datasets/aitx_01J --out ./runs/qwen-2026-10
```

Three configurations ship:

| Config | Base model | License | When |
|---|---|---|---|
| `qwen2.5-7b-instruct.yaml` | Qwen2.5-7B-Instruct | Apache-2.0 | The reference model |
| `qwen2.5-7b-instruct-qlora.yaml` | the same, trained in 4-bit | Apache-2.0 | A GPU without the memory for bf16 LoRA |
| `qwen3-4b-instruct-2507.yaml` | Qwen3-4B-Instruct-2507 | Apache-2.0 | About half the weights to read per token; use it if it scores within the gate |

Qwen2.5-3B-Instruct is deliberately absent. Its license (`qwen-research`) does not allow
commercial use, unlike the rest of the Qwen2.5 family. Qwen3-4B-Instruct-2507 is the
non-thinking release; the thinking variants spend output tokens on reasoning that production
would pay for on every document. Check the license of any base model before adding a config.

`run` works through these stages and records each one in `run.json`:

1. **verify**: every dataset file is checked against the manifest.
2. **targets**: the recipe builds the SFT and preference data into `data/`.
3. **sft**: LoRA on the base model, with loss on the completion only. That is TRL's default for
   prompt-completion data, set explicitly here.
4. **merge-sft**: the adapter is folded into the base model.
5. **dpo**: preference training on top of the SFT model. It is skipped when the configuration
   disables it or there are fewer than `dpo.min_pairs` pairs.
6. **merge-dpo**: the preference adapter is folded into the SFT model.
7. **predict**: vLLM answers the validation prompts, asking for JSON the way the provider will,
   with Trenova's default output ceiling of 5000 tokens (`ai.extractionMaxTokens`). A lower
   ceiling would cut off replies that production would have finished, and count them as misses.

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

## 4. Score

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

## 5. Serve, measure and tune

Almost all of an extraction's time is spent in the model, in two parts:

- **Prefill.** Reading the prompt. It sets the time to first token.
- **Decode.** Writing the reply one token at a time. It sets everything after the first token.

The reply is long structured JSON, so decode dominates. Nothing outside the model is worth
optimizing: rendering, the HTTP round trip and scoring take milliseconds per document against
seconds of generation. The speed work happens in vLLM's settings and is judged by measurement.

### Serve

```bash
VLLM_API_KEY=... uv run trenova-finetune serve --config configs/qwen2.5-7b-instruct.yaml \
  --model ./runs/qwen-2026-10/dpo/model --name trenova-extract-2026-10
```

- **The command.** It is built from the config's `serve` section (`serve.py`), so the settings
  that were benchmarked are the settings that run.
- **Safety checks.** It refuses a directory without the pipeline's `trenova-model.json`.
- **The API key.** vLLM reads it from `VLLM_API_KEY`. It is never put on the command line, where
  any user of the machine could read it from the process list. Passing `--api-key` is refused.
- **Checking the command.** `--dry-run` prints the command without running it.
- **Extra flags.** Anything after `--` is passed through to vLLM unchanged.

| Setting | Shipped | What it does for extraction |
|---|---|---|
| `enable_prefix_caching` | on | Every request starts with the same system prompt and schema, so their prefill is computed once and reused |
| `enable_chunked_prefill` | on | A long document's prefill is split so it does not stall other requests' decoding |
| `speculative` | `ngram`, 4 tokens, lookup 2–5 | Drafts tokens by finding the text just written elsewhere in the prompt. The target model then checks several drafted tokens in one pass. Extracted values are copied from the document, so drafts are often accepted. Structured outputs work with it |
| `structured_outputs_backend` | `xgrammar` | Grammar-constrained decoding for the strict `json_schema` |
| `compact_json` | on | Passed to vLLM as `disable_any_whitespace`, so the reply has no whitespace between tokens. Needs the `xgrammar` or `guidance` backend, which the config enforces |
| `max_num_seqs`, `max_num_batched_tokens` | 64, 8192 | The batch size: the trade between one document's latency and total throughput |
| `quantization` | none | `fp8` quantizes the merged model's weights when the model loads (Hopper/Ada GPUs). `awq`/`gptq` expect a model quantized ahead of time |
| `kv_cache_dtype` | `auto` | `fp8` halves the KV cache, which fits more concurrent documents |
| `report_cached_tokens` | on | Reports cached prompt tokens, so the benchmark can show prefix cache hits |

Speculative decoding, quantization and the KV-cache type can each change what the model writes,
not only how fast. Score a changed setup with `trenova ai fine-tune score` before using it.

### Keeping replies short

Output tokens dominate an extraction's time, so the reply carries only what the model has to
decide. A typical rate confirmation reply has 12 fields and 2 stops. Counted with the Qwen2.5
tokenizer:

| Reply | Tokens |
|---|---|
| Former schema, pretty-printed | 1767 |
| Former schema, compact | 1283 |
| Derived properties removed, compact | 1040 |
| Field evidence also removed (current), compact | 656 |

Everything the model used to write that Go can work out is filled in by
`aidocumentservice.convertExtractResponse` after parsing. That covers production, evaluation
runs and `trenova ai fine-tune score` alike:

| Property | Derived as | Why it is safe |
|---|---|---|
| field `label`, conflict `label` | `aicorrection.FieldLabel(key)`, the same labels the web app's `FIELD_LABELS` shows | The UI preferred the model's label over its own map, so reviewers see the same text |
| `source` on fields, stops and conflicts | `ai` | Nothing branches on it, and the AI path defaulted it to `ai` already |
| stop `sequence` | Position in the reply, from 1 | Ordering and scoring use array order, never the model's number |
| field `conflict` | The field's key appears in `conflicts` | The UI already treated either signal as a conflict |
| field `alternativeValues` | Not produced | It was dropped before storage; the UI's alternatives come from `conflicts[].values` |
| field `evidenceExcerpt`, and `pageNumber` when the model left it out | Found in the page text by `documentintelligencejobs.withFieldEvidence` when a finished extraction is applied (see below) | Reviewers see a quote of the document itself rather than the model's retelling of it |

Conflict keys are also run through `aicorrection.CanonicalFieldKey`, so a conflict on
`pickupwindow` now lines up with the `pickupWindow` field it is about.

The reply reader ignores properties it does not know, so a model fine-tuned on the former
schema is still read correctly: the dropped properties are derived as above. A schema change
changes the prompt fingerprint, so exports rendered before it must be rendered again before
training; `trenova-finetune targets` refuses a dataset whose schema asks for a property the
recipe no longer builds.

**Field evidence.** A field's excerpt is taken from the same page text the model was sent
(`toAIDocumentPages`), when the extraction is applied and before it is validated:

- The value is searched for case-insensitively, on the page the model named first and then on
  the others.
- A money value is also searched for with thousands separators (`2563.12` finds `$2,563.12`).
- The excerpt is the match with 60 bytes of text on each side, on one line, held to 200 runes.
- A field whose value the model reworded (a date written in another format, for example) has no
  excerpt; nothing else about it changes.
- An excerpt a model does write, as one trained on the former schema will, is kept.
- A page found for a field the model gave no page is filled in, which can let an extraction pass
  the required-page check it would otherwise fail.

Stop and conflict evidence stay model-written. `validateAIExtract` rejects an extraction whose
stops have no excerpt, and a stop name the model tidied up would often not be found verbatim.
The training recipe mirrors this: it still finds a field's page, and no longer writes its
excerpt.

### Benchmark

```bash
uv run trenova-finetune bench --dataset ./datasets/aitx_01J \
  --base-url http://gpu-1:8000/v1 --model trenova-extract-2026-10 \
  --concurrency 1,4,16,64 --label ngram-bf16 \
  --out ./bench/ngram-bf16.json --predictions-dir ./bench/ngram-bf16
```

What each request sends:

- The request is the one Trenova's OpenAIChat adapter sends.
- The prompt is the rendered validation prompt.
- The reply format is a strict `json_schema` named as in the dataset. It is `json_object` or
  nothing when the dataset was rendered for those modes.
- Temperature and top-p come from the dataset.
- `max_tokens` is 5000, Trenova's `ai.extractionMaxTokens` default; `--max-tokens` changes it.
- Replies are streamed with usage reporting.

How a run proceeds:

- **Warmup.** Untimed requests go first (`--warmup`). If every one fails, the run stops with the
  server's error, such as a wrong URL, key or model name.
- **Levels.** Each level then sends the same requests with that many in flight.
- **API key.** It comes from `VLLM_API_KEY` (`--api-key-env`) and is never written to the report.

What each level reports:

| Field | Meaning |
|---|---|
| `ttftMs` | Time to first token: queueing plus prefill. Mean, p50, p90, p99 and max |
| `tpotMs` | Time per output token after the first: the decode speed one document sees |
| `e2eMs` | Request to last token: what a user waiting on an upload feels |
| `outputTokensPerSecond`, `requestsPerSecond` | What the server sustains at that concurrency |
| `cachedPromptFraction` | Share of prompt tokens served from the prefix cache |
| `truncated`, `invalidReplies`, `failed`, `errors` | Replies cut off at `max_tokens`, replies that break the schema, and failed requests with their top errors. A fast setup that breaks replies is not faster |

Caveats:

- **Timing is taken on the client.** Run the benchmark near the server.
- **Prompts repeat.** The same prompts are sent in the warmup and at every level, so later
  levels find more of each document in the prefix cache than production would. Decode is
  unaffected. For a clean time to first token, run a single level against a freshly started
  server. `promptsRepeat` and `cachedPromptFraction` in the report show when this applies.

`--predictions-dir` writes each level's replies as `predictions-c<N>.jsonl`, in the format
`trenova ai fine-tune score` reads. Speed and accuracy come from the same run.

### Tune

Change one setting, restart the server, and benchmark again with the same dataset and levels:

```bash
uv run trenova-finetune bench-compare --baseline ./bench/bf16.json --candidate ./bench/ngram-bf16.json
```

`bench-compare` lines up the levels the two reports share. For each it shows:

- p50 and p90 end-to-end latency;
- p50 time to first token and time per output token;
- output tokens per second and requests per second.

Each row gives the relative change and whether it is better or worse (`--json` prints the rows
as JSON). It refuses to compare reports whose prompts, dataset or request settings differ,
because those numbers do not measure the same work.

A sensible order:

1. Prefix caching.
2. n-gram speculation, adjusting `num_speculative_tokens`.
3. Batch limits at the concurrency production sees.
4. A quantized model (below), and the `fp8` KV cache.
5. The smaller base model, trained with `configs/qwen3-4b-instruct-2507.yaml` on the same
   dataset, then quantized and benchmarked the same way.

After each change, score the replies. Keep a change only if the score holds.

### Quantize

Decode reads every weight for every token, so smaller weights make each token cheaper.
`quantize` compresses a merged model ahead of serving with llm-compressor, in the
compressed-tensors format vLLM detects on its own:

```bash
uv sync --extra train --extra quantize        # its own environment; see below
uv run trenova-finetune quantize --config configs/qwen2.5-7b-instruct.yaml \
  --model ./runs/qwen-2026-10/dpo/model --scheme w4a16 \
  --data ./runs/qwen-2026-10/data --out ./runs/qwen-2026-10/w4a16
```

| Scheme | Weights | Calibration | GPUs |
|---|---|---|---|
| `fp8-dynamic` | 8-bit float, activations scaled per token at run time | None | FP8-capable: Ada, Hopper and later |
| `w4a16` | 4-bit integer by GPTQ, groups of 128 | `quantize.calibration_samples` (512) conversations from the run's training data | Any GPU vLLM supports |

- **Calibration data.** `w4a16` calibrates on whole training conversations: the production
  prompt followed by the reply the model was trained to give. The error GPTQ minimizes is the
  error on extraction, not on generic chat. The sample is drawn with the config's seed, so it is
  reproducible.
- **The output.** The model card is copied with a `quantization` entry: the scheme, method,
  calibration size, source directory and llm-compressor version.
- **Refusals.** `quantize` refuses a model that is already quantized, and an output directory
  that holds files.
- **Serving it.** Serve the output with `serve.quantization: null`, because vLLM reads the
  scheme from the model. `serve` refuses to quantize an already-quantized model a second time.
- **A separate environment.** llm-compressor 0.14 pins `compressed-tensors` 0.19, and vLLM 0.30
  pins 0.17. The `predict` and `quantize` extras are declared as conflicting, so each is
  installed on its own. The two versions write and read the same quantization config and
  compression formats, so a model quantized with one is served by the other.

`fp8` in the `serve` section quantizes at load time with no step ahead. `fp8-dynamic` produces
the same kind of weights once, which the benchmark and scorer can then judge as a fixed
artifact. Score every quantized model before serving it: quantization changes what a model
writes.

### Register the provider in Trenova

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
