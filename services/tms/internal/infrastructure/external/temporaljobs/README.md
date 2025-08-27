# Temporal Jobs Infrastructure

This directory contains the Temporal workflow implementation for Trenova's background job processing system. We are gradually migrating from Asynq to Temporal for improved workflow orchestration, better error handling, and enhanced observability.

## Architecture Overview

The Temporal implementation follows a domain-driven design with clear separation of concerns:

```
temporaljobs/
├── client.go           # Temporal client initialization
├── worker.go           # Worker management and lifecycle
├── registry.go         # Centralized workflow/activity registration
├── scheduler/          # Cron-based scheduled workflows
├── domains/            # Domain-specific workflows and activities
│   ├── email/         # Email sending workflows
│   ├── patterns/      # Pattern analysis workflows
│   ├── compliance/    # Compliance check workflows
│   └── shipment/      # Shipment-related workflows
├── payloads/          # Shared payload definitions
└── types.go           # Common types and constants
```

## Core Components

### 1. Registry (`registry.go`)

The Registry provides centralized management of workflows, activities, and schedules:

- **Thread-safe registration** of workflows and activities
- **Metadata tracking** for each registered component
- **Bulk application** to workers
- **Runtime discovery** of available workflows

```go
registry := NewRegistry(params)
registry.RegisterWorkflow(&WorkflowDefinition{...})
registry.RegisterActivity(&ActivityDefinition{...})
registry.ApplyToWorker(worker)
```

### 2. Worker (`worker.go`)

The Worker manages the lifecycle of Temporal workers:

- **Automatic registration** of workflows and activities
- **Graceful shutdown** handling
- **Health monitoring** and statistics
- **Error recovery** and panic handling

```go
worker := NewWorker(params)
worker.Start() // Non-blocking
defer worker.Stop()
```

### 3. Client (`client.go`)

Provides the Temporal client for workflow execution:

- **Connection management** to Temporal server
- **Namespace configuration**
- **Retry policies** and timeouts
- **Workflow execution** interface

### 4. Scheduler (`scheduler/scheduler.go`)

Manages recurring workflows using Temporal's schedule feature:

- **Cron-based scheduling**
- **Dynamic schedule management**
- **Schedule monitoring** and updates
- **Graceful shutdown** of schedules

## Domain Organization

Each domain package follows a consistent structure:

### Email Domain (`domains/email/`)

- `workflows.go`: Email sending and queue processing workflows
- `activities.go`: Email service integration activities

### Patterns Domain (`domains/patterns/`)

- `workflows.go`: Pattern analysis and suggestion workflows
- `activities.go`: Pattern detection and analysis activities

### Compliance Domain (`domains/compliance/`)

- `workflows.go`: Compliance check workflows
- `activities.go`: DOT and Hazmat compliance activities

### Shipment Domain (`domains/shipment/`)

- `workflows.go`: Shipment duplication and management workflows
- `activities.go`: Shipment repository operations

## Workflow Patterns

### Basic Workflow Structure

```go
func MyWorkflow(ctx workflow.Context, payload *MyPayload) error {
    logger := workflow.GetLogger(ctx)
    
    // Configure activity options
    ao := workflow.ActivityOptions{
        StartToCloseTimeout: 30 * time.Second,
        HeartbeatTimeout:    5 * time.Second,
        RetryPolicy: &temporal.RetryPolicy{
            InitialInterval:    time.Second,
            BackoffCoefficient: 2.0,
            MaximumInterval:    30 * time.Second,
            MaximumAttempts:    3,
        },
    }
    ctx = workflow.WithActivityOptions(ctx, ao)
    
    // Execute activities
    var result string
    err := workflow.ExecuteActivity(ctx, MyActivity, payload).Get(ctx, &result)
    if err != nil {
        return err
    }
    
    return nil
}
```

### Activity with Dependencies

```go
type ActivityProvider struct {
    service MyService
    logger  *zerolog.Logger
}

func (p *ActivityProvider) MyActivity(ctx context.Context, payload *MyPayload) (string, error) {
    // Record heartbeat for long-running activities
    activity.RecordHeartbeat(ctx, "processing")
    
    // Use injected dependencies
    result, err := p.service.Process(ctx, payload)
    if err != nil {
        return "", err
    }
    
    return result, nil
}
```

## Migration from Asynq

We're migrating from Asynq to Temporal in phases:

### Phase 1: Infrastructure Setup ✅

- Temporal client, worker, and registry implementation
- Domain-based organization structure
- Basic workflows (email, patterns, compliance)

### Phase 2: Gradual Migration (In Progress)

- Implementing adapter layer for backward compatibility
- Migrating workflows one at a time
- Starting with duplicate shipment workflow

### Phase 3: Complete Migration

- Remove Asynq dependencies
- Full Temporal adoption
- Enhanced monitoring and observability

## Task Queues

Temporal uses task queues to route workflows to appropriate workers:

- `critical-tasks`: High-priority operations
- `email-tasks`: Email sending operations
- `shipment-tasks`: Shipment processing
- `pattern-analysis-tasks`: Pattern detection
- `compliance-tasks`: Compliance checks
- `default-tasks`: General background jobs

## Configuration

The Temporal implementation uses environment variables for configuration:

```bash
TEMPORAL_HOST_PORT=localhost:7233
TEMPORAL_NAMESPACE=default
TEMPORAL_TASK_QUEUE=default-tasks
```

### Search Attributes Setup (Optional)

Search attributes allow you to query workflows based on custom fields. To enable them:

1. **Register search attributes in Temporal:**

```bash
# Create organization ID attribute
temporal operator search-attribute create \
  --namespace default \
  --name OrganizationId \
  --type Keyword

# Create user ID attribute  
temporal operator search-attribute create \
  --namespace default \
  --name UserId \
  --type Keyword

# Verify attributes were created
temporal operator search-attribute list
```

2. **Enable in adapter code:**
Once registered, uncomment the search attributes code in `adapter.go`:

```go
if duplicatePayload.OrganizationID.String() != "" {
    workflowOptions.SearchAttributes = map[string]any{
        "OrganizationId": duplicatePayload.OrganizationID.String(),
        "UserId":         duplicatePayload.UserID.String(),
    }
}
```

3. **Query workflows:**

```bash
# Find workflows for a specific organization
temporal workflow list \
  --query 'OrganizationId="org_123"'

# Find workflows by user
temporal workflow list \
  --query 'UserId="usr_456"'
```

## Monitoring and Observability

### Workflow Statistics

Access workflow statistics through the worker:

```go
stats := worker.GetStatistics()
// Returns: IsRunning, StartTime, WorkflowsStarted, WorkflowsFailed, etc.
```

### Temporal Web UI

Access the Temporal Web UI at `http://localhost:8080` to:

- View workflow executions
- Inspect workflow history
- Debug failed workflows
- Monitor schedule status

## Best Practices

1. **Idempotency**: Design activities to be idempotent
2. **Heartbeats**: Use heartbeats for long-running activities
3. **Error Handling**: Let Temporal handle retries, don't catch and retry manually
4. **Versioning**: Use workflow versioning for backward compatibility
5. **Timeouts**: Set appropriate timeouts for activities and workflows
6. **Logging**: Use workflow.GetLogger() for proper log correlation

## Adding New Workflows

To add a new workflow:

1. Create a new domain package if needed
2. Define workflow and activity functions
3. Create registration functions
4. Update `registration.go` to include new domain
5. Define payload types in `payloads/`
6. Add constants to `types.go`

Example:

```go
// domains/mydomain/workflows.go
func MyNewWorkflow(ctx workflow.Context, payload *MyPayload) error {
    // Implementation
}

func RegisterWorkflows() []WorkflowDefinition {
    return []WorkflowDefinition{
        {
            Name:        "MyNewWorkflow",
            Fn:          MyNewWorkflow,
            TaskQueue:   "my-tasks",
            Description: "Processes my domain logic",
        },
    }
}
```

## Troubleshooting

### Common Issues

1. **Worker not picking up tasks**: Check task queue names match between client and worker
2. **Workflows timing out**: Increase activity timeouts or add heartbeats
3. **Registration errors**: Ensure unique workflow/activity names
4. **Connection issues**: Verify Temporal server is running and accessible

### Debug Commands

```bash
# Check Temporal server status
temporal operator namespace list

# List workflows
temporal workflow list

# Describe a workflow
temporal workflow describe -w <workflow-id>

# View workflow history
temporal workflow show -w <workflow-id>
```

## Future Enhancements

- [ ] Workflow versioning strategy
- [ ] Enhanced retry policies per domain
- [ ] Workflow templates for common patterns
- [ ] Integration with OpenTelemetry
- [ ] Custom workflow search attributes
- [ ] Workflow testing framework
