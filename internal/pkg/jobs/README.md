<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
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

### Job Notifications

The job system automatically generates notifications for job completions and failures. Notifications can be customized per job type.

#### Default Notification Behavior

By default, notifications use templated messages:

- **Title**: `"Job Type Completed"` or `"Job Type Failed"`
- **Message**: `"Job type job {jobID} has completed successfully: {result}"`

#### Custom Notification Messages

For jobs that need specific notification formats, you can enable custom messages in the `JobNotificationConfig`:

```go
// In job_registry.go
"delay_shipment": {
    EventType:        notification.EventJobShipmentDelay,
    Priority:         notification.PriorityMedium,
    FailurePriority:  notification.PriorityHigh,
    TitleTemplate:    "Shipment Delay Notice!",  // Custom static title
    MessageTemplate:  "%s",                       // Only show the result
    Tags:             []string{"job", "shipment", "delay"},
    UseCustomTitle:   true,    // Enable custom title (no status suffix)
    UseCustomMessage: true,    // Enable custom message (only result shown)
},
```

When `UseCustomTitle` is `true`:

- The title will be exactly what's in `TitleTemplate` without appending status

When `UseCustomMessage` is `true`:

- The message will only format with the job result, not the job ID or status
- This is useful when the job result already contains all necessary information

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

## Creating a New Job - Step by Step Guide

### 1. Define the Job Type

Add your new job type to `types.go`:

```go
const (
    // ... existing job types
    JobTypeYourNewJob JobType = "category:your_job_name"
)
```

### 2. Create the Payload Structure

Add your payload to `types.go`:

```go
type YourJobPayload struct {
    BasePayload
    // Add your specific fields here
    YourField string `json:"yourField"`
}
```

### 3. Create the Job Handler

Create a new file in `handlers/your_job.go`:

```go
package handlers

import (
    "context"
    "github.com/emoss08/trenova/internal/pkg/jobs"
    "github.com/hibiken/asynq"
    "go.uber.org/fx"
)

type YourJobHandlerParams struct {
    fx.In

    Logger *logger.Logger
    // Add your dependencies here
}

type YourJobHandler struct {
    l *zerolog.Logger
    // Add your dependencies here
}

func NewYourJobHandler(p YourJobHandlerParams) jobs.JobHandler {
    log := p.Logger.With().
        Str("handler", "your_job").
        Logger()

    return &YourJobHandler{
        l: &log,
    }
}

func (h *YourJobHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
    var payload jobs.YourJobPayload
    if err := jobs.UnmarshalPayload(task.Payload(), &payload); err != nil {
        return err
    }

    // Implement your job logic here

    return nil
}

func (h *YourJobHandler) JobType() jobs.JobType {
    return jobs.JobTypeYourNewJob
}
```

### 4. Register the Handler in Infrastructure Module

Update `internal/infrastructure/jobs/module.go`:

```go
var Module = fx.Module(
    "jobs",
    fx.Provide(
        // ... existing providers

        // Add your handler
        fx.Annotate(
            handlers.NewYourJobHandler,
            fx.As(new(jobs.JobHandler)),
            fx.ResultTags(`group:"job_handlers"`),
        ),
    ),
    // ... rest of module
)
```

### 5. Configure the Queue (IMPORTANT!)

If using a custom queue, update `service.go` to include it in the server config:

```go
Queues: map[string]int{
    QueueCritical:   6,
    QueueShipment:   2,  // If using shipment queue
    QueuePattern:    1,
    QueueCompliance: 1,
    QueueDefault:    1,
    // Add your queue here if using a custom one
},
```

### 6. Create Job Options (Optional)

Add to `types.go` if you need custom options:

```go
func YourJobOptions() *JobOptions {
    return &JobOptions{
        Queue:    QueueShipment, // or your custom queue
        Priority: PriorityNormal,
        MaxRetry: 3,
    }
}
```

### 7. Add Scheduling Method (Optional)

Add to `service.go` if you want a dedicated scheduling method:

```go
func (js *JobService) ScheduleYourJob(
    payload *YourJobPayload,
    opts *JobOptions,
) (*asynq.TaskInfo, error) {
    if opts == nil {
        opts = YourJobOptions()
    }

    payload.JobID = pulid.MustNew("job_").String()
    payload.Timestamp = timeutils.NowUnix()

    return js.Enqueue(JobTypeYourNewJob, payload, opts)
}
```

### 8. Add to Cron Scheduler (If Recurring)

Update `scheduler/cronscheduler.go`:

```go
func (cs *CronScheduler) ScheduleYourRecurringJob() error {
    task := asynq.NewTask(
        string(jobs.JobTypeYourNewJob),
        cs.createYourJobPayload(),
    )

    entryID, err := cs.scheduler.Register("@every 5m", task,
        asynq.Queue(jobs.QueueShipment), // MUST match handler's queue!
        asynq.MaxRetry(2),
    )

    // Don't forget to call this in Start()
    return err
}
```

## Common Pitfalls to Avoid

1. **Queue Mismatch**: Ensure the queue in scheduler matches the queue in job options
2. **Missing Queue Configuration**: Add custom queues to the server's Queues map
3. **Wrong Payload Type**: Double-check you're unmarshaling the correct payload type
4. **Handler Not Registered**: Ensure handler is added to infrastructure module
5. **Missing Dependencies**: Verify all handler dependencies are available via DI

## Debugging Tips

1. Check logs for "job enqueued successfully" messages
2. Verify the queue name in logs matches your configuration
3. Use Asynq Web UI to inspect pending/failed jobs
4. Check Redis directly: `redis-cli KEYS "asynq:*"`
5. Enable debug logging in Asynq for more details

## Testing Your Job

```go
// Manual test
ctx := context.Background()
jobService.ScheduleYourJob(&YourJobPayload{
    BasePayload: jobs.BasePayload{
        OrganizationID: orgID,
        BusinessUnitID: buID,
        UserID:         userID,
    },
    YourField: "test value",
}, nil)
```

## Next Steps

1. **Add job service to your infrastructure module**
2. **Integrate shipment triggers into shipment service**
3. **Configure Redis connection for Asynq**
4. **Set up Asynq web UI for monitoring**
5. **Test pattern detection with sample shipment data**
