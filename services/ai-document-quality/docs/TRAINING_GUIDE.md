# Document Quality Assessment - Training Guide

Complete guide for training the document quality assessment model with best practices.

## Table of Contents

1. [Quick Start](#quick-start)
2. [Setup](#setup)
3. [Training Configuration](#training-configuration)
4. [Class Imbalance Handling](#class-imbalance-handling)
5. [Training Process](#training-process)
6. [Monitoring Training](#monitoring-training)
7. [Troubleshooting](#troubleshooting)
8. [Advanced Topics](#advanced-topics)

---

## Quick Start

### Minimal Training

```bash
# 1. Create virtual environment and install dependencies
python -m venv .venv
source .venv/bin/activate  # or `.venv\Scripts\activate` on Windows
pip install -r requirements.txt

# 2. Prepare dataset (if not already done)
python -m src.data.dataset \
    --input-dir documents \
    --output-dir datasets \
    --mode synthetic

# 3. Train with default config
python scripts/train_production.py

# 4. Monitor training (in another terminal)
mlflow ui
# Open http://localhost:5000
```

### Full Training with Custom Config

```bash
# Train with specific config
python scripts/train_production.py \
    --config config/best_training.yaml \
    --experiment production_model \
    --run-name v1.0 \
    --epochs 50
```

---

## Setup

### 1. Install Dependencies

```bash
# Create virtual environment
python -m venv .venv
source .venv/bin/activate

# Install all dependencies
pip install -r requirements.txt

# Verify installation
python -c "import torch; print(f'PyTorch: {torch.__version__}')"
python -c "import torchvision; print(f'TorchVision: {torchvision.__version__}')"
```

### 2. Prepare Dataset

If you don't have a dataset yet:

```bash
# Option 1: Generate synthetic dataset from clean documents
python -m src.data.dataset \
    --input-dir documents \
    --output-dir datasets \
    --mode synthetic \
    --train-ratio 0.7 \
    --val-ratio 0.2 \
    --test-ratio 0.1

# Option 2: Use real labeled documents
python -m src.data.dataset \
    --input-dir labeled_documents \
    --output-dir datasets \
    --mode real
```

**Dataset Structure:**

```
datasets/
├── default/
│   ├── train/
│   │   ├── train_metadata.csv
│   │   └── *.jpg
│   ├── val/
│   │   ├── val_metadata.csv
│   │   └── *.jpg
│   └── test/
│       ├── test_metadata.csv
│       └── *.jpg
├── full_dataset_metadata.csv
└── dataset_summary.txt
```

### 3. Check Dataset Quality

```bash
# View dataset summary
cat datasets/dataset_summary.txt

# Check for class imbalance
python -c "
import pandas as pd
df = pd.read_csv('datasets/full_dataset_metadata.csv')
print('Class Distribution:')
print(df['quality_class'].value_counts().sort_index())
print(f'\nTotal samples: {len(df)}')
"
```

**⚠️ Warning Signs:**

- Any class has < 50 samples
- Class imbalance ratio > 10:1 (most common / least common)
- < 500 total training samples

---

## Training Configuration

### Configuration Files

Configurations are in YAML format in the `config/` directory:

- `config/default.yaml` - Basic configuration
- `config/best_training.yaml` - Recommended production settings

### Key Configuration Sections

#### 1. Model Architecture

```yaml
model:
  backbone: "efficientnet_b0"  # Options: efficientnet_b0, resnet50, mobilenet_v3_small
  num_quality_classes: 5
  num_issue_classes: 10
  hidden_dim: 256
  dropout_rate: 0.5
  use_attention: true
  freeze_backbone_layers: 5  # Freeze first N layers for transfer learning
```

**Backbone Selection:**

- `efficientnet_b0`: Best accuracy, slower training (recommended)
- `mobilenet_v3_small`: Faster inference, good for production
- `resnet50`: Balance of speed and accuracy

#### 2. Training Hyperparameters

```yaml
training:
  num_epochs: 50
  batch_size: 32
  learning_rate: 0.001
  backbone_lr_factor: 0.1  # Use 10x lower LR for backbone
  weight_decay: 0.01
  gradient_clip_norm: 1.0
  patience: 15  # Early stopping patience

  # Class imbalance handling
  use_balanced_sampling: true  # Recommended!

  # Loss weights
  regression_weight: 10.0
  classification_weight: 1.0
  issue_weight: 1.0
  consistency_weight: 0.3

  # Advanced features
  use_focal_loss: true
  use_ordinal_regression: true

  # Learning rate scheduling
  scheduler_type: "cosine"  # Options: cosine, cosine_warm_restarts, plateau
```

**Tuning Tips:**

- **batch_size**: Larger is better (32-64) but limited by GPU memory
- **learning_rate**: Start with 0.001, reduce if loss diverges
- **backbone_lr_factor**: Keep at 0.1 to avoid destroying pretrained features
- **patience**: Increase for large datasets (15-20), decrease for quick experiments (5-10)

#### 3. Data Augmentation

```yaml
dataset:
  augmentation:
    use_domain_augmentations: true  # Transportation-specific augmentations
    mixup_alpha: 0.2  # Mixup augmentation strength
    cutmix_alpha: 1.0  # CutMix augmentation strength

  num_workers: 4  # DataLoader workers (increase for faster data loading)
  train_ratio: 0.7
  val_ratio: 0.2
  test_ratio: 0.1
```

#### 4. MLflow Tracking

```yaml
mlflow:
  tracking_uri: "mlruns"  # Or remote server: "http://mlflow-server:5000"
  experiment_name: "document-quality-training"
  run_name: null  # Auto-generated if not specified
```

### Example Configurations

#### Fast Experimentation Config

```yaml
# config/fast_experiment.yaml
model:
  backbone: "mobilenet_v3_small"
  hidden_dim: 128

training:
  num_epochs: 20
  batch_size: 64
  learning_rate: 0.002
  patience: 5
```

#### High-Accuracy Config

```yaml
# config/high_accuracy.yaml
model:
  backbone: "efficientnet_b3"
  hidden_dim: 512
  dropout_rate: 0.3

training:
  num_epochs: 100
  batch_size: 16
  learning_rate: 0.0005
  patience: 20
  use_balanced_sampling: true
```

---

## Class Imbalance Handling

The training script automatically handles class imbalance using multiple strategies:

### 1. Balanced Batch Sampling (Recommended)

**Enabled by default** (`use_balanced_sampling: true`)

Ensures each batch contains roughly equal numbers of each quality class:

```python
# Automatically applied in train_production.py
sampler = BalancedBatchSampler(
    dataset=train_dataset,
    batch_size=32,
    num_classes=5,
)
```

**Benefits:**

- Model sees all classes equally during training
- No need to manually adjust class weights
- Works well with large imbalances (10:1 or higher)

### 2. Class Weighting

Automatically calculates and applies inverse frequency weights:

```bash
# Output during training:
Class weights calculated:
  High        : 1.2500 (count: 200)
  Good        : 3.4722 (count: 72)
  Moderate    : 2.5510 (count: 98)
  Poor        : 0.7962 (count: 314)
  Very Poor   : 0.6010 (count: 416)
```

**How it works:**

- Classes with fewer samples get higher weights
- Loss for minority classes contributes more to gradient
- Prevents model from ignoring rare classes

### 3. Focal Loss

Reduces weight of easy examples, focuses on hard ones:

```yaml
training:
  use_focal_loss: true  # Recommended for imbalanced datasets
```

### Monitoring Class Balance

Check per-class accuracy during training:

```bash
# View MLflow dashboard
mlflow ui

# Check metrics:
# - classification/balanced_accuracy (should be high)
# - Per-class accuracy in logs
```

**Target:** Balanced accuracy should be within 5% of overall accuracy. If gap is larger, class imbalance is still an issue.

---

## Training Process

### Basic Training

```bash
# Train with default config
python scripts/train_production.py

# Output:
# ================================================================================
# DOCUMENT QUALITY ASSESSMENT - PRODUCTION TRAINING
# ================================================================================
#
# Loading config from: config/best_training.yaml
# Using device: cuda
#
# ✓ MLflow tracking initialized
#   Experiment: document-quality-training
#   Run: run_20250930_143022
#
# ✓ Datasets loaded
#   Train: 770 images
#   Val:   220 images
#   Test:  110 images
#
# Using balanced batch sampling to address class imbalance
# ✓ Dataloaders created
#   Batch size: 32
#   Train batches: 25
#   Val batches: 7
#
# ✓ Model created
#   Backbone: efficientnet_b0
#   Total parameters: 4,234,567
#   Trainable parameters: 4,234,567
#
# Class weights calculated:
#   High        : 1.2500 (count: 200)
#   Good        : 3.4722 (count: 72)
#   Moderate    : 2.5510 (count: 98)
#   Poor        : 0.7962 (count: 314)
#   Very Poor   : 0.6010 (count: 416)
#
# ================================================================================
# STARTING TRAINING
# ================================================================================
#   Total epochs: 50
#   Patience: 15
#   Device: cuda
#   Output directory: models/20250930_143022
#
# Epoch 1 [Train]: 100%|████████████████| 25/25 [00:15<00:00,  1.62it/s]
# Epoch 1 [Val]:   100%|████████████████| 7/7 [00:02<00:00,  3.15it/s]
#
# Epoch 1/50
#   Train Loss: 2.3456
#   Train MAE:  0.2345
#   Train Acc:  0.4567
#   Val Loss:   2.1234
#   Val MAE:    0.2123
#   Val Acc:    0.5123
#   LR:         0.001000
#   ✓ New best model!
# ...
```

### Advanced Training Options

#### Resume from Checkpoint

```bash
python scripts/train_production.py \
    --resume models/20250930_143022/checkpoints/checkpoint_epoch_10.pth
```

#### Custom Output Directory

```bash
python scripts/train_production.py \
    --output-dir models/my_experiment \
    --experiment my_experiment \
    --run-name trial_01
```

#### Override Epochs

```bash
# Quick test run
python scripts/train_production.py --epochs 5

# Long training
python scripts/train_production.py --epochs 100
```

#### Create Dataset on the Fly

```bash
# Train on new documents (will create dataset automatically)
python scripts/train_production.py --data-dir documents/new_batch/
```

#### CPU-Only Training

```bash
python scripts/train_production.py --device cpu
```

### Training Output

**Directory Structure:**

```
models/20250930_143022/
├── checkpoints/
│   ├── checkpoint_epoch_5.pth
│   ├── checkpoint_epoch_10.pth
│   ├── checkpoint_epoch_15.pth
│   └── ...
└── best_model.pth  # Best model based on validation loss
```

**Checkpoint Contents:**

- Model weights
- Optimizer state
- Scheduler state
- Training metrics
- Configuration

---

## Monitoring Training

### MLflow Dashboard

```bash
# Start MLflow UI (in separate terminal)
mlflow ui

# Open in browser
http://localhost:5000
```

**Dashboard Features:**

- Compare multiple runs
- View metrics over time
- Download models
- Compare hyperparameters

**Key Metrics to Watch:**

1. **Loss Curves**
   - `train/total` should decrease steadily
   - `val/total` should decrease (gap from train indicates overfitting)
   - If val loss increases while train decreases → overfitting

2. **Accuracy Metrics**
   - `train/accuracy` vs `val/accuracy`
   - `classification/balanced_accuracy` (should be close to regular accuracy)

3. **Learning Rate**
   - Should decrease over time (with cosine scheduler)
   - Check if it's too high (loss explodes) or too low (slow learning)

4. **Regression Metrics**
   - `regression/mae` should be < 0.10 for good model
   - `regression/rmse` should be < 0.15

### Real-Time Monitoring

**Console Output:**

```bash
Epoch 1 [Train]: 100%|████████████| 25/25 [00:15<00:00,  1.62it/s]
Epoch 1 [Val]:   100%|██████████████| 7/7 [00:02<00:00,  3.15it/s]

Epoch 1/50
  Train Loss: 2.3456
  Train MAE:  0.2345
  Train Acc:  0.4567
  Val Loss:   2.1234
  Val MAE:    0.2123
  Val Acc:    0.5123
  LR:         0.001000
  ✓ New best model!
```

**What to Look For:**

- ✅ **Healthy Training:** Loss decreases, accuracy increases, occasional "New best model!"
- ⚠️ **Overfitting:** Train accuracy >> val accuracy (gap > 10%)
- ❌ **Not Learning:** Loss stays constant or increases

### Stopping Criteria

**Early Stopping:**
Automatically stops if no improvement for `patience` epochs:

```
⚠ Early stopping triggered after 15 epochs without improvement
```

**Manual Stop:**
Press `Ctrl+C` to stop gracefully (saves checkpoint):

```
⚠ Training interrupted by user
  Checkpoint saved to: models/.../checkpoints/interrupted.pth
```

---

## Troubleshooting

### Common Issues

#### 1. CUDA Out of Memory

**Error:**

```
RuntimeError: CUDA out of memory
```

**Solutions:**

```bash
# Reduce batch size
# In config YAML:
training:
  batch_size: 16  # or 8

# Or use CPU (slower)
python scripts/train_production.py --device cpu

# Or enable gradient accumulation (advanced)
```

#### 2. Loss is NaN

**Error:**

```
Epoch 1/50
  Train Loss: nan
```

**Causes & Solutions:**

- **Learning rate too high** → Reduce to 0.0001
- **Bad data** → Check dataset for corrupted images
- **Numerical instability** → Add gradient clipping:

  ```yaml
  training:
    gradient_clip_norm: 1.0
  ```

#### 3. Model Not Learning

**Symptoms:**

- Loss stays constant
- Accuracy doesn't improve
- Predictions all the same

**Solutions:**

```yaml
# 1. Increase learning rate
training:
  learning_rate: 0.005  # Higher

# 2. Unfreeze backbone earlier
model:
  freeze_backbone_layers: 0  # Train all layers

# 3. Check class weights are applied
training:
  use_balanced_sampling: true
```

#### 4. Overfitting

**Symptoms:**

- Train accuracy: 95%, Val accuracy: 70%
- Val loss increases while train loss decreases

**Solutions:**

```yaml
# 1. Increase regularization
model:
  dropout_rate: 0.6  # Higher dropout

training:
  weight_decay: 0.05  # Higher weight decay

# 2. More data augmentation
dataset:
  augmentation:
    use_domain_augmentations: true
    mixup_alpha: 0.4
    cutmix_alpha: 1.0

# 3. Early stopping with lower patience
training:
  patience: 10
```

#### 5. Slow Training

**Issue:** Training taking too long

**Solutions:**

```bash
# 1. Increase num_workers
# In config:
dataset:
  num_workers: 8  # Match CPU cores

# 2. Use smaller backbone
model:
  backbone: "mobilenet_v3_small"

# 3. Reduce image size (advanced)
# Edit dataset.py transform

# 4. Use mixed precision (advanced)
# Requires AMP support
```

#### 6. Dataset Not Found

**Error:**

```
Dataset not found. Please provide --data-dir or create dataset first.
```

**Solution:**

```bash
# Create dataset
python -m src.data.dataset \
    --input-dir documents \
    --output-dir datasets

# Or specify data directory
python scripts/train_production.py --data-dir documents/
```

### Debugging Checklist

Before reporting an issue:

1. ✅ Check dataset exists and has correct structure
2. ✅ Verify config file is valid YAML
3. ✅ Check GPU memory usage (`nvidia-smi`)
4. ✅ Review last 50 lines of logs
5. ✅ Try with minimal config (fast_experiment.yaml)
6. ✅ Verify PyTorch version: `python -c "import torch; print(torch.__version__)"`

---

## Advanced Topics

### Transfer Learning

**Fine-tune from pretrained model:**

```python
# In config
model:
  freeze_backbone_layers: 10  # Freeze more layers

training:
  backbone_lr_factor: 0.01  # Very low LR for backbone
```

### Multi-GPU Training

```bash
# Use all available GPUs
CUDA_VISIBLE_DEVICES=0,1,2,3 python scripts/train_production.py

# Or specific GPUs
CUDA_VISIBLE_DEVICES=0,2 python scripts/train_production.py
```

### Hyperparameter Tuning

**Manual Grid Search:**

```bash
# Try different learning rates
for lr in 0.0001 0.001 0.01; do
  python scripts/train_production.py \
    --experiment hyperparam_search \
    --run-name lr_${lr} \
    --config <(sed "s/learning_rate:.*/learning_rate: $lr/" config/best_training.yaml)
done

# Compare in MLflow UI
```

### Custom Loss Weights

```yaml
training:
  regression_weight: 20.0  # Emphasize quality score accuracy
  classification_weight: 1.0
  issue_weight: 0.5  # De-emphasize issue detection
```

### Curriculum Learning

Train on easy examples first, gradually add harder ones (implemented in `advanced_strategies.py`).

### Test-Time Augmentation

```python
# During inference, average predictions over augmented versions
# Improves robustness (implemented in evaluation)
```

---

## Best Practices Summary

### ✅ Do's

1. **Always use balanced sampling** for imbalanced datasets
2. **Monitor MLflow dashboard** during training
3. **Start with default config** then customize
4. **Save checkpoints frequently** (every 5 epochs)
5. **Use early stopping** to prevent overfitting
6. **Validate on real data** before deploying
7. **Track experiments** with descriptive names
8. **Use gradient clipping** for stable training

### ❌ Don'ts

1. **Don't skip evaluation** - always run evaluation after training
2. **Don't ignore class imbalance** - use balanced sampling
3. **Don't use same LR for backbone and heads** - use lower LR for backbone
4. **Don't train on validation set** - keep it separate
5. **Don't deploy without calibration check** - run full evaluation
6. **Don't use too small datasets** - aim for 500+ samples minimum
7. **Don't forget to save best model** - use checkpoint system

---

## Next Steps

After training:

1. **Evaluate Model**

   ```bash
   python scripts/evaluate_model.py --model models/.../best_model.pth --explain
   ```

2. **Review Metrics**
   - Check evaluation report
   - Ensure ECE < 0.10 (calibration)
   - Verify balanced accuracy > 0.85

3. **Test on Real Data**
   - Upload production documents
   - Run evaluation on real test set
   - Check false reject rate

4. **Deploy or Retrain**
   - If metrics are good → Deploy to production
   - If metrics are poor → Collect more data and retrain

---

## Support

For issues:

1. Check [Troubleshooting](#troubleshooting)
2. Review [EVALUATION_GUIDE.md](EVALUATION_GUIDE.md)
3. Open GitHub issue with:
   - Training command used
   - Config file
   - Error logs
   - System info
