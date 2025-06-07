# Background Job Service

A comprehensive background job processing system built on Asynq (Redis-based) for the Trenova transportation management system.

## Overview

This job service enables automated pattern detection for dedicated lane suggestions, along with other background processing tasks. When users repeatedly create similar shipments, the system automatically analyzes patterns and suggests dedicated lanes for optimization.

## Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Shipment      │───▶│  Job Triggers   │───▶│   Job Queue     │
│   Events        │    │                 │    │   (Redis)       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                                        │
┌─────────────────┐    ┌─────────────────┐             │
│  Cron Scheduler │───▶│   Job Handlers  │◀────────────┘
│                 │    │                 │
└─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │  Pattern Analysis│
                       │  & Suggestions  │
                       └─────────────────┘
```

## Key Components

### 1. Job Service (`service.go`)

- **Job Scheduling**: Enqueue jobs immediately, with delay, or at specific times
- **Queue Management**: Multiple priority queues (critical, pattern, compliance, default)
- **Retry Logic**: Exponential backoff with configurable max retries
- **Job Deduplication**: Unique keys prevent duplicate analysis

### 2. Job Handlers (`handlers/`)

- **Pattern Analysis Handler**: Processes shipment patterns and creates suggestions
- **Expire Suggestions Handler**: Cleans up old expired suggestions

### 3. Triggers (`triggers/`)

- **Shipment Trigger**: Automatically schedules jobs based on shipment events
  - `OnShipmentCreated`: Triggers pattern analysis when new shipments are created
  - `OnShipmentCompleted`: Analyzes completed shipments for patterns
  - `OnShipmentStatusChanged`: Handles status update notifications

### 4. Scheduler (`scheduler/`)

- **Cron Scheduler**: Manages recurring jobs
  - Daily pattern analysis (2 AM)
  - Weekly comprehensive analysis (Sunday 1 AM)
  - Suggestion expiration cleanup (every 6 hours)

## Integration Guide

### 1. Add to Bootstrap Module

```go
// In internal/bootstrap/modules/infrastructure/module.go
import "github.com/emoss08/trenova/internal/infrastructure/jobs"

var Module = fx.Module(
    "infrastructure",
    // ... existing modules
    jobs.Module,
)
```

### 2. Integrate with Shipment Service

```go
// In your shipment service
type ShipmentServiceParams struct {
    fx.In
    
    // ... existing dependencies
    ShipmentTrigger triggers.ShipmentTriggerInterface
}

// In shipment creation method
func (ss *ShipmentService) Create(ctx context.Context, shipment *shipment.Shipment) error {
    // ... create shipment logic
    
    // Trigger background pattern analysis
    if err := ss.shipmentTrigger.OnShipmentCreated(ctx, shipment); err != nil {
        // Log error but don't fail shipment creation
        ss.logger.Error().Err(err).Msg("failed to trigger pattern analysis")
    }
    
    return nil
}
```

### 3. Manual Pattern Analysis

```go
// Trigger analysis for specific customer
err := shipmentTrigger.TriggerPatternAnalysisForCustomer(
    ctx, 
    customerID, 
    orgID, 
    buID, 
    userID, 
    "manual_trigger"
)
```

## Job Types

### Pattern Analysis Jobs

- **`pattern:analyze`**: Analyzes shipment patterns and creates suggestions
- **`pattern:expire_suggestions`**: Expires old suggestions based on TTL

### System Jobs

- **`shipment:status_update`**: Handles shipment status change notifications
- **`compliance:check`**: Performs compliance verification
- **`system:cleanup_temp_files`**: System maintenance tasks

## Configuration

### Queue Priorities

- **Critical Queue**: 60% of workers (shipment notifications, critical tasks)
- **Pattern Queue**: 20% of workers (pattern analysis)
- **Compliance Queue**: 10% of workers (compliance checks)
- **Default Queue**: 10% of workers (general tasks)

### Retry Strategy

- Exponential backoff: 1s, 2s, 4s, 8s, 16s
- Configurable max retries per job type
- Dead letter queue for failed jobs

## Monitoring

### Asynq Web UI

Access job monitoring at: `http://localhost:8080` (when running Asynq web UI)

### Key Metrics

- Job processing times
- Success/failure rates
- Queue depths
- Pattern analysis results

## Example Usage

### 1. Automatic Pattern Detection

When a user creates shipments:

```
User creates shipment → ShipmentTrigger.OnShipmentCreated() 
→ Schedules pattern analysis job (30s delay)
→ PatternAnalysisHandler processes job
→ Creates dedicated lane suggestions if patterns found
```

### 2. Scheduled Analysis

```
Daily 2 AM → CronScheduler triggers global pattern analysis
→ Analyzes last 7 days across all customers
→ Creates suggestions for newly detected patterns
```

### 3. Manual Triggers

```go
// From API endpoint or admin interface
_, err := jobService.SchedulePatternAnalysis(ctx, &jobs.PatternAnalysisPayload{
    BasePayload: jobs.BasePayload{
        OrganizationID: orgID,
        BusinessUnitID: buID,
        UserID:         userID,
    },
    CustomerID:    &customerID,
    StartDate:     startDate,
    EndDate:       endDate,
    MinFrequency:  3,
    TriggerReason: "manual",
}, nil)
```

## Benefits

1. **Automatic Detection**: No manual intervention needed for pattern detection
2. **Scalable Processing**: Redis-based job queue handles high throughput
3. **Configurable Analysis**: Adjust frequency, confidence, and time windows
4. **Non-blocking**: Shipment operations never blocked by analysis
5. **Monitoring**: Full visibility into job processing and results
6. **Extensible**: Easy to add new job types for future features

## Next Steps

1. **Add job service to your infrastructure module**
2. **Integrate shipment triggers into shipment service**
3. **Configure Redis connection for Asynq**
4. **Set up Asynq web UI for monitoring**
5. **Test pattern detection with sample shipment data**
