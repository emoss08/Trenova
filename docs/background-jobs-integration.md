# Background Jobs Integration Guide

## Overview

The background job service automatically detects patterns in shipment data and creates dedicated lane suggestions. Here's how to integrate it into your application.

## How It Works

When users create repetitive shipments:

```
User creates shipment → Automatic pattern analysis (30s delay) → Suggestions created if patterns found
```

### Pattern Detection Flow

1. **Shipment Created**: `OnShipmentCreated()` triggers pattern analysis
2. **Background Processing**: Job analyzes last 60 days of shipments for patterns
3. **Suggestion Generation**: Creates suggestions if patterns meet confidence thresholds
4. **User Notification**: Suggestions appear in dashboard for review

## Integration Steps

### 1. Add Job Service to Bootstrap

```go
// In internal/bootstrap/app.go or infrastructure module
import "github.com/emoss08/trenova/internal/infrastructure/jobs"

var Module = fx.Module(
    "infrastructure", 
    // ... existing modules
    jobs.Module, // Add this line
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

func (ss *ShipmentService) Create(ctx context.Context, shipment *shipment.Shipment) (*shipment.Shipment, error) {
    // ... create shipment logic
    
    // Trigger automatic pattern analysis
    go func() {
        if err := ss.shipmentTrigger.OnShipmentCreated(ctx, createdShipment); err != nil {
            ss.logger.Error().Err(err).Msg("failed to trigger pattern analysis")
        }
    }()
    
    return createdShipment, nil
}

func (ss *ShipmentService) UpdateStatus(ctx context.Context, shipmentID pulid.ID, newStatus shipment.Status) error {
    // ... update logic
    
    // Trigger status change jobs
    go func() {
        if err := ss.shipmentTrigger.OnShipmentStatusChanged(ctx, shipment, oldStatus, newStatus); err != nil {
            ss.logger.Error().Err(err).Msg("failed to trigger status change jobs")
        }
    }()
    
    return nil
}
```

### 3. Manual Pattern Analysis (Optional)

```go
// In your API handler for manual triggers
func (h *Handler) TriggerPatternAnalysis(c *fiber.Ctx) error {
    reqCtx, err := appctx.WithRequestContext(c)
    if err != nil {
        return h.eh.HandleError(c, err)
    }

    customerID := c.Params("customerId")
    
    err = h.shipmentTrigger.TriggerPatternAnalysisForCustomer(
        reqCtx.Context,
        pulid.MustParse(customerID),
        reqCtx.OrgID,
        reqCtx.BuID, 
        reqCtx.UserID,
        "manual_admin_trigger",
    )
    
    if err != nil {
        return h.eh.HandleError(c, err)
    }
    
    return c.JSON(fiber.Map{"message": "Pattern analysis scheduled"})
}
```

## Configuration

### Redis Requirements

- Redis server running (already required for your cache)
- No additional configuration needed

### Job Queues

- **Critical**: Shipment notifications (60% workers)
- **Pattern**: Pattern analysis (20% workers)  
- **Default**: General tasks (20% workers)

### Scheduling

- **Daily**: Pattern analysis at 2 AM (last 7 days)
- **Weekly**: Comprehensive analysis Sunday 1 AM (last 30 days)
- **Real-time**: Triggered by shipment creation/completion

## Example Pattern Detection

### Scenario

Customer "ACME Corp" creates these shipments:

```
Ship 1: Los Angeles → Dallas (LTL, 5000 lbs)
Ship 2: Los Angeles → Dallas (LTL, 4800 lbs) 
Ship 3: Los Angeles → Dallas (LTL, 5200 lbs)
Ship 4: Los Angeles → Dallas (LTL, 4900 lbs)
```

### Result

After the 3rd shipment, the system:

1. Detects pattern (same route, equipment, customer)
2. Calculates confidence score (0.85 - high confidence)  
3. Creates suggestion: "Lane-LA-to-DAL"
4. Shows in dashboard for review

### User Actions

- **Accept**: Creates dedicated lane, assigns drivers/equipment
- **Reject**: Dismisses suggestion with reason
- **Ignore**: Suggestion expires after TTL

## Monitoring

### Asynq Web UI (Optional)

```bash
# Install Asynq web UI
go install github.com/hibiken/asynq/tools/asynq@latest

# Run monitoring dashboard  
asynq dash --redis-addr=localhost:6379
```

Access at: `http://localhost:8080`

### Key Metrics

- Pattern analysis completion time
- Suggestions created per day
- Acceptance rate by customer
- Queue processing status

## Benefits

1. **Zero Configuration**: Works automatically after integration
2. **Non-Blocking**: Never slows down shipment operations
3. **Smart Detection**: Configurable confidence thresholds
4. **User Control**: Accept/reject suggestions with reasons
5. **Scalable**: Redis-based queue handles high volume

## Next Steps

1. **Add jobs module to bootstrap**
2. **Integrate shipment triggers**
3. **Test with sample data**
4. **Configure monitoring dashboard**
5. **Train users on suggestion workflow**

The system will immediately start detecting patterns and creating suggestions once integrated!
