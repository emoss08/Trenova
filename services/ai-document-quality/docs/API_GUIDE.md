# Document Quality Assessment API - Complete Guide

Production-ready REST API for assessing document quality in real-time.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Deployment](#deployment)
3. [API Endpoints](#api-endpoints)
4. [Usage Examples](#usage-examples)
5. [Error Handling](#error-handling)
6. [Performance & Monitoring](#performance--monitoring)
7. [Integration Guide](#integration-guide)

---

## Quick Start

### Local Development

```bash
# 1. Ensure model is trained
ls models/best_model.pth

# 2. Start API server
./scripts/start_api.sh

# 3. Test API (in another terminal)
curl http://localhost:8000/health
```

### Docker Deployment

```bash
# Build and start
docker-compose up -d api

# Check logs
docker-compose logs -f api

# Stop
docker-compose down
```

### Access Documentation

Once the API is running:
- **Swagger UI**: http://localhost:8000/docs
- **ReDoc**: http://localhost:8000/redoc
- **Health Check**: http://localhost:8000/health

---

## Deployment

### Option 1: Direct Python Execution

```bash
# Production mode
./scripts/start_api.sh --port 8000 --workers 4

# Development mode (auto-reload)
./scripts/start_api.sh --dev

# Custom model and config
./scripts/start_api.sh \
    --model models/my_model.pth \
    --config config/my_config.yaml \
    --port 8080
```

**Script Options:**
- `--model PATH` - Model checkpoint path
- `--config PATH` - Configuration file path
- `--port PORT` - Server port (default: 8000)
- `--host HOST` - Bind host (default: 0.0.0.0)
- `--workers N` - Number of worker processes (default: 1)
- `--dev` - Development mode with auto-reload
- `--cpu` - Force CPU usage
- `--cuda` - Force CUDA/GPU usage

### Option 2: Docker

```bash
# Production deployment
docker-compose up -d api

# Development deployment (with hot-reload)
docker-compose --profile dev up api-dev

# With MLflow tracking
docker-compose --profile mlflow up mlflow
```

**Environment Variables:**
```bash
# Create .env file
cat > .env << EOF
PORT=8000
DEVICE=cuda
WORKERS=4
CORS_ORIGINS=https://app.trenova.com,https://api.trenova.com
CPU_LIMIT=4.0
MEMORY_LIMIT=8G
EOF

# Start with env file
docker-compose --env-file .env up -d
```

### Option 3: Kubernetes (Production)

```yaml
# deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: doc-quality-api
spec:
  replicas: 3
  selector:
    matchLabels:
      app: doc-quality-api
  template:
    metadata:
      labels:
        app: doc-quality-api
    spec:
      containers:
      - name: api
        image: doc-quality-api:2.0.0
        ports:
        - containerPort: 8000
        env:
        - name: DEVICE
          value: "cuda"
        - name: WORKERS
          value: "2"
        resources:
          requests:
            memory: "2Gi"
            cpu: "1"
          limits:
            memory: "4Gi"
            cpu: "2"
        livenessProbe:
          httpGet:
            path: /health
            port: 8000
          initialDelaySeconds: 30
          periodSeconds: 10
```

---

## API Endpoints

### 1. Health Check

```http
GET /health
```

**Response:**
```json
{
  "status": "healthy",
  "version": "2.0.0",
  "model_loaded": true,
  "device": "cuda",
  "uptime_seconds": 3600.5,
  "requests_processed": 1234
}
```

### 2. Performance Metrics

```http
GET /metrics
```

**Response:**
```json
{
  "total_requests": 1000,
  "average_processing_time_ms": 150.5,
  "p50_processing_time_ms": 145.0,
  "p95_processing_time_ms": 250.0,
  "p99_processing_time_ms": 400.0,
  "acceptance_rate": 0.85,
  "rejection_rate": 0.15,
  "average_quality_score": 0.73,
  "errors": 5
}
```

### 3. Analyze Single Document

```http
POST /analyze
Content-Type: multipart/form-data

Parameters:
- file: image file (required)
- threshold: float (0-1, default: 0.5)
- include_issues: boolean (default: true)
```

**Response:**
```json
{
  "request_id": "req_abc123def456",
  "timestamp": "2025-09-30T14:30:22Z",
  "quality": {
    "score": 0.85,
    "quality_class": "Good",
    "quality_class_index": 1,
    "is_acceptable": true,
    "confidence": 0.92
  },
  "issues": [
    {
      "issue_type": "slight_blur",
      "probability": 0.35,
      "severity": "minor"
    }
  ],
  "recommendations": [
    "✓ Document quality is acceptable (score: 0.850)",
    "Document is suitable for processing"
  ],
  "processing_time_ms": 145.5,
  "visualization_url": null
}
```

### 4. Batch Analysis

```http
POST /analyze/batch
Content-Type: multipart/form-data

Parameters:
- files: array of image files (required, max 100)
- threshold: float (0-1, default: 0.5)
- include_issues: boolean (default: true)
```

**Response:**
```json
{
  "request_id": "batch_xyz789abc123",
  "timestamp": "2025-09-30T14:30:22Z",
  "total_documents": 10,
  "results": [
    { /* DocumentAnalysisResponse */ },
    { /* DocumentAnalysisResponse */ },
    // ... more results
  ],
  "summary": {
    "acceptable": 8,
    "rejected": 2,
    "average_quality_score": 0.75,
    "quality_distribution": {
      "High": 2,
      "Good": 6,
      "Moderate": 1,
      "Poor": 1,
      "Very Poor": 0
    }
  },
  "total_processing_time_ms": 1250.0
}
```

---

## Usage Examples

### cURL

#### Single Document Analysis

```bash
# Basic analysis
curl -X POST "http://localhost:8000/analyze" \
  -F "file=@document.jpg"

# Custom threshold
curl -X POST "http://localhost:8000/analyze?threshold=0.6" \
  -F "file=@document.jpg"

# Without issue detection (faster)
curl -X POST "http://localhost:8000/analyze?include_issues=false" \
  -F "file=@document.jpg"
```

#### Batch Analysis

```bash
curl -X POST "http://localhost:8000/analyze/batch" \
  -F "files=@doc1.jpg" \
  -F "files=@doc2.jpg" \
  -F "files=@doc3.jpg"
```

### Python Client

```python
import requests

# Single document
def analyze_document(file_path: str, threshold: float = 0.5):
    with open(file_path, "rb") as f:
        files = {"file": f}
        params = {"threshold": threshold}
        response = requests.post(
            "http://localhost:8000/analyze",
            files=files,
            params=params
        )
        return response.json()

# Usage
result = analyze_document("invoice.jpg", threshold=0.6)
print(f"Quality Score: {result['quality']['score']}")
print(f"Acceptable: {result['quality']['is_acceptable']}")

# Batch processing
def analyze_batch(file_paths: list[str], threshold: float = 0.5):
    files = [("files", open(path, "rb")) for path in file_paths]
    params = {"threshold": threshold}
    response = requests.post(
        "http://localhost:8000/analyze/batch",
        files=files,
        params=params
    )
    for f in files:
        f[1].close()
    return response.json()

# Usage
results = analyze_batch(["doc1.jpg", "doc2.jpg", "doc3.jpg"])
print(f"Processed {results['total_documents']} documents")
print(f"Acceptable: {results['summary']['acceptable']}")
print(f"Rejected: {results['summary']['rejected']}")
```

### JavaScript/TypeScript

```typescript
// Single document
async function analyzeDocument(file: File, threshold: number = 0.5) {
  const formData = new FormData();
  formData.append('file', file);

  const response = await fetch(
    `http://localhost:8000/analyze?threshold=${threshold}`,
    {
      method: 'POST',
      body: formData,
    }
  );

  return await response.json();
}

// Usage in React
const handleFileUpload = async (event: React.ChangeEvent<HTMLInputElement>) => {
  const file = event.target.files?.[0];
  if (!file) return;

  try {
    const result = await analyzeDocument(file, 0.5);

    if (result.quality.is_acceptable) {
      console.log('✓ Document accepted');
    } else {
      console.log('✗ Document rejected');
      console.log('Recommendations:', result.recommendations);
    }
  } catch (error) {
    console.error('Analysis failed:', error);
  }
};

// Batch processing
async function analyzeBatch(files: File[], threshold: number = 0.5) {
  const formData = new FormData();
  files.forEach(file => formData.append('files', file));

  const response = await fetch(
    `http://localhost:8000/analyze/batch?threshold=${threshold}`,
    {
      method: 'POST',
      body: formData,
    }
  );

  return await response.json();
}
```

### Go Client

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
    "os"
)

type QualityResponse struct {
    RequestID string `json:"request_id"`
    Quality   struct {
        Score        float64 `json:"score"`
        QualityClass string  `json:"quality_class"`
        IsAcceptable bool    `json:"is_acceptable"`
    } `json:"quality"`
    Recommendations []string `json:"recommendations"`
}

func analyzeDocument(filePath string, threshold float64) (*QualityResponse, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return nil, err
    }
    defer file.Close()

    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)

    part, err := writer.CreateFormFile("file", filePath)
    if err != nil {
        return nil, err
    }

    _, err = io.Copy(part, file)
    if err != nil {
        return nil, err
    }

    writer.Close()

    url := fmt.Sprintf("http://localhost:8000/analyze?threshold=%.2f", threshold)
    req, err := http.NewRequest("POST", url, body)
    if err != nil {
        return nil, err
    }

    req.Header.Set("Content-Type", writer.FormDataContentType())

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    var result QualityResponse
    err = json.NewDecoder(resp.Body).Decode(&result)
    return &result, err
}

func main() {
    result, err := analyzeDocument("invoice.jpg", 0.5)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Quality Score: %.3f\n", result.Quality.Score)
    fmt.Printf("Acceptable: %v\n", result.Quality.IsAcceptable)
}
```

---

## Error Handling

### Error Response Format

```json
{
  "error": "InvalidImage",
  "message": "Unable to process image file",
  "request_id": "req_abc123",
  "timestamp": "2025-09-30T14:30:22Z"
}
```

### Common Error Codes

| Status Code | Error Type | Description | Solution |
|-------------|------------|-------------|----------|
| 400 | InvalidFile | File type not supported | Use JPG, PNG, or TIFF |
| 400 | FileTooLarge | File exceeds size limit | Compress or resize image |
| 400 | InvalidThreshold | Threshold outside 0-1 range | Use value between 0 and 1 |
| 503 | ServiceUnavailable | Model not loaded | Wait for startup or check logs |
| 500 | InternalServerError | Processing error | Check logs, retry request |

### Retry Logic

```python
import time
from requests.adapters import HTTPAdapter
from requests.packages.urllib3.util.retry import Retry

def get_session_with_retries():
    session = requests.Session()
    retries = Retry(
        total=3,
        backoff_factor=1,
        status_forcelist=[500, 502, 503, 504],
    )
    adapter = HTTPAdapter(max_retries=retries)
    session.mount('http://', adapter)
    session.mount('https://', adapter)
    return session

# Usage
session = get_session_with_retries()
response = session.post("http://localhost:8000/analyze", files=files)
```

---

## Performance & Monitoring

### Benchmarks

**Single Document:**
- Average latency: 150ms (GPU), 500ms (CPU)
- P95 latency: 250ms (GPU), 800ms (CPU)
- Throughput: ~7 docs/sec (GPU), ~2 docs/sec (CPU)

**Batch Processing (10 documents):**
- Average latency: 800ms (GPU), 2500ms (CPU)
- Throughput: ~12 docs/sec (GPU), ~4 docs/sec (CPU)

### Monitoring Endpoints

```bash
# Health check (use for liveness probe)
curl http://localhost:8000/health

# Metrics (use for monitoring dashboard)
curl http://localhost:8000/metrics

# Reset metrics (admin only)
curl -X POST http://localhost:8000/metrics/reset
```

### Prometheus Integration

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'doc-quality-api'
    static_configs:
      - targets: ['localhost:8000']
    metrics_path: '/metrics'
    scrape_interval: 30s
```

### Load Testing

```bash
# Install Apache Bench
sudo apt-get install apache2-utils

# Test with 100 requests, 10 concurrent
ab -n 100 -c 10 -p test_image.jpg -T 'image/jpeg' \
   http://localhost:8000/analyze

# Or use wrk
wrk -t4 -c100 -d30s --latency http://localhost:8000/health
```

---

## Integration Guide

### React Integration Example

```typescript
// useDocumentQuality.ts
import { useState } from 'react';

interface QualityResult {
  score: number;
  isAcceptable: boolean;
  recommendations: string[];
}

export function useDocumentQuality(apiUrl: string = 'http://localhost:8000') {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const analyzeDocument = async (
    file: File,
    threshold: number = 0.5
  ): Promise<QualityResult | null> => {
    setLoading(true);
    setError(null);

    try {
      const formData = new FormData();
      formData.append('file', file);

      const response = await fetch(
        `${apiUrl}/analyze?threshold=${threshold}`,
        {
          method: 'POST',
          body: formData,
        }
      );

      if (!response.ok) {
        throw new Error(`API error: ${response.statusText}`);
      }

      const data = await response.json();

      return {
        score: data.quality.score,
        isAcceptable: data.quality.is_acceptable,
        recommendations: data.recommendations,
      };
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
      return null;
    } finally {
      setLoading(false);
    }
  };

  return { analyzeDocument, loading, error };
}

// Component usage
function DocumentUpload() {
  const { analyzeDocument, loading, error } = useDocumentQuality();

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;

    const result = await analyzeDocument(file, 0.5);

    if (result) {
      if (result.isAcceptable) {
        alert('✓ Document accepted for processing');
      } else {
        alert(`✗ Document rejected\n${result.recommendations.join('\n')}`);
      }
    }
  };

  return (
    <div>
      <input type="file" onChange={handleFileChange} disabled={loading} />
      {loading && <p>Analyzing...</p>}
      {error && <p style={{ color: 'red' }}>Error: {error}</p>}
    </div>
  );
}
```

### Mobile App Integration (React Native)

```typescript
// DocumentScanner.tsx
import { launchCamera, launchImageLibrary } from 'react-native-image-picker';

async function captureAndAnalyze() {
  const result = await launchCamera({ mediaType: 'photo' });

  if (result.assets && result.assets[0].uri) {
    const formData = new FormData();
    formData.append('file', {
      uri: result.assets[0].uri,
      type: 'image/jpeg',
      name: 'document.jpg',
    } as any);

    const response = await fetch('http://api.example.com:8000/analyze', {
      method: 'POST',
      body: formData,
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    });

    const data = await response.json();

    if (data.quality.is_acceptable) {
      // Proceed with upload
      uploadDocument(result.assets[0].uri);
    } else {
      // Show quality feedback
      Alert.alert(
        'Document Quality Issue',
        data.recommendations.join('\n'),
        [
          { text: 'Retry', onPress: () => captureAndAnalyze() },
          { text: 'Cancel', style: 'cancel' },
        ]
      );
    }
  }
}
```

### Backend Integration (Go/TMS)

```go
// services/document/quality_checker.go
package document

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "mime/multipart"
    "net/http"
)

type QualityChecker struct {
    apiURL    string
    threshold float64
    client    *http.Client
}

func NewQualityChecker(apiURL string, threshold float64) *QualityChecker {
    return &QualityChecker{
        apiURL:    apiURL,
        threshold: threshold,
        client:    &http.Client{Timeout: 30 * time.Second},
    }
}

func (qc *QualityChecker) CheckDocument(documentPath string) (bool, string, error) {
    // Read document
    file, err := os.Open(documentPath)
    if err != nil {
        return false, "", err
    }
    defer file.Close()

    // Create multipart request
    body := &bytes.Buffer{}
    writer := multipart.NewWriter(body)
    part, _ := writer.CreateFormFile("file", documentPath)
    io.Copy(part, file)
    writer.Close()

    // Make API request
    url := fmt.Sprintf("%s/analyze?threshold=%.2f", qc.apiURL, qc.threshold)
    req, _ := http.NewRequest("POST", url, body)
    req.Header.Set("Content-Type", writer.FormDataContentType())

    resp, err := qc.client.Do(req)
    if err != nil {
        return false, "", err
    }
    defer resp.Body.Close()

    // Parse response
    var result struct {
        Quality struct {
            IsAcceptable bool    `json:"is_acceptable"`
            Score        float64 `json:"score"`
        } `json:"quality"`
        Recommendations []string `json:"recommendations"`
    }

    json.NewDecoder(resp.Body).Decode(&result)

    message := fmt.Sprintf("Quality score: %.2f", result.Quality.Score)
    if !result.Quality.IsAcceptable {
        message += "\n" + strings.Join(result.Recommendations, "\n")
    }

    return result.Quality.IsAcceptable, message, nil
}

// Usage in shipment handler
func (h *ShipmentHandler) UploadDocument(c *gin.Context) {
    // ... handle file upload ...

    // Check quality before accepting
    qc := NewQualityChecker("http://doc-quality-api:8000", 0.5)
    acceptable, message, err := qc.CheckDocument(tempFilePath)

    if err != nil {
        c.JSON(500, gin.H{"error": "Quality check failed"})
        return
    }

    if !acceptable {
        c.JSON(400, gin.H{"error": message})
        return
    }

    // Quality check passed, proceed with storage
    // ...
}
```

---

## Best Practices

### 1. Threshold Selection

```python
# Production thresholds based on use case
THRESHOLDS = {
    "strict": 0.7,      # Critical documents (invoices for payment)
    "standard": 0.5,    # Normal documents (BOL, receipts)
    "lenient": 0.3,     # Internal documents (notes, memos)
}

# Use appropriate threshold
result = analyze_document("invoice.jpg", threshold=THRESHOLDS["strict"])
```

### 2. Batch Processing for Efficiency

```python
# ✅ Good: Batch processing
files = ["doc1.jpg", "doc2.jpg", "doc3.jpg", ...]
results = analyze_batch(files)  # ~3x faster than individual

# ❌ Bad: Individual requests in loop
for file in files:
    result = analyze_document(file)  # Slower
```

### 3. Error Handling and Retries

```python
def analyze_with_retry(file_path: str, max_retries: int = 3):
    for attempt in range(max_retries):
        try:
            return analyze_document(file_path)
        except requests.RequestException as e:
            if attempt == max_retries - 1:
                raise
            time.sleep(2 ** attempt)  # Exponential backoff
```

### 4. Caching Results

```python
from functools import lru_cache
import hashlib

def file_hash(file_path: str) -> str:
    with open(file_path, "rb") as f:
        return hashlib.md5(f.read()).hexdigest()

@lru_cache(maxsize=1000)
def analyze_cached(file_hash: str, threshold: float):
    # Cache results by file hash
    return analyze_document(file_path, threshold)
```

---

## Security Considerations

1. **File Size Limits**: Configure max upload size (default: 16MB)
2. **Rate Limiting**: Implement rate limiting in production
3. **Authentication**: Add API key or OAuth2 for production
4. **CORS**: Configure allowed origins properly
5. **Input Validation**: API validates file types and parameters

---

## Support

For issues or questions:
- API Documentation: http://localhost:8000/docs
- Main README: [README.md](README.md)
- Training Guide: [TRAINING_GUIDE.md](TRAINING_GUIDE.md)
- GitHub Issues: Open an issue with API logs

---

**API Version**: 2.0.0
**Last Updated**: 2025-09-30
