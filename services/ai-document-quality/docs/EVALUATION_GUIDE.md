# Document Quality Assessment - Evaluation Guide

This guide explains how to comprehensively evaluate the document quality assessment model.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Evaluation Metrics](#evaluation-metrics)
3. [Command-Line Evaluation](#command-line-evaluation)
4. [Interactive Notebook](#interactive-notebook)
5. [Understanding Results](#understanding-results)
6. [Troubleshooting](#troubleshooting)

---

## Quick Start

### Prerequisites

```bash
# Ensure you have a trained model
ls models/best_model.pth

# Ensure you have test data
ls datasets/default/test/test_metadata.csv
```

### Run Evaluation (CLI)

```bash
# Basic evaluation
python scripts/evaluate_model.py --model models/best_model.pth

# With explainability visualizations
python scripts/evaluate_model.py --model models/best_model.pth --explain --num-samples 20

# Custom output directory
python scripts/evaluate_model.py \
    --model models/best_model.pth \
    --output-dir my_evaluation_results \
    --explain
```

### Run Evaluation (Notebook)

```bash
# Start Jupyter
jupyter notebook notebooks/evaluate_model.ipynb
```

---

## Evaluation Metrics

The evaluation suite calculates comprehensive metrics across multiple dimensions:

### 1. Regression Metrics (Quality Scores)

Measures how well the model predicts continuous quality scores (0-1 range):

- **MAE (Mean Absolute Error)**: Average absolute difference between predictions and ground truth
  - *Good*: < 0.10
  - *Acceptable*: 0.10 - 0.15
  - *Poor*: > 0.15

- **RMSE (Root Mean Squared Error)**: Square root of average squared errors (penalizes large errors more)
  - *Good*: < 0.12
  - *Acceptable*: 0.12 - 0.18
  - *Poor*: > 0.18

- **R² (R-squared)**: Proportion of variance explained by the model
  - *Excellent*: > 0.90
  - *Good*: 0.80 - 0.90
  - *Acceptable*: 0.70 - 0.80
  - *Poor*: < 0.70

- **Within 5%/10%/20%**: Percentage of predictions within tolerance of ground truth

### 2. Classification Metrics (Quality Classes)

Measures classification of documents into 5 quality classes:

- **Accuracy**: Overall percentage of correct classifications
- **Balanced Accuracy**: Accuracy accounting for class imbalance
- **Weighted F1**: Harmonic mean of precision and recall, weighted by class frequency
- **Per-Class Metrics**: Precision, recall, F1 for each quality class

**Target Values:**

- Balanced Accuracy: > 0.85
- Weighted F1: > 0.80
- Per-Class F1: > 0.70 for all classes

### 3. Binary Classification (Accept/Reject)

Measures accept/reject decision based on quality threshold:

- **Precision**: Of documents accepted, what % are truly acceptable?
- **Recall**: Of truly acceptable documents, what % did we accept?
- **F1 Score**: Harmonic mean of precision and recall
- **FPR (False Positive Rate)**: % of bad documents incorrectly accepted ⚠️
- **FNR (False Negative Rate)**: % of good documents incorrectly rejected ⚠️
- **ROC AUC**: Area under receiver operating characteristic curve
- **PR AUC**: Area under precision-recall curve

**Target Values:**

- F1 Score: > 0.85
- FPR (False Accept Rate): < 0.05 (critical for quality control)
- FNR (False Reject Rate): < 0.10 (affects user experience)
- ROC AUC: > 0.90

### 4. Calibration Metrics

Measures whether model confidence matches actual accuracy:

- **ECE (Expected Calibration Error)**: Average difference between confidence and accuracy
  - *Well Calibrated*: < 0.05
  - *Acceptable*: 0.05 - 0.10
  - *Poorly Calibrated*: > 0.10

- **MCE (Maximum Calibration Error)**: Worst-case calibration error

**Why Calibration Matters:**

- If ECE is high, confidence scores are unreliable
- Users can't trust probability estimates
- Threshold selection becomes difficult

### 5. Issue Detection Metrics

Measures multi-label classification of document issues:

- **Per-Issue F1**: Performance detecting each specific issue type
- **Macro F1**: Average F1 across all issue types (treats issues equally)
- **Micro F1**: Overall F1 treating all samples equally
- **Hamming Loss**: Fraction of incorrect issue labels

**Issue Types:**

1. Blur
2. Noise
3. Lighting
4. Shadow
5. Physical Damage
6. Skew
7. Partial Capture
8. Glare
9. Compression
10. Overall Poor Quality

---

## Command-Line Evaluation

### Basic Usage

```bash
python scripts/evaluate_model.py --model <MODEL_PATH> [OPTIONS]
```

### Options

| Option | Description | Default |
|--------|-------------|---------|
| `--model` | Path to model checkpoint (required) | - |
| `--config` | Path to config YAML | `config/best_training.yaml` |
| `--output-dir` | Output directory for results | `evaluation_results` |
| `--threshold` | Quality score threshold for accept/reject | `0.5` |
| `--explain` | Generate Grad-CAM explanations | `False` |
| `--num-samples` | Number of samples to explain | `20` |
| `--device` | Device to use (auto/cuda/cpu) | `auto` |

### Examples

#### 1. Quick Evaluation

```bash
python scripts/evaluate_model.py --model models/best_model.pth
```

**Output:**

- `evaluation_results/evaluation_results.json` - All metrics in JSON
- `evaluation_results/best_model_evaluation_report.txt` - Human-readable report
- `evaluation_results/*.png` - Visualization plots

#### 2. Full Evaluation with Explanations

```bash
python scripts/evaluate_model.py \
    --model models/best_model.pth \
    --explain \
    --num-samples 50 \
    --output-dir full_evaluation
```

**Additional Output:**

- `full_evaluation/explanations/*.png` - Grad-CAM visualizations for 50 samples

#### 3. Custom Threshold Evaluation

```bash
python scripts/evaluate_model.py \
    --model models/best_model.pth \
    --threshold 0.6 \
    --output-dir strict_threshold_eval
```

Useful for testing stricter quality requirements.

### Output Files

After running evaluation, you'll find:

```
evaluation_results/
├── evaluation_results.json          # Complete metrics in JSON format
├── best_model_evaluation_report.txt # Human-readable text report
├── best_model_calibration.png       # Calibration curve
├── best_model_confusion_matrix.png  # Confusion matrix heatmap
├── best_model_roc_curve.png         # ROC curve
├── best_model_pr_curve.png          # Precision-Recall curve
├── best_model_predictions.png       # Prediction distribution analysis
├── best_model_issue_analysis.png    # Per-issue performance
├── best_model_threshold_analysis.png # Threshold vs metrics
└── explanations/                     # (if --explain used)
    ├── explanation_image1.png
    ├── explanation_image2.png
    └── ...
```

---

## Interactive Notebook

The Jupyter notebook provides an interactive environment for exploration.

### Starting the Notebook

```bash
# Activate virtual environment
source .venv/bin/activate

# Start Jupyter
jupyter notebook notebooks/evaluate_model.ipynb
```

### Notebook Sections

1. **Setup** - Import libraries and configure
2. **Load Model** - Load trained model and configuration
3. **Load Test Data** - Load test dataset
4. **Run Evaluation** - Calculate all metrics
5. **Visualizations** - Interactive plots
6. **Test Individual Images** - Explain specific predictions
7. **Analyze Worst Predictions** - Debug problem cases
8. **Per-Class Analysis** - Performance by quality class
9. **Summary** - Overall assessment and recommendations
10. **Export Results** - Save for reporting

### Key Features

- **Interactive Plotting**: Zoom, pan, and explore visualizations
- **Single Image Testing**: Upload and test your own images
- **Error Analysis**: Identify and visualize worst predictions
- **Per-Class Analysis**: Understand performance for each quality level
- **Export**: Save results for reports or presentations

---

## Understanding Results

### Reading the Evaluation Report

The text report (`*_evaluation_report.txt`) contains:

```
================================================================================
EVALUATION REPORT: best_model
================================================================================

Number of samples: 110

REGRESSION METRICS (Quality Scores)
--------------------------------------------------------------------------------
  MAE:  0.0842
  RMSE: 0.1123
  R²:   0.8456
  Predictions within 5%:  45.45%
  Predictions within 10%: 72.73%
  Predictions within 20%: 90.91%

CLASSIFICATION METRICS (Quality Classes)
--------------------------------------------------------------------------------
  Accuracy:          0.8273
  Balanced Accuracy: 0.7891
  Weighted F1:       0.8156
  ...
```

### Interpreting Visualizations

#### 1. Calibration Curve

![Calibration Example](docs/calibration_example.png)

- **Perfect calibration**: Points fall on diagonal line
- **Overconfident**: Points below diagonal (predicted confidence > actual accuracy)
- **Underconfident**: Points above diagonal (predicted confidence < actual accuracy)
- **ECE < 0.10**: Model is well-calibrated

#### 2. Confusion Matrix

![Confusion Matrix Example](docs/confusion_matrix_example.png)

- **Diagonal**: Correct predictions
- **Off-diagonal**: Misclassifications
- **Look for**: Are misclassifications adjacent classes? (Good → Moderate is better than Good → Very Poor)

#### 3. ROC Curve

![ROC Curve Example](docs/roc_example.png)

- **Top-left corner**: Perfect classifier
- **Diagonal line**: Random guessing
- **AUC > 0.90**: Excellent discriminative ability
- **Optimal point**: Marked with red star (maximizes TPR - FPR)

#### 4. Prediction Distribution

![Prediction Distribution Example](docs/distribution_example.png)

- **Top-left**: Scatter of predictions vs ground truth
- **Top-right**: Histogram comparing distributions
- **Bottom-left**: Residuals (should be randomly scattered around 0)
- **Bottom-right**: Box plots by quality category

### Grad-CAM Explanations

Grad-CAM visualizations show what the model "sees" when making predictions:

![Grad-CAM Example](docs/gradcam_example.png)

**Components:**

1. **Original Document**: Input image
2. **Quality Assessment Focus**: Heatmap overlay (red = high importance)
3. **Attention Heatmap**: Heatmap alone
4. **Quality Score**: Predicted score and class
5. **Class Probabilities**: Distribution across quality classes
6. **Detected Issues**: Issues with probability > 0.3
7. **Recommendations**: Actionable feedback

**Interpreting Heatmaps:**

- **Red/Yellow regions**: Model focuses here for quality assessment
- **Blue regions**: Less important for decision
- **Expected patterns**:
  - Text regions should be highlighted
  - Edges and signatures important
  - Damaged areas should be detected

---

## Troubleshooting

### Common Issues

#### 1. "Model file not found"

```bash
# Check if model exists
ls -lh models/

# If missing, train a model first
python scripts/train.py --config config/best_training.yaml
```

#### 2. "Test metadata not found"

```bash
# Check dataset structure
ls -R datasets/default/

# If missing, generate dataset
python -m src.data.dataset \
    --input-dir documents \
    --output-dir datasets \
    --mode synthetic
```

#### 3. "CUDA out of memory"

```bash
# Use CPU instead
python scripts/evaluate_model.py \
    --model models/best_model.pth \
    --device cpu

# Or reduce batch size in config
```

#### 4. Poor Calibration (ECE > 0.10)

**Solutions:**

- Implement temperature scaling (post-training calibration)
- Collect more diverse training data
- Use mixup or other regularization techniques

```python
# Temperature scaling example
import torch.nn.functional as F

# Find optimal temperature on validation set
def find_temperature(logits, targets):
    temps = torch.linspace(0.1, 5.0, 50)
    best_temp = 1.0
    best_ece = float('inf')

    for temp in temps:
        scaled_probs = F.softmax(logits / temp, dim=1)
        ece = calculate_ece(scaled_probs, targets)
        if ece < best_ece:
            best_ece = ece
            best_temp = temp

    return best_temp

# Apply during inference
scaled_logits = logits / temperature
calibrated_probs = F.softmax(scaled_logits, dim=1)
```

#### 5. High Class Imbalance

Check class distribution:

```python
import pandas as pd

df = pd.read_csv("datasets/full_dataset_metadata.csv")
print(df["quality_class"].value_counts())
```

**Solutions:**

- Use weighted sampling during training
- Apply class weights to loss function
- Generate more synthetic samples for underrepresented classes

#### 6. Low Performance on Specific Issue

```bash
# Check per-issue metrics in report
grep -A 5 "Per-Issue Metrics" evaluation_results/*_report.txt

# If specific issue (e.g., "Blur") has F1 < 0.5:
# 1. Collect more examples with that issue
# 2. Increase augmentation strength for that degradation type
# 3. Adjust issue detection threshold
```

### Performance Benchmarks

Use these benchmarks to assess your model:

| Metric | Minimum Acceptable | Good | Excellent |
|--------|-------------------|------|-----------|
| Quality Score MAE | < 0.15 | < 0.10 | < 0.08 |
| Quality Score R² | > 0.70 | > 0.80 | > 0.90 |
| Classification Accuracy | > 0.75 | > 0.85 | > 0.90 |
| Balanced Accuracy | > 0.70 | > 0.80 | > 0.85 |
| Binary F1 (Accept/Reject) | > 0.80 | > 0.85 | > 0.90 |
| False Reject Rate (FNR) | < 0.15 | < 0.10 | < 0.05 |
| False Accept Rate (FPR) | < 0.10 | < 0.05 | < 0.03 |
| Calibration ECE | < 0.15 | < 0.10 | < 0.05 |
| Issue Detection Macro F1 | > 0.65 | > 0.75 | > 0.85 |

### Getting Help

If you encounter issues not covered here:

1. Check the [main README](README.md) for general setup
2. Review the [training guide](TRAINING_GUIDE.md) if model quality is poor
3. Open an issue on GitHub with:
   - Evaluation command used
   - Full error message
   - Evaluation metrics (if available)
   - Environment details (`python --version`, `torch --version`)

---

## Next Steps

After evaluating your model:

### ✅ If Performance is Good

1. **Deploy to Production**
   - Set up inference API
   - Integrate with TMS workflow
   - Configure monitoring and alerts

2. **Create Documentation**
   - Document expected performance
   - Create user guides for drivers
   - Set up feedback collection

### ⚠️ If Performance Needs Improvement

1. **Diagnose Issues**
   - Use notebook to analyze errors
   - Check per-class performance
   - Review worst predictions

2. **Collect More Data**
   - Focus on underperforming classes
   - Add real production documents
   - Balance class distribution

3. **Retrain with Improvements**
   - Adjust hyperparameters
   - Implement class balancing
   - Add calibration techniques
   - Try different backbones

4. **Re-evaluate**
   - Run evaluation again
   - Compare metrics with previous version
   - Track improvements over time

---

## Summary

The evaluation suite provides comprehensive analysis of your document quality model across:

- ✅ Regression and classification performance
- ✅ Accept/reject decision quality
- ✅ Model calibration
- ✅ Multi-label issue detection
- ✅ Visual explanations (Grad-CAM)

Use both the CLI script for quick evaluations and the notebook for deep analysis and debugging.

For production deployment, ensure:

- Balanced accuracy > 0.80
- ECE < 0.10
- False reject rate < 0.10
- Performance validated on real-world test set
