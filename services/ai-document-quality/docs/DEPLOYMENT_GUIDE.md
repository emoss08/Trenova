# Document Quality Assessment - Deployment Guide

Complete production deployment guide for the document quality assessment service.

## Table of Contents

1. [Pre-Deployment Checklist](#pre-deployment-checklist)
2. [Local Development](#local-development)
3. [Docker Deployment](#docker-deployment)
4. [Production Deployment](#production-deployment)
5. [Monitoring & Maintenance](#monitoring--maintenance)
6. [Troubleshooting](#troubleshooting)

---

## Pre-Deployment Checklist

Before deploying to production, ensure:

### Model Readiness

```bash
# 1. Model is trained and evaluated
ls -lh models/best_model.pth

# 2. Evaluation metrics are acceptable
cat evaluation_results/best_model_evaluation_report.txt

# Key metrics to check:
# - Balanced Accuracy > 0.85
# - Calibration ECE < 0.10
# - False Reject Rate < 0.10
# - False Accept Rate < 0.05
```

### Infrastructure Requirements

**Minimum:**

- CPU: 2 cores
- RAM: 4GB
- Storage: 2GB
- Python: 3.11+

**Recommended (with GPU):**

- CPU: 4 cores
- RAM: 8GB
- GPU: NVIDIA GPU with 4GB+ VRAM
- Storage: 5GB
- CUDA: 11.8+

### Dependencies

```bash
# Verify all dependencies are installed
pip install -r requirements.txt

# Test model loading
python -c "
from src.api.inference import DocumentQualityPredictor
predictor = DocumentQualityPredictor('models/best_model.pth')
print('✓ Model loads successfully')
"
```

---

## Local Development

### Quick Start

```bash
# 1. Clone and setup
cd services/ai-document-quality
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

# 2. Train model (if not already done)
python scripts/train_production.py

# 3. Start API in development mode
./scripts/start_api.sh --dev

# 4. Test in another terminal
curl http://localhost:8000/health
```

### Development Workflow

```bash
# Start API with auto-reload
./scripts/start_api.sh --dev --port 8000

# In another terminal, run tests
pytest tests/ -v

# Or use docker for development
docker-compose --profile dev up api-dev
```

### Testing the API

```bash
# Health check
curl http://localhost:8000/health

# Analyze a test document
curl -X POST "http://localhost:8000/analyze" \
  -F "file=@test_document.jpg" \
  | jq '.'

# Check metrics
curl http://localhost:8000/metrics | jq '.'
```

---

## Docker Deployment

### Build and Run

```bash
# Build image
docker build -t doc-quality-api:2.0.0 .

# Run container
docker run -d \
  --name doc-quality-api \
  -p 8000:8000 \
  -v $(pwd)/models:/app/models:ro \
  -v $(pwd)/config:/app/config:ro \
  -e DEVICE=cuda \
  doc-quality-api:2.0.0

# Check logs
docker logs -f doc-quality-api

# Test
curl http://localhost:8000/health
```

### Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f api

# Scale workers (if using multiple instances)
docker-compose up -d --scale api=3

# Stop
docker-compose down
```

### Environment Configuration

Create `.env` file:

```env
# Model configuration
MODEL_PATH=/app/models/best_model.pth
CONFIG_PATH=/app/config/best_training.yaml

# Server configuration
PORT=8000
HOST=0.0.0.0
WORKERS=4
DEVICE=cuda

# CORS
CORS_ORIGINS=https://app.trenova.com,https://api.trenova.com

# Resource limits
CPU_LIMIT=4.0
MEMORY_LIMIT=8G
CPU_RESERVATION=2.0
MEMORY_RESERVATION=4G

# MLflow (optional)
MLFLOW_PORT=5000
```

Run with environment file:

```bash
docker-compose --env-file .env up -d
```

---

## Production Deployment

### Option 1: Direct Deployment (VM/Bare Metal)

#### Setup

```bash
# 1. Create service user
sudo useradd -r -s /bin/false docquality

# 2. Install dependencies
sudo apt-get update
sudo apt-get install -y python3.11 python3-pip

# 3. Setup application
sudo mkdir -p /opt/doc-quality-api
sudo chown docquality:docquality /opt/doc-quality-api
cd /opt/doc-quality-api

# 4. Install Python packages
sudo -u docquality python3.11 -m venv .venv
sudo -u docquality .venv/bin/pip install -r requirements.txt

# 5. Copy model and config
sudo cp models/best_model.pth /opt/doc-quality-api/models/
sudo cp config/best_training.yaml /opt/doc-quality-api/config/
sudo chown -R docquality:docquality /opt/doc-quality-api
```

#### Systemd Service

Create `/etc/systemd/system/doc-quality-api.service`:

```ini
[Unit]
Description=Document Quality Assessment API
After=network.target

[Service]
Type=simple
User=docquality
Group=docquality
WorkingDirectory=/opt/doc-quality-api
Environment="PATH=/opt/doc-quality-api/.venv/bin"
Environment="MODEL_PATH=/opt/doc-quality-api/models/best_model.pth"
Environment="CONFIG_PATH=/opt/doc-quality-api/config/best_training.yaml"
Environment="DEVICE=cuda"
Environment="WORKERS=4"
ExecStart=/opt/doc-quality-api/.venv/bin/uvicorn src.api.app:app \
    --host 0.0.0.0 \
    --port 8000 \
    --workers 4

Restart=always
RestartSec=10

# Resource limits
LimitNOFILE=65536
MemoryMax=8G
CPUQuota=400%

[Install]
WantedBy=multi-user.target
```

Start service:

```bash
# Enable and start
sudo systemctl enable doc-quality-api
sudo systemctl start doc-quality-api

# Check status
sudo systemctl status doc-quality-api

# View logs
sudo journalctl -u doc-quality-api -f
```

#### Nginx Reverse Proxy

Create `/etc/nginx/sites-available/doc-quality-api`:

```nginx
upstream doc_quality_api {
    least_conn;
    server 127.0.0.1:8000 max_fails=3 fail_timeout=30s;
    keepalive 32;
}

server {
    listen 80;
    server_name api.trenova.com;

    # Redirect to HTTPS
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name api.trenova.com;

    # SSL configuration
    ssl_certificate /etc/letsencrypt/live/api.trenova.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.trenova.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;

    # Large file uploads
    client_max_body_size 20M;

    # Logging
    access_log /var/log/nginx/doc-quality-api-access.log;
    error_log /var/log/nginx/doc-quality-api-error.log;

    location / {
        proxy_pass http://doc_quality_api;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # Timeouts
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
    }

    # Health check endpoint (no auth)
    location /health {
        proxy_pass http://doc_quality_api/health;
        access_log off;
    }
}
```

Enable and restart:

```bash
sudo ln -s /etc/nginx/sites-available/doc-quality-api /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

### Option 2: Kubernetes Deployment

#### Deployment Manifest

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: doc-quality-api
  namespace: trenova
spec:
  replicas: 3
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: doc-quality-api
  template:
    metadata:
      labels:
        app: doc-quality-api
        version: v2.0.0
    spec:
      containers:
      - name: api
        image: registry.trenova.com/doc-quality-api:2.0.0
        ports:
        - containerPort: 8000
          name: http
        env:
        - name: MODEL_PATH
          value: "/app/models/best_model.pth"
        - name: CONFIG_PATH
          value: "/app/config/best_training.yaml"
        - name: DEVICE
          value: "cuda"
        - name: WORKERS
          value: "2"
        resources:
          requests:
            memory: "2Gi"
            cpu: "1000m"
            nvidia.com/gpu: "1"
          limits:
            memory: "4Gi"
            cpu: "2000m"
            nvidia.com/gpu: "1"
        livenessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
          timeoutSeconds: 5
          failureThreshold: 3
        readinessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 10
          periodSeconds: 5
          timeoutSeconds: 3
        volumeMounts:
        - name: model-volume
          mountPath: /app/models
          readOnly: true
        - name: config-volume
          mountPath: /app/config
          readOnly: true
      volumes:
      - name: model-volume
        persistentVolumeClaim:
          claimName: doc-quality-models
      - name: config-volume
        configMap:
          name: doc-quality-config
---
apiVersion: v1
kind: Service
metadata:
  name: doc-quality-api
  namespace: trenova
spec:
  type: ClusterIP
  ports:
  - port: 80
    targetPort: 8000
    protocol: TCP
    name: http
  selector:
    app: doc-quality-api
---
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: doc-quality-api-hpa
  namespace: trenova
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: doc-quality-api
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

Deploy:

```bash
# Apply manifests
kubectl apply -f deployment.yaml

# Check status
kubectl get pods -n trenova -l app=doc-quality-api

# View logs
kubectl logs -n trenova -l app=doc-quality-api -f

# Test service
kubectl port-forward -n trenova service/doc-quality-api 8000:80
curl http://localhost:8000/health
```

---

## Monitoring & Maintenance

### Health Checks

```bash
# Liveness probe (is service running?)
curl http://localhost:8000/health

# Detailed metrics
curl http://localhost:8000/metrics
```

### Performance Monitoring

```bash
# Watch metrics in real-time
watch -n 5 'curl -s http://localhost:8000/metrics | jq .'

# Check processing times
curl -s http://localhost:8000/metrics | jq '.p95_processing_time_ms'

# Check acceptance rate
curl -s http://localhost:8000/metrics | jq '.acceptance_rate'
```

### Log Management

```bash
# Docker logs
docker-compose logs -f api | grep ERROR

# Systemd logs
sudo journalctl -u doc-quality-api -f --no-pager

# Kubernetes logs
kubectl logs -n trenova -l app=doc-quality-api --tail=100 -f
```

### Alerts

Set up alerts for:

1. **Service Health**: `/health` endpoint returns non-200
2. **High Latency**: P95 processing time > 500ms
3. **High Error Rate**: Error rate > 5%
4. **Low Acceptance Rate**: Acceptance rate < 60% (may indicate model degradation)
5. **Resource Usage**: Memory > 90%, CPU > 90%

Example Prometheus alerts:

```yaml
groups:
- name: doc_quality_api
  rules:
  - alert: HighErrorRate
    expr: rate(doc_quality_errors_total[5m]) > 0.05
    for: 5m
    annotations:
      summary: "High error rate in document quality API"

  - alert: HighLatency
    expr: doc_quality_p95_latency_ms > 500
    for: 10m
    annotations:
      summary: "High latency in document quality API"
```

### Model Updates

```bash
# 1. Train new model
python scripts/train_production.py --experiment production_v2

# 2. Evaluate new model
python scripts/evaluate_model.py \
    --model models/<new_model>.pth \
    --output-dir evaluation_v2

# 3. Compare with production model
# Check if new model is better

# 4. Backup current model
cp models/best_model.pth models/best_model_$(date +%Y%m%d).pth

# 5. Deploy new model
cp models/<new_model>.pth models/best_model.pth

# 6. Restart service
sudo systemctl restart doc-quality-api

# Or for Docker
docker-compose restart api

# 7. Monitor for issues
curl http://localhost:8000/metrics
```

### Gradual Rollout (Canary Deployment)

```bash
# 1. Deploy new version alongside old
docker run -d --name doc-quality-api-v2 \
    -p 8001:8000 \
    -v $(pwd)/models_v2:/app/models:ro \
    doc-quality-api:2.0.0

# 2. Route 10% of traffic to new version (nginx)
# upstream doc_quality_api {
#     server 127.0.0.1:8000 weight=9;
#     server 127.0.0.1:8001 weight=1;
# }

# 3. Monitor metrics for both versions
curl http://localhost:8000/metrics
curl http://localhost:8001/metrics

# 4. Gradually increase traffic to new version
# 5. If successful, fully switch to new version
# 6. Decommission old version
```

---

## Troubleshooting

### Service Won't Start

```bash
# Check if model exists
ls -lh models/best_model.pth

# Check if port is available
sudo lsof -i :8000

# Check logs
docker logs doc-quality-api
# or
sudo journalctl -u doc-quality-api -n 100

# Test model loading manually
python -c "
from src.api.inference import DocumentQualityPredictor
predictor = DocumentQualityPredictor('models/best_model.pth')
"
```

### CUDA Out of Memory

```bash
# Solution 1: Use CPU
./scripts/start_api.sh --cpu

# Solution 2: Reduce batch size in batch endpoint
# Edit src/api/app.py: max batch size

# Solution 3: Use smaller model
# Retrain with mobilenet_v3_small backbone
```

### High Latency

```bash
# Check current latency
curl -s http://localhost:8000/metrics | jq '.p95_processing_time_ms'

# Solutions:
# 1. Enable GPU
./scripts/start_api.sh --cuda

# 2. Increase workers
./scripts/start_api.sh --workers 4

# 3. Use batch processing for multiple documents

# 4. Add Redis caching layer (advanced)
```

### Model Predictions Look Wrong

```bash
# 1. Verify model is correct version
python -c "
import torch
checkpoint = torch.load('models/best_model.pth')
print(checkpoint.keys())
"

# 2. Check evaluation metrics
cat evaluation_results/best_model_evaluation_report.txt

# 3. Test with known good/bad documents
curl -X POST "http://localhost:8000/analyze" \
    -F "file=@test_high_quality.jpg"

# 4. Check for data drift
# Compare acceptance rate with historical data
```

### Memory Leak

```bash
# Monitor memory usage
docker stats doc-quality-api

# Reset metrics to free memory
curl -X POST http://localhost:8000/metrics/reset

# Restart service periodically (cron job)
0 2 * * * systemctl restart doc-quality-api
```

---

## Security Checklist

Before production deployment:

- [ ] Change default ports if exposed to internet
- [ ] Add authentication (API keys or OAuth2)
- [ ] Configure CORS properly (not `*` in production)
- [ ] Enable HTTPS with valid SSL certificate
- [ ] Set up rate limiting
- [ ] Configure file size limits
- [ ] Implement request logging
- [ ] Set up firewall rules
- [ ] Use secrets management for sensitive config
- [ ] Enable security headers in nginx

---

## Backup and Recovery

### Backup Model

```bash
# Automated backup script
#!/bin/bash
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR=/backups/doc-quality

# Backup model
cp models/best_model.pth $BACKUP_DIR/best_model_$DATE.pth

# Backup config
cp config/best_training.yaml $BACKUP_DIR/config_$DATE.yaml

# Keep only last 10 backups
ls -t $BACKUP_DIR/best_model_*.pth | tail -n +11 | xargs rm -f
```

### Restore from Backup

```bash
# List available backups
ls -lh /backups/doc-quality/

# Restore model
cp /backups/doc-quality/best_model_20250930.pth models/best_model.pth

# Restart service
sudo systemctl restart doc-quality-api
```

---

## Performance Tuning

### Optimize for Throughput

```bash
# Increase workers
./scripts/start_api.sh --workers 8

# Use batch processing
# Process 10 documents at once instead of individually

# Enable HTTP/2
# Configure nginx with http2
```

### Optimize for Latency

```bash
# Use GPU
./scripts/start_api.sh --cuda

# Reduce model size
# Use mobilenet_v3_small backbone

# Pre-load model (already done in startup)
```

### Optimize for Cost

```bash
# Use CPU for low-traffic services
./scripts/start_api.sh --cpu --workers 2

# Auto-scale based on traffic
# Use Kubernetes HPA or cloud provider auto-scaling
```

---

## Summary

You now have a production-ready document quality assessment service with:

✅ FastAPI REST API with comprehensive endpoints
✅ Docker containerization
✅ Kubernetes deployment manifests
✅ Systemd service configuration
✅ Nginx reverse proxy
✅ Health checks and monitoring
✅ Deployment automation
✅ Troubleshooting guides

**Next Steps:**

1. Choose deployment method (Docker/K8s/Direct)
2. Configure monitoring and alerts
3. Set up CI/CD pipeline
4. Implement authentication
5. Deploy to staging
6. Load test
7. Deploy to production
8. Monitor and iterate

For additional help, see:

- [API_GUIDE.md](API_GUIDE.md) - API usage and integration
- [TRAINING_GUIDE.md](TRAINING_GUIDE.md) - Model training
- [EVALUATION_GUIDE.md](EVALUATION_GUIDE.md) - Model evaluation

---

**Last Updated**: 2025-09-30
