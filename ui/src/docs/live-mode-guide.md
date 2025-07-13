# Live Mode Implementation Guide

This guide explains how to implement live mode (real-time updates) for data tables in the Trenova application.

## Overview

Live mode allows tables to receive real-time updates from the server using Server-Sent Events (SSE). When new data is available, users see a banner notification and can refresh to see the latest data.

## Architecture

### Backend (Go)
- SSE endpoint that streams real-time updates
- Polling-based approach for simplicity and reliability
- JSON-formatted event messages

### Frontend (React)
- Custom hooks for SSE connection management
- Integration with TanStack Query for data management
- Reusable banner component for notifications

## Implementation Steps

### 1. Backend - Add SSE Endpoint

Add a live stream handler to your existing handler:

```go
func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
    api := r.Group("/your-resource")
    
    // Add live endpoint
    api.Get("/live", rl.WithRateLimit(
        []fiber.Handler{h.liveStream},
        middleware.PerSecond(1), // 1 connection per second
    )...)
    
    // ... other routes
}

func (h *Handler) liveStream(c *fiber.Ctx) error {
    reqCtx, err := appctx.WithRequestContext(c)
    if err != nil {
        return h.errorHandler.HandleError(c, err)
    }

    // Set SSE headers
    c.Set("Content-Type", "text/event-stream")
    c.Set("Cache-Control", "no-cache")
    c.Set("Connection", "keep-alive")
    c.Set("Access-Control-Allow-Origin", "*")
    c.Set("Access-Control-Allow-Headers", "Cache-Control")

    // Create a ticker for polling
    ticker := time.NewTicker(2 * time.Second)
    defer ticker.Stop()

    // Track last timestamp to only send new entries
    lastTimestamp := time.Now().Unix()

    // Send initial connection event
    _, err = c.WriteString("event: connected\ndata: {\"status\":\"connected\"}\n\n")
    if err != nil {
        return err
    }

    // Handle client disconnect
    done := make(chan bool)
    defer close(done)
    
    go func() {
        <-c.Context().Done()
        done <- true
    }()

    for {
        select {
        case <-done:
            return nil
        case <-ticker.C:
            // Query for new entries since last check
            filter := &ports.LimitOffsetQueryOptions{
                TenantOpts: ports.TenantOptions{
                    BuID:   reqCtx.BuID,
                    OrgID:  reqCtx.OrgID,
                    UserID: reqCtx.UserID,
                },
                Limit:  10,
                Offset: 0,
                Query:  fmt.Sprintf("timestamp_after:%d", lastTimestamp),
            }

            result, err := h.yourService.List(c.UserContext(), filter)
            if err != nil {
                // Send error event but continue streaming
                errorData := map[string]string{"error": "Failed to fetch data"}
                errorJSON, _ := json.Marshal(errorData)
                c.WriteString(fmt.Sprintf("event: error\ndata: %s\n\n", errorJSON))
                continue
            }

            // Send new entries
            if len(result.Items) > 0 {
                for _, entry := range result.Items {
                    if entry.Timestamp > lastTimestamp {
                        lastTimestamp = entry.Timestamp
                    }

                    entryJSON, err := json.Marshal(entry)
                    if err != nil {
                        continue
                    }
                    _, err = c.WriteString(fmt.Sprintf("event: new-entry\ndata: %s\n\n", entryJSON))
                    if err != nil {
                        return err
                    }
                }
            }

            // Send heartbeat
            _, err = c.WriteString("event: heartbeat\ndata: {\"timestamp\":\"" + time.Now().Format(time.RFC3339) + "\"}\n\n")
            if err != nil {
                return err
            }
        }
    }
}
```

### 2. Frontend - Update Your Table Component

```tsx
import { DataTable } from "@/components/data-table/data-table";
import { LIVE_MODE_ENDPOINTS } from "@/types/live-mode";

export default function YourTable() {
  const columns = useMemo(() => getColumns(), []);

  return (
    <DataTable<YourDataType>
      resource={Resource.YourResource}
      name="Your Resource"
      link="/your-resource/"
      queryKey="your-resource-list"
      exportModelName="your-resource"
      columns={columns}
      liveMode={{
        enabled: true,
        endpoint: "/your-resource/live", // Add to LIVE_MODE_ENDPOINTS
      }}
    />
  );
}
```

### 3. Update Live Mode Configuration

Add your new endpoint to the constants:

```typescript
// In /types/live-mode.ts
export const LIVE_MODE_ENDPOINTS = {
  AUDIT_LOGS: '/audit-logs/live',
  YOUR_RESOURCE: '/your-resource/live', // Add this
} as const;
```

## Configuration Options

The `liveMode` prop accepts these options:

```typescript
liveMode={{
  enabled: true,                    // Enable/disable live mode
  endpoint: "/your-resource/live",  // SSE endpoint
  options: {
    pollInterval: 2000,             // Polling interval (ms)
    maxReconnectAttempts: 5,        // Max reconnection attempts
    showConnectionStatus: true,     // Show connection indicator
    onNewData: (data) => {          // Custom new data handler
      console.log("New data:", data);
    },
    onError: (error) => {           // Custom error handler
      console.error("Live mode error:", error);
    },
  }
}}
```

## Event Types

The SSE implementation supports these event types:

- `connected`: Initial connection established
- `new-entry`: New data item available
- `heartbeat`: Keep-alive signal with timestamp
- `error`: Error occurred (with error message)

## Features

### Automatic Reconnection
- Exponential backoff strategy
- Maximum retry attempts limit
- Connection state tracking

### User Experience
- Non-intrusive banner notification
- Manual refresh control
- Connection status indicator
- Dismiss without refreshing

### Performance
- Efficient polling strategy
- Query result caching
- Minimal bandwidth usage

## Testing

To test live mode:

1. Enable live mode on a table
2. Open the table in your browser
3. Create/modify data via API or another browser tab
4. Verify the banner appears with new item count
5. Click refresh to see updated data

## Troubleshooting

### Connection Issues
- Check browser console for SSE errors
- Verify endpoint is accessible
- Check for CORS configuration

### Performance Issues
- Reduce polling interval if needed
- Limit query result size
- Monitor server resource usage

### Data Sync Issues
- Verify timestamp handling in backend
- Check query filtering logic
- Ensure proper timezone handling

## Best Practices

1. **Enable Gradually**: Start with low-traffic tables
2. **Monitor Performance**: Track server and client resources
3. **Graceful Degradation**: Handle connection failures gracefully
4. **User Control**: Allow users to disable live mode
5. **Caching Strategy**: Don't over-invalidate query cache

## Future Enhancements

Possible improvements for the live mode system:

- WebSocket support for bi-directional communication
- Push notifications for critical updates
- Selective column updates instead of full refresh
- User preference storage for live mode settings
- Real-time collaboration indicators