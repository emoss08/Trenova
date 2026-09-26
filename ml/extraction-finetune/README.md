# Extraction fine-tuning

Fine-tunes an open, instruction-tuned model on the anonymized datasets Trenova renders from
an AI training export, and predicts the validation set so the result can be scored against
the production model.

How the pieces fit together, from export to a served model, is in
[docs/engineering/extraction-fine-tuning.md](../../docs/engineering/extraction-fine-tuning.md).

```bash
uv sync --extra train --extra predict --extra dev   # on the GPU machine
uv run trenova-finetune verify --dataset ./datasets/aitx_01J
uv run trenova-finetune run --config configs/qwen2.5-7b-instruct.yaml \
  --dataset ./datasets/aitx_01J --out ./runs/qwen-2026-10
uv run pytest                                       # anywhere; needs no GPU
```
