#!/bin/bash
#
# Copyright 2023-2025 Eric Moss
# Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
# Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md

# Start Document Quality Assessment API
#
# Usage:
#   ./scripts/start_api.sh                    # Start with defaults
#   ./scripts/start_api.sh --port 8080        # Custom port
#   ./scripts/start_api.sh --dev              # Development mode with auto-reload

set -e

# Default configuration
MODEL_PATH="${MODEL_PATH:-models/best_model.pth}"
CONFIG_PATH="${CONFIG_PATH:-config/best_training.yaml}"
HOST="${HOST:-0.0.0.0}"
PORT="${PORT:-8000}"
WORKERS="${WORKERS:-1}"
RELOAD="false"
DEVICE="${DEVICE:-auto}"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --model)
            MODEL_PATH="$2"
            shift 2
            ;;
        --config)
            CONFIG_PATH="$2"
            shift 2
            ;;
        --port)
            PORT="$2"
            shift 2
            ;;
        --host)
            HOST="$2"
            shift 2
            ;;
        --workers)
            WORKERS="$2"
            shift 2
            ;;
        --dev)
            RELOAD="true"
            shift
            ;;
        --cpu)
            DEVICE="cpu"
            shift
            ;;
        --cuda)
            DEVICE="cuda"
            shift
            ;;
        --help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --model PATH      Path to model checkpoint (default: models/best_model.pth)"
            echo "  --config PATH     Path to config file (default: config/best_training.yaml)"
            echo "  --port PORT       Port to run on (default: 8000)"
            echo "  --host HOST       Host to bind to (default: 0.0.0.0)"
            echo "  --workers N       Number of worker processes (default: 1)"
            echo "  --dev             Enable development mode with auto-reload"
            echo "  --cpu             Force CPU usage"
            echo "  --cuda            Force CUDA usage"
            echo "  --help            Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  MODEL_PATH        Path to model checkpoint"
            echo "  CONFIG_PATH       Path to config file"
            echo "  HOST              Host to bind to"
            echo "  PORT              Port to run on"
            echo "  WORKERS           Number of workers"
            echo "  DEVICE            Device to use (auto/cpu/cuda)"
            echo "  CORS_ORIGINS      Allowed CORS origins (comma-separated)"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Check if model exists
if [ ! -f "$MODEL_PATH" ]; then
    echo "❌ Model not found: $MODEL_PATH"
    echo ""
    echo "Please train a model first:"
    echo "  python scripts/train_production.py"
    exit 1
fi

# Check if config exists (optional)
if [ ! -f "$CONFIG_PATH" ]; then
    echo "⚠️  Config not found: $CONFIG_PATH (will use default config)"
fi

# Print configuration
echo "================================================================================"
echo "STARTING DOCUMENT QUALITY ASSESSMENT API"
echo "================================================================================"
echo "Model:    $MODEL_PATH"
echo "Config:   $CONFIG_PATH"
echo "Host:     $HOST"
echo "Port:     $PORT"
echo "Workers:  $WORKERS"
echo "Device:   $DEVICE"
echo "Reload:   $RELOAD"
echo "================================================================================"
echo ""

# Export environment variables
export MODEL_PATH
export CONFIG_PATH
export DEVICE
export HOST
export PORT
export RELOAD

# Start API
if [ "$RELOAD" = "true" ]; then
    echo "🔄 Starting in development mode with auto-reload..."
    uvicorn src.api.app:app \
        --host "$HOST" \
        --port "$PORT" \
        --reload \
        --log-level info
else
    echo "🚀 Starting in production mode..."
    uvicorn src.api.app:app \
        --host "$HOST" \
        --port "$PORT" \
        --workers "$WORKERS" \
        --log-level info \
        --access-log
fi
