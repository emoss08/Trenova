# Extraction fine-tuning

Fine-tunes an open, instruction-tuned model on the anonymized examples Trenova renders from
an AI training export. The training recipe — how a confirmed answer becomes the reply the
model learns, and which examples become preference pairs — lives here, in `targets.py`, and is
configured by the `targets` section of each config. The run predicts the validation set so the
result can be scored against the production model.

How the pieces fit together, from export to a served model, is in
[docs/engineering/extraction-fine-tuning.md](../../docs/engineering/extraction-fine-tuning.md).

```bash
uv sync --extra train --extra predict --extra dev   # on the GPU machine
uv run trenova-finetune verify --dataset ./datasets/aitx_01J
uv run trenova-finetune targets --config configs/qwen2.5-7b-instruct.yaml \
  --dataset ./datasets/aitx_01J --out ./runs/inspect-targets   # optional: inspect the recipe
uv run trenova-finetune run --config configs/qwen2.5-7b-instruct.yaml \
  --dataset ./datasets/aitx_01J --out ./runs/qwen-2026-10
VLLM_API_KEY=... uv run trenova-finetune serve --config configs/qwen2.5-7b-instruct.yaml \
  --model ./runs/qwen-2026-10/dpo/model --name trenova-extract-2026-10
uv run trenova-finetune bench --dataset ./datasets/aitx_01J --base-url http://gpu-1:8000/v1 \
  --model trenova-extract-2026-10 --label ngram --out ./bench/ngram.json --predictions-dir ./bench/ngram
uv run trenova-finetune bench-compare --baseline ./bench/base.json --candidate ./bench/ngram.json
uv run pytest                                       # anywhere; needs no GPU
```

`serve` runs vLLM with the config's `serve` section (prefix caching, n-gram speculative decoding,
batch limits, quantization). `bench` times production-shaped extraction requests against it, and
`bench-compare` shows whether a serving change made things faster. The replies `bench` writes are
scored with `trenova ai fine-tune score`, so no change is kept that costs accuracy.
