# Document Quality Assessment Service - Dockerfile
# Multi-stage build for optimized production image

# Stage 1: Base image with Python and dependencies
FROM python:3.11-slim as base

# Set working directory
WORKDIR /app

# Install system dependencies
RUN apt-get update && apt-get install -y \
    libgl1-mesa-glx \
    libglib2.0-0 \
    libsm6 \
    libxext6 \
    libxrender-dev \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

# Stage 2: Builder image for installing Python packages
FROM base as builder

# Install build dependencies
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    && rm -rf /var/lib/apt/lists/*

# Copy requirements
COPY requirements.txt .

# Install Python dependencies
RUN pip install --no-cache-dir --user -r requirements.txt

# Stage 3: Production image
FROM base as production

# Copy Python packages from builder
COPY --from=builder /root/.local /root/.local

# Make sure scripts in .local are usable
ENV PATH=/root/.local/bin:$PATH

# Copy application code
COPY . /app

# Create directories for models and data
RUN mkdir -p /app/models /app/uploads /app/logs

# Set environment variables
ENV PYTHONUNBUFFERED=1 \
    MODEL_PATH=/app/models/best_model.pth \
    CONFIG_PATH=/app/config/best_training.yaml \
    HOST=0.0.0.0 \
    PORT=8000 \
    DEVICE=auto \
    WORKERS=1

# Expose port
EXPOSE 8000

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=40s --retries=3 \
    CMD python -c "import requests; requests.get('http://localhost:8000/health', timeout=5)"

# Run API
CMD ["uvicorn", "src.api.app:app", "--host", "0.0.0.0", "--port", "8000"]

# Stage 4: Development image (optional)
FROM production as development

ENV RELOAD=true

CMD ["uvicorn", "src.api.app:app", "--host", "0.0.0.0", "--port", "8000", "--reload"]
