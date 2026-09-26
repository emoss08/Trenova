#!/usr/bin/env bash
# Serve a fine-tuned extraction model with vLLM's OpenAI-compatible server.
#
# Usage: scripts/serve.sh <model-dir> <served-model-name> [extra vllm serve args...]
#
# Register the result in Trenova as an AI provider of kind OpenAIChat whose
# base URL is http://<host>:${PORT:-8000}/v1, whose model is <served-model-name>,
# and whose structured output mode is the one in <model-dir>/trenova-model.json.
set -euo pipefail

if [[ $# -lt 2 ]]; then
  echo "usage: $0 <model-dir> <served-model-name> [vllm serve args...]" >&2
  exit 2
fi

model_dir=$1
served_name=$2
shift 2

if [[ ! -f "${model_dir}/trenova-model.json" ]]; then
  echo "error: ${model_dir} is not a model this pipeline produced (no trenova-model.json)" >&2
  exit 2
fi
if [[ -z "${VLLM_API_KEY:-}" ]]; then
  echo "error: set VLLM_API_KEY; the key Trenova's provider sends as its API key" >&2
  exit 2
fi

exec vllm serve "${model_dir}" \
  --served-model-name "${served_name}" \
  --host "${HOST:-0.0.0.0}" \
  --port "${PORT:-8000}" \
  --max-model-len "${MAX_MODEL_LEN:-16384}" \
  --api-key "${VLLM_API_KEY}" \
  --generation-config vllm \
  "$@"
