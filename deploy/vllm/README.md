# vLLM Deployment

Local Docker deployment of [vLLM](https://docs.vllm.ai/) serving `Qwen/Qwen2.5-14B-Instruct` via an OpenAI-compatible API. This is the model backend for the Trenova `ai-service`.

## Prerequisites

- NVIDIA GPU with sufficient VRAM (~28 GB for 14B at fp16, ~16 GB at AWQ/GPTQ)
- Docker with [NVIDIA Container Toolkit](https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/latest/install-guide.html)
- Hugging Face account with access to the model (set `HF_TOKEN`)

## Quick Start

```bash
cd deploy/vllm
cp .env.example .env
# Edit .env — set HF_TOKEN and optionally change VLLM_API_KEY

docker compose up -d
```

First startup will download the model weights (~28 GB). Subsequent starts use the cached weights.

## API

Once running, the OpenAI-compatible API is available at:

```
http://localhost:8000/v1
```

### Health check

```bash
curl http://localhost:8000/health
```

### Test completion

```bash
curl http://localhost:8000/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer change-me" \
  -d '{
    "model": "Qwen/Qwen2.5-14B-Instruct",
    "messages": [{"role": "user", "content": "Hello"}],
    "max_tokens": 64
  }'
```

### List models

```bash
curl http://localhost:8000/v1/models \
  -H "Authorization: Bearer change-me"
```

## Connecting ai-service

The `ai-service` connects to vLLM over HTTP using the OpenAI Python SDK. Set these in the ai-service `.env`:

```
AI_SERVICE_VLLM_BASE_URL=http://localhost:8000/v1
AI_SERVICE_VLLM_API_KEY=change-me
AI_SERVICE_VLLM_MODEL=Qwen/Qwen2.5-14B-Instruct
```

If running both services in Docker on the same network, replace `localhost` with the container name (`trenova-vllm`).

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `HF_TOKEN` | (required) | Hugging Face access token |
| `VLLM_API_KEY` | `change-me` | API key for the vLLM server |
| `VLLM_MODEL` | `Qwen/Qwen2.5-14B-Instruct` | Model to serve |
| `VLLM_MAX_MODEL_LEN` | `8192` | Max sequence length |
| `VLLM_GPU_MEM_UTIL` | `0.90` | GPU memory utilization (0.0-1.0) |

## Logs

```bash
docker compose logs -f vllm
```
