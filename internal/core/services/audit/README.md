# Audit Service Developer Documentation

## Introduction

The Audit Service provides a robust, enterprise-grade auditing capability for the transportation management system. It records user actions with detailed context, enabling compliance with regulatory requirements (including FMCSA regulations), security monitoring, and operational insights.

## Core Features

- **Queue-based asynchronous processing**: Non-blocking audit logging with configurable workers
- **Automatic sensitive data detection**: AI-powered detection of SSNs, credit cards, API keys, etc.
- **Environment-aware configuration**: Automatic adjustment of security levels based on deployment
- **Advanced masking strategies**: Configurable data masking for different security requirements
- **Thread-safe design**: Prevents memory corruption and race conditions
- **Resilient error handling**: Automatic retries with exponential backoff
- **Performance metrics**: Built-in monitoring and observability

## Setup and Configuration

### Configuration Options

The audit service uses the following configuration options:

```go
type AuditConfig struct {
    // Maximum number of entries in the buffer before flushing
    BufferSize int `yaml:"bufferSize" validate:"required,min=50"`
    
    // Interval in seconds to flush buffer even if not full
    FlushInterval int `yaml:"flushInterval" validate:"required,min=5"`
    
    // Number of entries to process in a single batch
    BatchSize int `yaml:"batchSize" default:"50"`
    
    // Number of worker goroutines for processing
    Workers int `yaml:"workers" default:"2"`
    
    // Enable compression for large audit entries
    CompressionEnabled bool `yaml:"compressionEnabled" default:"false"`
    
    // Compression level (1-9)
    CompressionLevel int `yaml:"compressionLevel" default:"6"`
    
    // Size in KB before compression is applied
    CompressionThreshold int `yaml:"compressionThreshold" default:"10"`
}
```

Example configuration in YAML:

```yaml
audit:
  bufferSize: 1000
  flushInterval: 30
  batchSize: 50
  workers: 2
  compressionEnabled: true
  compressionLevel: 6
  compressionThreshold: 10
```

### Environment-Based Configuration

The audit service automatically configures sensitive data handling based on your environment:

| Environment | Auto-Detection | Masking Strategy | Description |
|-------------|----------------|------------------|-------------|
| Production | ON | Strict | Maximum security, minimal information shown |
| Staging | ON | Default | Balanced between security and debugging |
| Development | ON | Partial | More information shown for easier debugging |
| Testing | OFF | Partial | Predictable output for testing |

### Service Registration

The audit service is automatically registered with the dependency injection container (fx) during application startup. No manual initialization is required.

## Basic Usage

The primary method for recording audit events is `LogAction`. Here's a basic example:

```go
err := auditService.LogAction(
    &services.LogActionParams{
        Resource:       permission.ResourceDriver,
        ResourceID:     driver.GetID(),
        Action:         permission.ActionCreate,
        UserID:         userID,
        CurrentState:   jsonutils.MustToJSON(driver),
        OrganizationID: driver.OrganizationID,
        BusinessUnitID: driver.BusinessUnitID,
    },
    audit.WithComment("Driver created"),
)
```

### Required Parameters

All `LogAction` calls must include these parameters:

| Parameter | Description | Required |
|-----------|-------------|----------|
| Resource | The type of resource being audited (e.g., Driver, Shipment) | Yes |
| ResourceID | Unique identifier of the resource | Yes |
| Action | The action being performed (e.g., Create, Update) | Yes |
| UserID | ID of the user performing the action | Yes |
| OrganizationID | ID of the organization | Yes |
| BusinessUnitID | ID of the business unit | Yes |
| CurrentState | Current state of the resource as a map | No |
| PreviousState | Previous state of the resource as a map | No |
| Critical | Whether this is a critical audit entry | No |

## Audit Options

The service supports a variety of options to customize audit entries:

### Common Options

```go
// Add a comment to the audit entry
audit.WithComment("Driver license updated")

// Generate a detailed difference between before and after states
audit.WithDiff(originalDriver, updatedDriver)

// Generate a simplified difference format for large objects
audit.WithCompactDiff(originalShipment, updatedShipment)

// Add custom metadata
audit.WithMetadata(map[string]any{
    "requestSource": "mobile-app",
    "ipAddress": "192.168.1.1",
})

// Record user agent information
audit.WithUserAgent(userAgent)
```

### Advanced Options

```go
// Generate a correlation ID automatically
audit.WithCorrelationID()

// Use a specific correlation ID (e.g., to link multiple related actions)
audit.WithCustomCorrelationID("corr_123456789")

// Categorize the audit entry
audit.WithCategory("compliance")

// Mark an audit entry as critical (ensures it's logged even during backpressure)
audit.WithCritical()

// Record the client IP address
audit.WithIP("192.168.1.100")

// Use a specific timestamp instead of current time
audit.WithTimestamp(specificTime)

// Add geographic location information
audit.WithLocation("Atlanta, GA")

// Record the user's session ID
audit.WithSessionID("sess_abcdefg")

// Add searchable tags to the audit entry
audit.WithTags("safety-critical", "eld-compliance", "hours-of-service")
```

## Handling Sensitive Data

### Automatic Detection

The audit service automatically detects and masks common sensitive patterns:

- **Social Security Numbers**: `123-45-6789` → `XXX-XX-6789` (or based on strategy)
- **Credit Cards**: `4111111111111111` → `************1111`
- **API Keys**: Long alphanumeric strings are masked
- **JWT Tokens**: Three-part tokens are automatically detected
- **Private Keys**: PEM-formatted keys are completely redacted
- **Common Fields**: `password`, `secret`, `token`, `apiKey`, etc.

### Registering Custom Sensitive Fields

To protect additional sensitive data in your audit logs:

```go
// During service initialization
auditService.RegisterSensitiveFields(permission.ResourceDriver, []services.SensitiveField{
    {Name: "licenseNumber", Action: services.SensitiveFieldMask},
    {Name: "socialSecurityNumber", Action: services.SensitiveFieldOmit},
    {Name: "medicalCardNumber", Action: services.SensitiveFieldMask},
    {Name: "password", Action: services.SensitiveFieldOmit},
})
```

Available actions for sensitive fields:

| Action | Description | Example |
|--------|-------------|---------|
| SensitiveFieldOmit | Completely removes the field | `password: "secret"` → field removed |
| SensitiveFieldMask | Masks based on current strategy | `ssn: "123-45-6789"` → `ssn: "XXX-XX-6789"` |
| SensitiveFieldHash | SHA-256 hash (first 16 chars) | `token: "abc123"` → `token: "sha256:ba7816bf8f01cfea"` |
| SensitiveFieldEncrypt | AES-GCM encryption | `data: "sensitive"` → `data: "enc:gcm:base64data..."` |

### Pattern-based Sensitive Data

Apply sensitive data handling based on field name patterns:

```go
auditService.RegisterSensitiveFields(permission.ResourceFinancial, []services.SensitiveField{
    {Name: "", Action: services.SensitiveFieldMask, Pattern: "card.*number$"},
    {Name: "", Action: services.SensitiveFieldMask, Pattern: "^account.*"},
})
```

### Masking Strategies

Different masking strategies are automatically applied based on environment:

#### Strict Strategy (Production)
- Email: `test@example.com` → `****@example.com`
- String: `mysecretvalue` → `*************`
- SSN: `123-45-6789` → `XXX-XX-XXXX`

#### Default Strategy (Staging)
- Email: `test@example.com` → `t***@example.com`
- String: `mysecretvalue` → `m***********e`
- SSN: `123-45-6789` → `***-**-6789`

#### Partial Strategy (Development)
- Email: `test@example.com` → `te***@example.com`
- String: `mysecretvalue` → `my*********ue`
- SSN: `123-45-6789` → `XXX-XX-6789`

### Runtime Configuration

You can adjust sensitive data handling at runtime:

```go
// Temporarily enable more verbose masking for debugging
auditService.SetSensitiveDataMaskStrategy(MaskStrategyPartial)
defer auditService.SetSensitiveDataMaskStrategy(MaskStrategyStrict)

// Disable auto-detection during data migration
auditService.SetSensitiveDataAutoDetect(false)
// ... perform migration ...
auditService.SetSensitiveDataAutoDetect(true)

// Clear regex cache if patterns are updated
auditService.ClearSensitiveDataCache()
```

## Real-World Examples

### Example 1: Auditing Shipment Creation

```go
// When creating a new shipment
func (s *ShipmentService) Create(ctx context.Context, shipment *shipment.Shipment, userID pulid.ID) (*shipment.Shipment, error) {
    // ... validation and creation logic ...
    
    // Record the audit with transportation-specific metadata
    err := s.auditService.LogAction(
        &services.LogActionParams{
            Resource:       permission.ResourceShipment,
            ResourceID:     createdShipment.GetID(),
            Action:         permission.ActionCreate,
            UserID:         userID,
            CurrentState:   jsonutils.MustToJSON(createdShipment),
            OrganizationID: createdShipment.OrganizationID,
            BusinessUnitID: createdShipment.BusinessUnitID,
            Critical:       true, // Shipment creation is critical for compliance
        },
        audit.WithComment("Shipment created"),
        audit.WithCategory("operations"),
        audit.WithMetadata(map[string]any{
            "proNumber": createdShipment.ProNumber,
            "customerID": createdShipment.CustomerID.String(),
            "origin": getOriginInfo(createdShipment),
            "destination": getDestinationInfo(createdShipment),
        }),
        audit.WithTags("shipment-creation", "customer-"+createdShipment.CustomerID.String()),
    )
    
    // ... rest of the function ...
}
```

### Example 2: Auditing Driver HOS Updates (Hours of Service)

```go
func (s *HOSService) UpdateDriverLogs(ctx context.Context, logs *hos.DriverLogs, userID pulid.ID) (*hos.DriverLogs, error) {
    // ... validation and update logic ...
    
    // Retrieve the original logs for comparison
    originalLogs, _ := s.repo.GetByID(ctx, logs.ID)
    
    // ... update the logs ...
    
    // Audit the update with FMCSA compliance metadata
    err := s.auditService.LogAction(
        &services.LogActionParams{
            Resource:       permission.ResourceDriverLogs,
            ResourceID:     logs.ID.String(),
            Action:         permission.ActionUpdate,
            UserID:         userID,
            CurrentState:   jsonutils.MustToJSON(logs),
            PreviousState:  jsonutils.MustToJSON(originalLogs),
            OrganizationID: logs.OrganizationID,
            BusinessUnitID: logs.BusinessUnitID,
            Critical:       true, // HOS logs are critical for FMCSA compliance
        },
        audit.WithComment("Driver logs updated"),
        audit.WithCategory("compliance"),
        audit.WithDiff(originalLogs, logs),
        audit.WithTags("hos", "fmcsa-compliance", "driver-"+logs.DriverID.String()),
        audit.WithMetadata(map[string]any{
            "dutyStatusChange": getDutyStatusChange(originalLogs, logs),
            "totalDrivingTime": logs.TotalDrivingMinutes,
            "totalDutyTime": logs.TotalDutyMinutes,
            "eldSerialNumber": logs.ELDSerialNumber,
            "eldSequence": logs.LogSequence,
        }),
    )
    
    // ... rest of the function ...
}
```

### Example 3: Auditing Billing Operations

```go
func (s *BillingService) ProcessInvoice(ctx context.Context, invoice *billing.Invoice, userID pulid.ID) (*billing.Invoice, error) {
    // ... processing logic ...
    
    // Audit with financial context
    err := s.auditService.LogAction(
        &services.LogActionParams{
            Resource:       permission.ResourceInvoice,
            ResourceID:     invoice.ID.String(),
            Action:         permission.ActionCreate,
            UserID:         userID,
            CurrentState:   jsonutils.MustToJSON(invoice),
            OrganizationID: invoice.OrganizationID,
            BusinessUnitID: invoice.BusinessUnitID,
        },
        audit.WithComment(fmt.Sprintf("Invoice %s processed for $%.2f", invoice.InvoiceNumber, invoice.TotalAmount)),
        audit.WithCategory("financial"),
        audit.WithMetadata(map[string]any{
            "invoiceNumber": invoice.InvoiceNumber,
            "customerName": invoice.CustomerName,
            "totalAmount": invoice.TotalAmount,
            "dueDate": invoice.DueDate,
            "paymentTerms": invoice.PaymentTerms,
            "relatedShipments": getRelatedShipmentIDs(invoice),
        }),
        audit.WithCorrelationID(), // Generate a correlation ID for cross-referencing
    )
    
    // ... rest of the function ...
}
```

## Best Practices

### When to Use Critical Flag

Mark audit entries as critical when they involve:

- Financial transactions
- Safety-related changes
- FMCSA/DOT compliance-related actions
- Security-sensitive operations
- System configuration changes

Critical entries will be prioritized even during system degradation.

### Optimizing Performance

1. **Be selective with audit details**:
   - For large objects, use `WithCompactDiff` instead of `WithDiff`
   - Only include relevant fields in metadata

2. **Use categories effectively**:
   - Organize audit entries into logical categories
   - Common categories: "security", "compliance", "operations", "financial"

3. **Correlation IDs**:
   - Use correlation IDs to link related operations
   - For multi-step processes, pass the same correlation ID to all related audit entries

### Security Best Practices

1. **Always register sensitive fields** for all resources containing personal or financial information
2. **Double-check your JSON serialization** to ensure sensitive data isn't accidentally included
3. **Prefer field omission** over masking for highly sensitive data
4. **Apply the principle of least information** - only log what's necessary for auditing purposes

## Error Handling

The audit service is designed to fail gracefully and not block your application's main functionality.

```go
// Audit errors should be logged but shouldn't cause operations to fail
err := s.auditService.LogAction(...)
if err != nil {
    log.Error().Err(err).Msg("failed to log audit entry, continuing operation")
    // DO NOT return the error and fail the main operation
}
```

## Performance and Monitoring

### Queue Statistics

Monitor the audit service queue health:

```go
stats := auditService.GetQueueStats()
log.Info().
    Int("queued", stats.QueuedEntries).
    Int("capacity", stats.QueueCapacity).
    Msg("audit queue statistics")
```

### Sensitive Data Metrics

Track sensitive data sanitization performance:

```go
metrics := auditService.GetSensitiveDataMetrics()
log.Info().
    Int64("sanitized", metrics.SanitizedFields).
    Int64("encrypted", metrics.EncryptedFields).
    Int64("hashed", metrics.HashedFields).
    Int64("masked", metrics.MaskedFields).
    Int64("errors", metrics.Errors).
    Msg("sensitive data processing metrics")
```

## Troubleshooting

### Common Issues

1. **Queue Full**
   - Symptom: `ErrQueueFull` errors
   - Solution: Increase buffer size, add more workers, or reduce audit volume

2. **Sensitive Data Appearing in Logs**
   - Symptom: Unmasked sensitive data in audit logs
   - Solution: Ensure auto-detection is enabled, register custom fields, verify environment

3. **Performance Degradation**
   - Symptom: Operations slowing down when audit service is active
   - Solution: Adjust batch size, increase workers, enable compression for large entries

4. **Memory Corruption (Old Service)**
   - Symptom: Corrupted user IDs or field values
   - Solution: Update to ServiceV2 which uses queue-based processing without object pooling

### Monitoring the Audit Service

The audit service exposes its state through the `GetServiceStatus()` method, which returns one of the following:

- `initializing`: Service is starting up
- `running`: Normal operation
- `degraded`: Service is experiencing issues but still functioning
- `stopping`: Service is shutting down
- `stopped`: Service is not running

### Debug Mode

For troubleshooting sensitive data issues:

```go
// Enable partial masking to see more data
auditService.SetSensitiveDataMaskStrategy(MaskStrategyPartial)

// Check what fields are registered
fields := auditService.GetRegisteredSensitiveFields(permission.ResourceUser)
for name, field := range fields {
    log.Debug().
        Str("field", name).
        Str("action", field.Action.String()).
        Msg("registered sensitive field")
}
```

## Migration from V1 to V2

If you're upgrading from the original audit service to V2:

### Key Differences

1. **No Object Pooling**: V2 doesn't use `sync.Pool`, eliminating memory corruption issues
2. **Queue-Based**: Non-blocking with configurable worker threads
3. **Auto-Detection**: Automatic sensitive data detection is enabled by default
4. **Environment Aware**: Security settings adjust automatically based on deployment

### Migration Steps

1. Update your configuration to include new fields:
   ```yaml
   audit:
     bufferSize: 1000
     flushInterval: 30
     batchSize: 50      # New
     workers: 2         # New
   ```

2. The API remains the same - no code changes needed for `LogAction` calls

3. New runtime configuration options are available but optional

### Performance Improvements

- **Async Processing**: Audit logging no longer blocks main operations
- **Batch Processing**: Multiple entries processed together for efficiency
- **Worker Scaling**: Add more workers for higher throughput
- **Memory Safety**: No risk of corrupted data from object reuse

## Conclusion

The audit service provides a comprehensive solution for maintaining detailed records of system activity to meet regulatory requirements, support security monitoring, and enable operational insights. Version 2 introduces significant improvements in performance, security, and reliability while maintaining full backward compatibility.

For additional questions or concerns, contact the platform team or refer to the detailed code documentation.
