# Audit Service Developer Documentation

## Introduction

The Audit Service provides a robust, enterprise-grade auditing capability for the transportation management system. It records user actions with detailed context, enabling compliance with regulatory requirements (including FMCSA regulations), security monitoring, and operational insights.

## Core Features

- **High-throughput buffered logging**: Efficiently handles high volumes of audit events
- **Circuit breaker pattern**: Prevents system degradation during high load or outages
- **Sensitive data protection**: Configurable masking, hashing, and encryption of sensitive fields
- **Flexible metadata**: Rich contextual information for comprehensive audit trails
- **Critical event prioritization**: Special handling for high-priority audit events
- **Resilient error handling**: Automatic retries and failure management

## Setup and Configuration

### Configuration Options

The audit service uses the following configuration options:

```go
type AuditConfig struct {
    // Maximum number of entries in the buffer before flushing
    BufferSize int `yaml:"bufferSize" validate:"required,min=50"`
    
    // Interval in seconds to flush buffer even if not full
    FlushInterval int `yaml:"flushInterval" validate:"required,min=5"`
    
    // Enable or disable the audit service
    Enabled bool `yaml:"enabled" default:"true"`
}
```

Example configuration in YAML:

```yaml
audit:
  bufferSize: 1000
  flushInterval: 30
  enabled: true
```

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

### Registering Sensitive Fields

To protect sensitive data in your audit logs, register fields that need special handling:

```go
// During service initialization
auditService.RegisterSensitiveFields(permission.ResourceDriver, []services.SensitiveField{
    {Name: "licenseNumber", Action: audit.SensitiveFieldMask},
    {Name: "socialSecurityNumber", Action: audit.SensitiveFieldOmit},
    {Name: "medicalCardNumber", Action: audit.SensitiveFieldMask},
    {Name: "password", Action: audit.SensitiveFieldOmit},
})
```

Available actions for sensitive fields:

| Action | Description |
|--------|-------------|
| SensitiveFieldOmit | Completely removes the field from audit logs |
| SensitiveFieldMask | Replaces part of the value with asterisks (*) |
| SensitiveFieldHash | Replaces the value with a SHA-256 hash |
| SensitiveFieldEncrypt | Encrypts the value (requires encryption key setup) |

### Pattern-based Sensitive Data

You can also apply sensitive data handling based on field name patterns:

```go
auditService.RegisterSensitiveFields(permission.ResourceFinancial, []services.SensitiveField{
    {Name: "", Action: audit.SensitiveFieldMask, Pattern: "card.*number$"},
    {Name: "", Action: audit.SensitiveFieldMask, Pattern: "^account.*"},
})
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

## Troubleshooting

### Common Issues

1. **Circuit Breaker Open**
   - Symptom: Audit entries are being rejected
   - Solution: Check database connectivity, increase buffer size, or implement backoff strategy

2. **Sensitive Data Appearing in Logs**
   - Symptom: Unmasked sensitive data in audit logs
   - Solution: Register all sensitive fields and verify pattern matching

3. **Performance Degradation**
   - Symptom: Operations slowing down when audit service is active
   - Solution: Reduce the size of audited data, use compact diffs, or adjust buffer configuration

### Monitoring the Audit Service

The audit service exposes its state through the `GetServiceStatus()` method, which returns one of the following:

- `initializing`: Service is starting up
- `running`: Normal operation
- `degraded`: Service is experiencing issues but still functioning
- `stopping`: Service is shutting down
- `stopped`: Service is not running

## Conclusion

The audit service provides a comprehensive solution for maintaining detailed records of system activity to meet regulatory requirements, support security monitoring, and enable operational insights. By following the guidelines in this documentation, you can ensure effective use of the audit service throughout your application.

For additional questions or concerns, contact the platform team or refer to the detailed code documentation.
