<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->

# Document Type Classification Guide

Customer-aware document type classification for transportation management systems.

## Table of Contents

- [Overview](#overview)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [API Usage](#api-usage)
- [Customer Template Learning](#customer-template-learning)
- [Supported Document Types](#supported-document-types)
- [Best Practices](#best-practices)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)

## Overview

The document classification system automatically identifies document types (BOL, Invoice, Receipt, etc.) with **customer-specific template matching**. This is crucial for TMS systems because:

- **FedEx BOLs always look the same** - High accuracy with minimal training data
- **Customer-specific routing** - Route documents to correct workflows automatically
- **Quality + Classification** - Reject poor quality AND wrong document types
- **Self-improving** - Learns new customer templates automatically

### Key Features

✅ **10 Standard Document Types** - BOL, Invoice, Receipt, POD, Rate Confirmation, etc.
✅ **Customer Template Learning** - Learn unique customer layouts with 1-5 examples
✅ **High Accuracy Matching** - Deep learning features + cosine similarity
✅ **Few-Shot Learning** - Works with minimal examples per customer
✅ **Fast Inference** - < 100ms per document
✅ **Template Bank Persistence** - Save/load customer templates

## Architecture

### Three-Layer Classification

```
┌─────────────────────────────────────────────────┐
│  Input: Document Image                           │
└────────────────┬────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────┐
│  1. Feature Extraction (EfficientNet-B0)        │
│     - Extract 512-dim visual feature vector     │
│     - L2 normalized for cosine similarity       │
└────────────────┬────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────┐
│  2. Base Classification                          │
│     - Classify to 10 standard document types    │
│     - Returns: BOL, INVOICE, RECEIPT, etc.      │
│     - Confidence: 0-1                           │
└────────────────┬────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────┐
│  3. Customer Template Matching (if customer_id) │
│     - Compare features to customer templates    │
│     - Returns: customer_id + doc_type + sim     │
│     - High confidence (0.85-0.99) for matches   │
└────────────────┬────────────────────────────────┘
                 │
                 ▼
┌─────────────────────────────────────────────────┐
│  Output: Ranked Predictions                      │
│     - Base prediction: BOL (0.92)               │
│     - Customer match: fedex_BOL_std (0.97)      │
└─────────────────────────────────────────────────┘
```

### Customer Template Bank

Stores learned templates in memory for fast matching:

```python
{
  "fedex": {
    "BOL": [feature_vec_1, feature_vec_2, ...],
    "INVOICE": [feature_vec_1, ...]
  },
  "ups": {
    "BOL": [feature_vec_1, feature_vec_2, ...],
    "POD": [feature_vec_1, ...]
  }
}
```

## Quick Start

### 1. Start the API

```bash
cd services/ai-document-quality

# Start API
python -m src.api.app

# Or with Docker
docker-compose up api
```

### 2. Classify a Document (No Customer)

```bash
curl -X POST "http://localhost:8000/classify" \
  -F "file=@document.jpg" \
  -F "top_k=3" \
  -F "confidence_threshold=0.6"
```

Response:

```json
{
  "predictions": [
    {
      "document_type": "BOL",
      "base_type": "BOL",
      "customer_id": null,
      "confidence": 0.89,
      "source": "base_classifier",
      "description": "Bill of Lading - Primary shipping document"
    },
    {
      "document_type": "POD",
      "base_type": "POD",
      "confidence": 0.07,
      "source": "base_classifier"
    }
  ],
  "best_prediction": {...},
  "num_predictions": 2,
  "inference_time": 0.045,
  "has_customer_match": false
}
```

### 3. Learn Customer Template

```bash
curl -X POST "http://localhost:8000/templates/learn" \
  -F "file=@fedex_bol.jpg" \
  -F "customer_id=fedex" \
  -F "document_type=BOL" \
  -F "template_id=standard"
```

Response:

```json
{
  "success": true,
  "customer_id": "fedex",
  "document_type": "BOL",
  "template_id": "standard",
  "learning_time": 0.032,
  "customer_templates": {
    "BOL": 1
  },
  "total_templates": 1
}
```

### 4. Classify with Customer Match

```bash
curl -X POST "http://localhost:8000/classify" \
  -F "file=@another_fedex_bol.jpg" \
  -F "customer_id=fedex" \
  -F "top_k=3"
```

Response:

```json
{
  "predictions": [
    {
      "document_type": "fedex_BOL_standard",
      "base_type": "BOL",
      "customer_id": "fedex",
      "confidence": 0.97,
      "source": "customer_template",
      "description": "Customer-specific BOL",
      "template_metadata": {
        "template_id": "standard",
        "filename": "fedex_bol.jpg",
        "learned_at": "2025-09-30T12:00:00"
      }
    },
    {
      "document_type": "BOL",
      "base_type": "BOL",
      "confidence": 0.91,
      "source": "base_classifier"
    }
  ],
  "customer_id": "fedex",
  "has_customer_match": true,
  "inference_time": 0.038
}
```

## API Usage

### Classification Endpoints

#### POST /classify

Classify a single document.

**Parameters:**

- `file` (required): Document image file
- `customer_id` (optional): Customer ID for template matching
- `top_k` (default: 3): Number of predictions to return
- `confidence_threshold` (default: 0.6): Minimum confidence

**Example:**

```python
import requests

response = requests.post(
    "http://localhost:8000/classify",
    files={"file": open("document.jpg", "rb")},
    data={
        "customer_id": "fedex",
        "top_k": 3,
        "confidence_threshold": 0.7
    }
)

result = response.json()
best_match = result["best_prediction"]
print(f"Document Type: {best_match['document_type']}")
print(f"Confidence: {best_match['confidence']:.2f}")
```

#### POST /classify/batch

Classify multiple documents.

**Parameters:**

- `files` (required): List of document image files
- `customer_ids` (optional): Comma-separated customer IDs
- `top_k` (default: 3): Predictions per document
- `confidence_threshold` (default: 0.6): Minimum confidence

**Example:**

```python
files = [
    ("files", open("doc1.jpg", "rb")),
    ("files", open("doc2.jpg", "rb")),
    ("files", open("doc3.jpg", "rb"))
]

response = requests.post(
    "http://localhost:8000/classify/batch",
    files=files,
    data={
        "customer_ids": "fedex,ups,fedex",
        "top_k": 2
    }
)

results = response.json()["results"]
for i, result in enumerate(results):
    print(f"Document {i+1}: {result['best_prediction']['document_type']}")
```

### Template Management Endpoints

#### POST /templates/learn

Learn a new customer template.

**Parameters:**

- `file` (required): Document image file
- `customer_id` (required): Customer identifier
- `document_type` (required): Base document type (BOL, INVOICE, etc.)
- `template_id` (optional): Template identifier

**Example:**

```python
response = requests.post(
    "http://localhost:8000/templates/learn",
    files={"file": open("fedex_bol.jpg", "rb")},
    data={
        "customer_id": "fedex",
        "document_type": "BOL",
        "template_id": "standard_v1"
    }
)

print(f"Learned template: {response.json()}")
```

#### GET /templates/customer/{customer_id}

Get customer template information.

**Example:**

```python
response = requests.get(
    "http://localhost:8000/templates/customer/fedex"
)

info = response.json()
print(f"Customer: {info['customer_id']}")
print(f"Document Types: {info['document_types']}")
print(f"Template Counts: {info['template_counts']}")
```

#### GET /templates/customers

Get all customers with templates.

**Example:**

```python
response = requests.get(
    "http://localhost:8000/templates/customers"
)

customers = response.json()["customers"]
for customer in customers:
    print(f"{customer['customer_id']}: {customer['total_templates']} templates")
```

### Utility Endpoints

#### GET /classify/document-types

Get supported document types.

```python
response = requests.get(
    "http://localhost:8000/classify/document-types"
)

doc_types = response.json()["document_types"]
for code, description in doc_types.items():
    print(f"{code}: {description}")
```

#### GET /classify/metrics

Get classification service metrics.

```python
response = requests.get(
    "http://localhost:8000/classify/metrics"
)

metrics = response.json()
print(f"Total classifications: {metrics['total_classifications']}")
print(f"Average inference time: {metrics['average_inference_time']:.3f}s")
print(f"Customers with templates: {metrics['unique_customers']}")
```

## Customer Template Learning

### When to Learn Templates

Learn customer templates in these scenarios:

1. **First Document from Customer** - When you see the first document from a new customer
2. **New Document Type** - When customer sends a new type you haven't seen
3. **Template Updates** - When customer changes their document format
4. **Improved Accuracy** - Add more examples to improve matching

### How Many Examples Needed?

- **Minimum**: 1 example (often sufficient!)
- **Recommended**: 3-5 examples for robustness
- **Maximum**: No limit, but diminishing returns after ~10

Since customer documents always look the same (e.g., all FedEx BOLs are identical), **1-2 examples is usually enough** for 95%+ accuracy.

### Learning Workflow

```python
# 1. User uploads document via UI
# 2. Ask user to confirm document type and customer
# 3. Learn template
response = requests.post(
    "http://localhost:8000/templates/learn",
    files={"file": document_bytes},
    data={
        "customer_id": user_customer_id,
        "document_type": user_confirmed_type
    }
)

# 4. Future documents automatically classified
result = requests.post(
    "http://localhost:8000/classify",
    files={"file": new_document},
    data={"customer_id": user_customer_id}
)

# High confidence customer match!
# confidence: 0.95+
```

### Template Versioning

If customer changes document format, use `template_id`:

```python
# Original template
learn_template(customer_id="fedex", doc_type="BOL", template_id="v1")

# New template (after format change)
learn_template(customer_id="fedex", doc_type="BOL", template_id="v2")

# System now has both templates
# Will match whichever is more similar
```

## Supported Document Types

### Standard Document Types

| Code | Description | Common Use |
|------|-------------|------------|
| `BOL` | Bill of Lading | Primary shipping document |
| `INVOICE` | Freight Invoice | Payment document |
| `RECEIPT` | Delivery Receipt | Proof of goods received |
| `POD` | Proof of Delivery | Signed delivery confirmation |
| `RATE_CONF` | Rate Confirmation | Agreed shipping rates |
| `LUMPER` | Lumper Receipt | Unloading service receipt |
| `FUEL` | Fuel Receipt | Fuel purchase receipt |
| `SCALE` | Scale Ticket | Weight verification |
| `INSPECTION` | Inspection Report | Vehicle/cargo inspection |
| `OTHER` | Other Document | Unclassified |

### Adding Custom Types

To add custom document types, modify `STANDARD_DOCUMENT_TYPES` in `src/models/document_classifier.py`:

```python
STANDARD_DOCUMENT_TYPES = {
    # ... existing types ...
    "CUSTOMS": "Customs Declaration - International shipping",
    "PACKING_LIST": "Packing List - Detailed cargo contents",
}
```

Then retrain the base classifier with examples of the new types.

## Best Practices

### 1. Use Customer IDs Consistently

```python
# Good: Use same customer_id across all requests
customer_id = "fedex_freight"  # From your database

# Bad: Inconsistent IDs
customer_id = "FedEx"  # vs "fedex" vs "FEDEX"
```

### 2. Learn Templates Early

```python
# When driver uploads first FedEx BOL:
if not has_templates(customer_id="fedex", doc_type="BOL"):
    learn_template(...)  # Learn immediately
```

### 3. Confidence Thresholds

- **High confidence (0.9+)**: Very likely correct
- **Medium (0.7-0.9)**: Probably correct
- **Low (0.6-0.7)**: Uncertain, may need review
- **Very low (<0.6)**: Likely wrong or unusual

```python
best = result["best_prediction"]
if best["confidence"] >= 0.9:
    auto_process()  # High confidence, proceed
elif best["confidence"] >= 0.7:
    flag_for_review()  # Medium confidence
else:
    reject_or_manual_review()  # Low confidence
```

### 4. Combine with Quality Check

```python
# 1. Check quality first
quality_result = requests.post("/analyze", files={"file": doc})
if quality_result["quality_score"] < 0.5:
    return "Poor quality - reject"

# 2. Then classify
classification = requests.post("/classify", files={"file": doc})
return classification["best_prediction"]
```

### 5. Handle Multiple Customers

```python
# Map customer_id from your TMS database
tms_customer = get_customer_from_database(shipment_id)
customer_id = tms_customer.classification_id  # e.g., "fedex_freight"

result = classify_document(doc, customer_id=customer_id)
```

## Examples

### Complete Integration Example

```python
class DocumentProcessor:
    def __init__(self, api_base_url="http://localhost:8000"):
        self.api_url = api_base_url

    def process_uploaded_document(
        self,
        document_bytes: bytes,
        customer_id: str,
        shipment_id: str
    ):
        """Complete document processing workflow."""

        # 1. Check quality
        quality = requests.post(
            f"{self.api_url}/analyze",
            files={"file": document_bytes},
            data={"include_issues": True}
        ).json()

        if quality["quality_score"] < 0.5:
            return {
                "status": "rejected",
                "reason": "poor_quality",
                "quality_score": quality["quality_score"],
                "issues": quality["issues"]
            }

        # 2. Classify document
        classification = requests.post(
            f"{self.api_url}/classify",
            files={"file": document_bytes},
            data={
                "customer_id": customer_id,
                "top_k": 1,
                "confidence_threshold": 0.7
            }
        ).json()

        if not classification["predictions"]:
            return {
                "status": "rejected",
                "reason": "unrecognized_type"
            }

        best = classification["best_prediction"]

        # 3. Route based on type
        return {
            "status": "accepted",
            "document_type": best["document_type"],
            "base_type": best["base_type"],
            "confidence": best["confidence"],
            "customer_match": best["source"] == "customer_template",
            "route_to": self.get_workflow(best["base_type"])
        }

    def get_workflow(self, doc_type: str) -> str:
        """Map document type to workflow."""
        workflows = {
            "BOL": "shipping_workflow",
            "INVOICE": "billing_workflow",
            "POD": "delivery_confirmation",
            "RECEIPT": "delivery_confirmation"
        }
        return workflows.get(doc_type, "manual_review")

# Usage
processor = DocumentProcessor()
result = processor.process_uploaded_document(
    document_bytes=uploaded_file.read(),
    customer_id="fedex_freight",
    shipment_id="SHIP-12345"
)

print(f"Status: {result['status']}")
if result['status'] == 'accepted':
    print(f"Type: {result['document_type']} ({result['confidence']:.2f})")
    print(f"Route to: {result['route_to']}")
```

### React Integration

```typescript
async function classifyDocument(
  file: File,
  customerId: string
): Promise<ClassificationResult> {
  const formData = new FormData();
  formData.append('file', file);
  formData.append('customer_id', customerId);
  formData.append('top_k', '3');

  const response = await fetch('http://localhost:8000/classify', {
    method: 'POST',
    body: formData
  });

  return await response.json();
}

// Usage in component
const handleDocumentUpload = async (file: File) => {
  const result = await classifyDocument(file, shipment.customerId);

  if (result.best_prediction.confidence >= 0.9) {
    // High confidence - auto-process
    autoProcessDocument(file, result.best_prediction.document_type);
  } else {
    // Lower confidence - ask user to confirm
    setShowConfirmDialog(true);
    setClassification(result);
  }
};
```

## Troubleshooting

### Issue: Low Confidence Scores

**Symptoms:** All predictions have confidence < 0.7

**Causes:**

1. No customer templates learned yet
2. Document is unusual/damaged
3. Wrong customer_id provided

**Solutions:**

```python
# Check if customer has templates
info = requests.get(f"/templates/customer/{customer_id}").json()
if info["total_templates"] == 0:
    # No templates - learn one!
    requests.post("/templates/learn", ...)

# Try without customer_id to get base classification
result = requests.post("/classify", files={"file": doc})
```

### Issue: Wrong Document Type Predicted

**Symptoms:** Consistently classifies FedEx BOL as INVOICE

**Causes:**

1. Document is ambiguous (looks like both)
2. Base classifier needs retraining
3. Customer template not learned

**Solutions:**

```python
# Learn correct template
requests.post(
    "/templates/learn",
    files={"file": fedex_bol},
    data={
        "customer_id": "fedex",
        "document_type": "BOL"  # Correct type
    }
)

# Future predictions will be accurate
```

### Issue: Customer Match Not Working

**Symptoms:** `has_customer_match: false` even after learning template

**Causes:**

1. Different customer_id used for learning vs classification
2. Document significantly different from template
3. Confidence threshold too high

**Solutions:**

```python
# Verify customer_id matches
learn: customer_id="fedex"
classify: customer_id="fedex"  # Must match exactly!

# Lower threshold
result = requests.post(
    "/classify",
    data={"confidence_threshold": 0.5}  # Lower threshold
)

# Check customer templates
info = requests.get("/templates/customer/fedex").json()
print(f"Has templates: {info['has_templates']}")
```

### Issue: Slow Classification

**Symptoms:** Inference time > 500ms

**Causes:**

1. Running on CPU (normal: 50-150ms)
2. Large batch size
3. Model not optimized

**Solutions:**

```bash
# Use GPU if available
export DEVICE=cuda

# Check metrics
curl http://localhost:8000/classify/metrics

# Optimize: Use batch endpoint for multiple docs
# Instead of 10 separate calls, use 1 batch call
```

## Performance

### Benchmarks

| Metric | Value |
|--------|-------|
| Inference time (CPU) | 50-150ms |
| Inference time (GPU) | 10-30ms |
| Batch throughput | ~100 docs/sec |
| Template learning | 30-50ms |
| Memory usage | ~500MB |

### Scaling Tips

1. **Use batch endpoint** for multiple documents
2. **Enable GPU** for production workloads
3. **Cache customer templates** (automatically handled)
4. **Load balance** across multiple instances

## Next Steps

1. **Integrate with TMS**: Connect to your shipment workflow
2. **Learn customer templates**: Add templates for top customers
3. **Monitor accuracy**: Track confidence scores and errors
4. **Combine with quality**: Use both services together
5. **Automate**: Build automated document routing

For more information, see:

- [API_GUIDE.md](API_GUIDE.md) - Complete API documentation
- [DEPLOYMENT_GUIDE.md](DEPLOYMENT_GUIDE.md) - Production deployment
- [TESTING_GUIDE.md](TESTING_GUIDE.md) - Testing documentation
