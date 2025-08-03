<!--
Copyright 2023-2025 Eric Moss
Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md-->
# Audit Service

The audit service provides comprehensive audit logging capabilities for Trenova, including automatic sensitive data detection and masking.

## Key Features

- **Comprehensive Audit Logging**: Tracks all changes to resources with before/after states
- **Sensitive Data Protection**: Automatically detects and masks sensitive information
- **High Performance**: Queue-based architecture with batch processing
- **Flexible Configuration**: Environment-aware settings for different deployment scenarios
- **Array-Aware Path Handling**: Properly handles nested fields within arrays
- **Changes Field Sanitization**: Properly masks sensitive data in diff changes

## Architecture

### Components

1. **Service Layer** (`service.go`)
   - Main entry point for audit logging
   - Manages lifecycle and configuration
   - Handles queue management

2. **Sensitive Data Manager** (`sensitive.go`)
   - Auto-detects sensitive patterns (API keys, SSNs, credit cards, etc.)
   - Configurable masking strategies
   - Handles complex nested structures including arrays
   - Properly sanitizes the `Changes` field from diffs

3. **Queue System** (`queue.go`)
   - Buffered processing for high throughput
   - Configurable batch sizes and flush intervals
   - Graceful shutdown with timeout

4. **Batch Processor** (`processor.go`)
   - Efficiently writes audit entries to database
   - Handles retries and error recovery

## Sensitive Data Handling

### Auto-Detection Patterns

The service automatically detects and masks:
- API Keys (generic and provider-specific like Google)
- Social Security Numbers
- Credit Card Numbers
- Email Addresses
- Phone Numbers
- JWT Tokens
- Database Connection Strings
- IP Addresses
- Bank Account Numbers
- Driver's License Numbers
- Tax IDs
- Passport Numbers
- And more...

### Critical Improvements: Changes Field and User Object Handling

The service now properly sanitizes sensitive data in two critical areas:

#### 1. Changes Field Sanitization
The `Changes` field created by `WithDiff` is now properly sanitized. Previously, sensitive data like API keys were exposed in the diff output. Now:

- Both `from` and `to` values in change records are properly masked
- Nested objects within changes are handled correctly
- Auto-detection works on change paths (e.g., `configuration.apiKey`)

#### 2. User Object Sanitization
The `User` relationship object attached to audit entries is now sanitized to protect privacy:

- Email addresses are masked based on strategy (e.g., `admin@trenova.app` → `a****@trenova.app`)
- User IDs, Business Unit IDs, and Organization IDs are masked in strict/default modes
- Profile picture URLs are cleared in strict mode

Example of properly masked audit entry:
```json
{
  "changes": {
    "configuration.apiKey": {
      "from": "E****************************2", 
      "to": "A****************************U",
      "type": "updated"
    }
  },
  "user": {
    "id": "usr_************************",
    "emailAddress": "a****@trenova.app",
    "businessUnitId": "bu_************************",
    "currentOrganizationId": "org_************************"
  }
}
```

### Field Registration

You can explicitly register sensitive fields:

```go
auditService.RegisterSensitiveFields(permission.ResourceIntegration, []services.SensitiveField{
    // Simple field
    {Name: "apiKey", Action: services.SensitiveFieldMask},
    
    // Nested field
    {Path: "configuration", Name: "apiKey", Action: services.SensitiveFieldMask},
    
    // Array field notation
    {Path: "shipmentMoves[]", Name: "tractorId", Action: services.SensitiveFieldMask},
})
```

### Array Notation Support

The service supports several notations for fields within arrays:
- `shipmentMoves.tractorId` - Applies to all array elements
- `shipmentMoves[0].tractorId` - Specific array index
- `shipmentMoves[].tractorId` - Explicit array notation

### Enhanced Path Handling

The sensitive data manager now:
1. Builds proper paths for array elements (e.g., `shipmentMoves[0].tractorId`)
2. Checks multiple path variations when matching registered fields
3. Supports deeply nested structures with multiple array levels
4. Maintains context through recursive processing

## Configuration

### Environment-Based Settings

The service automatically configures based on environment:

- **Production**: Strict masking, auto-detection ON
- **Staging**: Default masking, auto-detection ON  
- **Development**: Partial masking, auto-detection ON
- **Testing**: Partial masking, auto-detection OFF

### Masking Strategies

1. **Strict**: Complete replacement with `****`
2. **Default**: Shows first and last character with dynamic masking (e.g., `E********************2`)
3. **Partial**: Shows more characters for debugging (e.g., `E40***************362`)

## Usage Examples

### Basic Audit Logging

```go
err := auditService.LogAction(&services.LogActionParams{
    Resource:       permission.ResourceIntegration,
    ResourceID:     integration.ID.String(),
    Action:        permission.ActionUpdate,
    UserID:        userID,
    OrganizationID: orgID,
    BusinessUnitID: buID,
    CurrentState:  currentIntegration,
    PreviousState: previousIntegration,
}, 
    audit.WithDiff(previousIntegration, currentIntegration),
    audit.WithComment("Updated API configuration"),
)
```

### Custom Sensitive Field Registration

```go
// Register fields for a custom resource
auditService.RegisterSensitiveFields(permission.ResourceShipment, []services.SensitiveField{
    // Simple field
    {Name: "trackingNumber", Action: services.SensitiveFieldMask},
    
    // Nested field
    {Path: "driver", Name: "licenseNumber", Action: services.SensitiveFieldMask},
    
    // Array field with explicit notation
    {Path: "stops[]", Name: "contactPhone", Action: services.SensitiveFieldMask},
    
    // Nested configuration
    {Path: "configuration", Name: "apiKey", Action: services.SensitiveFieldOmit},
})
```

## Performance Improvements

1. **Pre-compiled Patterns**: Field name patterns are compiled at initialization for better performance
2. **Efficient Path Checking**: Multiple path variations are checked in a single pass
3. **Batch Processing**: Entries are processed in configurable batches
4. **Concurrent Safe**: Uses `sync.Map` and atomic operations for thread safety
5. **Quick Pattern Matching**: Common API key patterns are checked first with a simple regex

## Troubleshooting

### API Keys Not Being Masked in Changes

This has been fixed. The service now:
1. Sanitizes the `Changes` field separately with special handling
2. Checks change paths against sensitive patterns
3. Masks both `from` and `to` values in change records

### Nested Fields in Arrays Not Masked

The service now properly handles:
1. Array-aware path building (e.g., `shipmentMoves[0].tractorId`)
2. Multiple path variation checking
3. Array notation in field registration (e.g., `shipmentMoves[].tractorId`)

### Performance Considerations

For optimal performance:
1. Register known sensitive fields explicitly rather than relying solely on auto-detection
2. Use appropriate batch sizes based on your audit volume
3. Consider adjusting the masking strategy based on your security requirements
4. Monitor queue statistics to ensure timely processing

## Security Best Practices

1. **Always verify masking**: Test your audit entries to ensure sensitive data is properly masked
2. **Use explicit registration**: Register all known sensitive fields for your resources
3. **Monitor auto-detection**: Review auto-detected fields periodically
4. **Consider environment**: Ensure production uses strict masking
5. **Review changes**: Pay special attention to the `Changes` field in audit entries