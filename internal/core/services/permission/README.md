# Trenova Permission Service Documentation

---
Document Version: 1.0.0

Document Date: 2024-11-26

Revision: 1

Author: Eric M.

---

## Overview

The Permission Service provides a comprehensive Role-Based Access Control (RBAC) system with additional support for:

- Field-level permissions
- Time-based access control
- Resource ownership checks
- Organizational hierarchy scoping
- Permission dependencies
- Conditional access rules

## Permission Hierarchy

### Scopes

```go
const (
    ScopeGlobal   Scope = "global"        // Across entire system
    ScopeBU       Scope = "business_unit" // Limited to business unit
    ScopeOrg      Scope = "organization"  // Limited to organization
    ScopePersonal Scope = "personal"      // Limited to user's own resources
)
```

### Actions

```go
const (
    // Standard CRUD
    ActionCreate Action = "create"
    ActionRead   Action = "read"
    ActionUpdate Action = "update"
    ActionDelete Action = "delete"

    // Field-level
    ActionModifyField Action = "modify_field"
    ActionViewField   Action = "view_field"

    // Workflow
    ActionApprove  Action = "approve"
    ActionReject   Action = "reject"
    ActionSubmit   Action = "submit"
    ActionCancel   Action = "cancel"
    ActionAssign   Action = "assign"
    ActionComplete Action = "complete"
    
    // Administrative
    ActionManage    Action = "manage"
    ActionAudit     Action = "audit"
    ActionDelegate  Action = "delegate"
    ActionConfigure Action = "configure"
)
```

## Real-World Examples

### 1. Shipment Management

```go
// Example: Complete Shipment Flow
type ShipmentService struct {
    permService *permission.Service
    repo        repository.ShipmentRepository
}

func (s *ShipmentService) CompleteShipment(ctx context.Context, params CompleteShipmentParams) error {
    // 1. Check basic completion permission
    completeCheck := permission.Check{
        UserID:         params.RequesterID,
        Resource:       models.ResourceShipment,
        Action:         models.ActionComplete,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
        ResourceID:     params.ShipmentID,
    }
    
    // We need both complete and invoice create permissions
    checks := []permission.Check{
        completeCheck,
        {
            UserID:         params.RequesterID,
            Resource:       models.ResourceInvoice,
            Action:         models.ActionCreate,
            BusinessUnitID: params.BusinessUnitID,
            OrganizationID: params.OrganizationID,
        },
    }
    
    result, err := s.permService.HasAllPermissions(ctx, checks)
    if err != nil {
        return eris.Wrap(err, "check shipment completion permissions")
    }
    if !result.Allowed {
        return validation.NewAuthorizationError("insufficient permissions to complete shipment")
    }

    // 2. Check field-level permissions for sensitive updates
    sensitiveFields := []string{"actual_revenue", "driver_pay", "fuel_cost"}
    fieldCheck := permission.Check{
        UserID:         params.RequesterID,
        Resource:       models.ResourceShipment,
        Action:         models.ActionUpdate,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
        ResourceID:     params.ShipmentID,
    }

    fieldResult, err := s.permService.HasAllFieldPermissions(ctx, sensitiveFields, fieldCheck)
    if err != nil {
        return eris.Wrap(err, "check sensitive field permissions")
    }
    if !fieldResult.Allowed {
        return validation.NewAuthorizationError("insufficient permissions for financial fields")
    }

    // Proceed with completion...
}

// Example: Dispatch Assignment with Temporal Check
func (s *ShipmentService) AssignDispatcher(ctx context.Context, params AssignDispatcherParams) error {
    // Check if dispatcher can be assigned during their shift
    check := permission.Check{
        UserID:         params.DispatcherID,
        Resource:       models.ResourceShipment,
        Action:         models.ActionAssign,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
        CustomData: map[string]any{
            "assignmentTime": time.Now(),
            "shiftStart":    params.ShiftStart,
            "shiftEnd":      params.ShiftEnd,
            "dispatchZone":  params.DispatchZone,
        },
    }

    result, err := s.permService.HasTemporalPermission(ctx, check)
    if err != nil {
        return eris.Wrap(err, "check dispatcher assignment permission")
    }
    if !result.Allowed {
        return validation.NewAuthorizationError("dispatcher cannot be assigned outside shift hours")
    }

    // Proceed with assignment...
}
```

### 2. Worker Management

```go
// Example: Comprehensive Worker Update
type WorkerService struct {
    permService *permission.Service
    repo        repository.DriverRepository
}

func (s *WorkerService) UpdateWorker(ctx context.Context, params UpdateWorkerParams) (*models.Worker, error) {
    logger := s.l.With().
        Str("operation", "UpdateWorker").
        Interface("params", params).
        Logger()

    // 1. Basic update permission
    updateCheck := permission.Check{
        UserID:         params.RequesterID,
        Resource:       models.ResourceWorker,
        Action:         models.ActionUpdate,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
        ResourceID:     params.WorkerID,
    }

    result, err := s.permService.HasPermission(ctx, updateCheck)
    if err != nil {
        logger.Error().Err(err).Msg("failed to check update permission")
        return nil, eris.Wrap(err, "check worker update permission")
    }
    if !result.Allowed {
        return nil, validation.NewAuthorizationError("cannot update worker")
    }

    // 2. Check field-level permissions based on update type
    if params.UpdateType == UpdateTypePersonal {
        personalFields := []string{"address", "phone", "email", "emergency_contact"}
        fieldCheck := updateCheck
        fieldCheck.Action = models.ActionModifyField

        fieldResult, err := s.permService.HasAllFieldPermissions(ctx, personalFields, fieldCheck)
        if err != nil {
            return nil, eris.Wrap(err, "check personal fields permission")
        }
        if !fieldResult.Allowed {
            return nil, validation.NewAuthorizationError("insufficient permissions for personal fields")
        }
    }

    if params.UpdateType == UpdateTypeSensitive {
        sensitiveFields := []string{"ssn", "license_number", "medical_card", "drug_test"}
        fieldCheck := updateCheck
        fieldCheck.Action = models.ActionModifyField

        // For sensitive updates, also check if user has the required role
        check := permission.Check{
            UserID:         params.RequesterID,
            Resource:       models.ResourceWorker,
            Action:         models.ActionModifyField,
            BusinessUnitID: params.BusinessUnitID,
            OrganizationID: params.OrganizationID,
            CustomData: map[string]any{
                "requiredRole": "hr_manager",
            },
        }

        results, err := s.permService.HasAllPermissions(ctx, []permission.Check{
            check,
            {
                UserID:         params.RequesterID,
                Resource:       models.ResourceWorker,
                Action:         models.ActionAudit,
                BusinessUnitID: params.BusinessUnitID,
            },
        })
        if err != nil {
            return nil, eris.Wrap(err, "check sensitive update permissions")
        }
        if !results.Allowed {
            return nil, validation.NewAuthorizationError("insufficient permissions for sensitive updates")
        }

        fieldResult, err := s.permService.HasAllFieldPermissions(ctx, sensitiveFields, fieldCheck)
        if err != nil {
            return nil, eris.Wrap(err, "check sensitive fields permission")
        }
        if !fieldResult.Allowed {
            return nil, validation.NewAuthorizationError("insufficient permissions for sensitive fields")
        }
    }

    // Proceed with update...
}
```

### 3. Financial Operations

```go
// Example: Invoice Management
type InvoiceService struct {
    permService *permission.Service
    repo        repository.InvoiceRepository
}

func (s *InvoiceService) ApproveInvoice(ctx context.Context, params ApproveInvoiceParams) error {
    // 1. Check approval permission with amount threshold
    check := permission.Check{
        UserID:         params.RequesterID,
        Resource:       models.ResourceInvoice,
        Action:         models.ActionApprove,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
        ResourceID:     params.InvoiceID,
        CustomData: map[string]any{
            "amount": params.Amount,
            "threshold": 10000.00,
        },
    }

    // Also check if user can approve high-value invoices
    if params.Amount > 10000.00 {
        result, err := s.permService.HasScopedPermission(ctx, check, models.ScopeBU)
        if err != nil {
            return eris.Wrap(err, "check high-value invoice approval permission")
        }
        if !result.Allowed {
            return validation.NewAuthorizationError("insufficient permissions for high-value invoice")
        }
    }

    // 2. Check field-level permissions for financial data
    financialFields := []string{"payment_terms", "due_date", "discount_amount"}
    fieldCheck := check
    fieldCheck.Action = models.ActionModifyField

    fieldResult, err := s.permService.HasAllFieldPermissions(ctx, financialFields, fieldCheck)
    if err != nil {
        return eris.Wrap(err, "check financial fields permission")
    }
    if !fieldResult.Allowed {
        return validation.NewAuthorizationError("insufficient permissions for financial fields")
    }

    // Proceed with approval...
}
```

### 4. Report Generation

```go
// Example: Report Access Control
type ReportService struct {
    permService *permission.Service
    repo        repository.ReportRepository
}

func (s *ReportService) GenerateFinancialReport(ctx context.Context, params ReportParams) (*Report, error) {
    // 1. Check basic report permission
    baseCheck := permission.Check{
        UserID:         params.RequesterID,
        Resource:       models.ResourceReport,
        Action:         models.ActionCreate,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
    }

    // 2. Check access to financial data
    financialCheck := permission.Check{
        UserID:         params.RequesterID,
        Resource:       models.ResourceInvoice,
        Action:         models.ActionRead,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
        CustomData: map[string]any{
            "reportType": "financial",
            "dateRange":  params.DateRange,
        },
    }

    // Need both permissions
    result, err := s.permService.HasAllPermissions(ctx, []permission.Check{
        baseCheck,
        financialCheck,
    })
    if err != nil {
        return nil, eris.Wrap(err, "check report generation permissions")
    }
    if !result.Allowed {
        return nil, validation.NewAuthorizationError("insufficient permissions for financial report")
    }

    // 3. Check field-level access for sensitive financial data
    sensitiveFields := []string{
        "profit_margin",
        "operating_costs",
        "revenue_breakdown",
        "payroll_summary",
    }

    fieldCheck := permission.Check{
        UserID:         params.RequesterID,
        Resource:       models.ResourceReport,
        Action:         models.ActionViewField,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
    }

    fieldResult, err := s.permService.HasAllFieldPermissions(ctx, sensitiveFields, fieldCheck)
    if err != nil {
        return nil, eris.Wrap(err, "check sensitive field permissions")
    }
    if !fieldResult.Allowed {
        return nil, validation.NewAuthorizationError("insufficient permissions for sensitive financial data")
    }

    // Proceed with report generation...
}
```

## Best Practices

### 1. Permission Granularity

```go
// BAD: Too coarse
result, err := s.permService.HasPermission(ctx, permission.Check{
    UserID:   requesterID,
    Resource: models.ResourceDriver,
    Action:   models.ActionUpdate,
})

// GOOD: Proper scope and context
result, err := s.permService.HasPermission(ctx, permission.Check{
    UserID:         requesterID,
    Resource:       models.ResourceDriver,
    Action:         models.ActionUpdate,
    BusinessUnitID: params.BusinessUnitID,
    OrganizationID: params.OrganizationID,
    ResourceID:     driverID,
    CustomData: map[string]any{
        "updateType": "qualification",
        "documents":  documentTypes,
    },
})
```

### 2. Error Handling

```go
// Handle different types of permission errors
func handlePermissionError(err error) error {
    switch {
    case validation.IsAuthorizationError(err):
        // Handle authorization errors
        return fiber.NewError(fiber.StatusForbidden, err.Error())
    case eris.Is(err, permission.ErrFieldRequired):
        // Handle field validation errors
        return fiber.NewError(fiber.StatusBadRequest, "missing required field")
    default:
        // Handle unexpected errors
        return fiber.NewError(fiber.StatusInternalServerError, "internal server error")
    }
}
```

### 3. Cache Management

```go
// Invalidate cache after permission changes
func (s *Service) UpdateUserRole(ctx context.Context, params UpdateRoleParams) error {
    // Update role...

    // Invalidate permissions cache
    if err := s.permService.InvalidateUserPermissions(ctx, params.UserID); err != nil {
        s.l.Error().Err(err).Msg("failed to invalidate permissions cache")
        // Continue despite cache error
    }
}
```

## Common Pitfalls and Solutions

1. **Resource Ownership**

```go
// WRONG: Not checking resource ownership
result, err := s.permService.HasPermission(ctx, permission.Check{
    UserID:   requesterID,
    Resource: models.ResourceDriver,
    Action:   models.ActionUpdate,
})

// RIGHT: Including ownership context
result, err := s.permService.HasPermission(ctx, permission.Check{
    UserID:     requesterID,
    Resource:   models.ResourceDriver,
    Action:     models.ActionUpdate,
    ResourceID: driverID,
    CustomData: map[string]any{
        "ownerID": driverOwnerID,
    },
})
```

2. **Scope Validation**

```go
// RIGHT: Explicit scope check
result, err := s.permService.HasScopedPermission(ctx, 
    permission.Check{
        UserID:         requesterID,
        Resource:       models.ResourceReport,
        Action:         models.ActionExport,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
    }, 
    models.ScopeBU,
)

// BETTER: Check multiple scopes for hierarchical permissions
globalCheck := permission.Check{
    UserID:   requesterID,
    Resource: models.ResourceReport,
    Action:   models.ActionExport,
}

buCheck := permission.Check{
    UserID:         requesterID,
    Resource:       models.ResourceReport,
    Action:         models.ActionExport,
    BusinessUnitID: params.BusinessUnitID,
}

result, err := s.permService.HasAnyPermissions(ctx, []permission.Check{
    globalCheck,
    buCheck,
})
```

3. **Time-Based Access**

```go
// WRONG: Not considering temporal constraints
result, err := s.permService.HasPermission(ctx, permission.Check{
    UserID:   dispatcherID,
    Resource: models.ResourceDispatch,
    Action:   models.ActionAssign,
})

// RIGHT: Including temporal context
check := permission.Check{
    UserID:   dispatcherID,
    Resource: models.ResourceDispatch,
    Action:   models.ActionAssign,
    CustomData: map[string]any{
        "currentTime": time.Now().UTC(),
        "shiftStart": shiftStart,
        "shiftEnd":   shiftEnd,
        "timezone":   "America/New_York",
    },
}

result, err := s.permService.HasTemporalPermission(ctx, check)
```

## Advanced Usage Examples

### 1. Complex Workflow Permission

```go
// Example: Load tender workflow with multiple permission checks
func (s *ShipmentService) ProcessTender(ctx context.Context, params TenderParams) error {
    // 1. Base permission check
    baseCheck := permission.Check{
        UserID:         params.RequesterID,
        Resource:       models.ResourceShipment,
        Action:         models.ActionSubmit,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
        ResourceID:     params.ShipmentID,
    }

    // 2. Financial threshold checks
    financialCheck := baseCheck
    financialCheck.Action = models.ActionApprove
    financialCheck.CustomData = map[string]any{
        "tenderAmount": params.Amount,
        "threshold":    params.Threshold,
    }

    // 3. Time window check
    temporalCheck := baseCheck
    temporalCheck.CustomData = map[string]any{
        "tenderDeadline": params.Deadline,
        "currentTime":    time.Now().UTC(),
    }

    // Combine all checks
    result, err := s.permService.HasAllPermissions(ctx, []permission.Check{
        baseCheck,
        financialCheck,
        temporalCheck,
    })
    if err != nil {
        return eris.Wrap(err, "check tender permissions")
    }
    if !result.Allowed {
        return validation.NewAuthorizationError("insufficient permissions for tender processing")
    }

    // Check field-level permissions
    sensitiveFields := []string{
        "rate_per_mile",
        "fuel_surcharge",
        "accessorial_charges",
    }

    fieldCheck := permission.Check{
        UserID:         params.RequesterID,
        Resource:       models.ResourceShipment,
        Action:         models.ActionModifyField,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
        ResourceID:     params.ShipmentID,
    }

    fieldResult, err := s.permService.HasAllFieldPermissions(ctx, sensitiveFields, fieldCheck)
    if err != nil {
        return eris.Wrap(err, "check tender field permissions")
    }
    if !fieldResult.Allowed {
        return validation.NewAuthorizationError("insufficient permissions for rate fields")
    }

    // Proceed with tender processing...
    return nil
}
```

### 2. Multi-Resource Operations

```go
// Example: Transfer driver between business units
func (s *DriverService) TransferDriver(ctx context.Context, params TransferParams) error {
    // 1. Check source business unit permissions
    sourceChecks := []permission.Check{
        {
            UserID:         params.RequesterID,
            Resource:       models.ResourceDriver,
            Action:         models.ActionUpdate,
            BusinessUnitID: params.SourceBUID,
            ResourceID:     params.DriverID,
        },
        {
            UserID:         params.RequesterID,
            Resource:       models.ResourceBusinessUnit,
            Action:         models.ActionManage,
            BusinessUnitID: params.SourceBUID,
        },
    }

    // 2. Check destination business unit permissions
    destChecks := []permission.Check{
        {
            UserID:         params.RequesterID,
            Resource:       models.ResourceDriver,
            Action:         models.ActionCreate,
            BusinessUnitID: params.DestBUID,
        },
        {
            UserID:         params.RequesterID,
            Resource:       models.ResourceBusinessUnit,
            Action:         models.ActionManage,
            BusinessUnitID: params.DestBUID,
        },
    }

    // Combine all checks
    allChecks := append(sourceChecks, destChecks...)
    result, err := s.permService.HasAllPermissions(ctx, allChecks)
    if err != nil {
        return eris.Wrap(err, "check transfer permissions")
    }
    if !result.Allowed {
        return validation.NewAuthorizationError("insufficient permissions for driver transfer")
    }

    // Check field-level permissions for transfer-related fields
    transferFields := []string{
        "business_unit_id",
        "employment_status",
        "transfer_date",
        "payroll_info",
    }

    fieldCheck := permission.Check{
        UserID:         params.RequesterID,
        Resource:       models.ResourceDriver,
        Action:         models.ActionModifyField,
        BusinessUnitID: params.SourceBUID,
        ResourceID:     params.DriverID,
    }

    fieldResult, err := s.permService.HasAllFieldPermissions(ctx, transferFields, fieldCheck)
    if err != nil {
        return eris.Wrap(err, "check transfer field permissions")
    }
    if !fieldResult.Allowed {
        return validation.NewAuthorizationError("insufficient permissions for transfer fields")
    }

    // Proceed with transfer...
    return nil
}
```

### 3. Audit-Aware Operations

```go
// Example: Handling sensitive data with audit requirements
func (s *UserService) UpdateUserSecurity(ctx context.Context, params SecurityUpdateParams) error {
    // 1. Check basic permission
    baseCheck := permission.Check{
        UserID:         params.RequesterID,
        Resource:       models.ResourceUser,
        Action:         models.ActionUpdate,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
        ResourceID:     params.TargetUserID,
    }

    // 2. Check audit permission
    auditCheck := permission.Check{
        UserID:         params.RequesterID,
        Resource:       models.ResourceAuditLog,
        Action:         models.ActionCreate,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
    }

    // Security fields require both permissions
    result, err := s.permService.HasAllPermissions(ctx, []permission.Check{
        baseCheck,
        auditCheck,
    })
    if err != nil {
        return eris.Wrap(err, "check security update permissions")
    }
    if !result.Allowed {
        return validation.NewAuthorizationError("insufficient permissions for security update")
    }

    // Check field-level permissions with audit requirements
    securityFields := []string{
        "password_policy",
        "mfa_settings",
        "access_restrictions",
        "security_questions",
    }

    fieldCheck := permission.Check{
        UserID:         params.RequesterID,
        Resource:       models.ResourceUser,
        Action:         models.ActionModifyField,
        BusinessUnitID: params.BusinessUnitID,
        OrganizationID: params.OrganizationID,
        ResourceID:     params.TargetUserID,
        CustomData: map[string]any{
            "auditLevel": string(models.AuditFull),
            "reason":     params.Reason,
        },
    }

    fieldResult, err := s.permService.HasAllFieldPermissions(ctx, securityFields, fieldCheck)
    if err != nil {
        return eris.Wrap(err, "check security field permissions")
    }
    if !fieldResult.Allowed {
        return validation.NewAuthorizationError("insufficient permissions for security fields")
    }

    // Proceed with security update...
    return nil
}
```

## Testing Permissions

```go
func TestShipmentService_CompleteShipment(t *testing.T) {
    tests := []struct {
        name     string
        setup    func(*mocks.PermissionService)
        params   CompleteShipmentParams
        wantErr  bool
        errType  error
    }{
        {
            name: "successful completion with all permissions",
            setup: func(ps *mocks.PermissionService) {
                ps.EXPECT().
                    HasAllPermissions(mock.Anything, mock.Anything).
                    Return(permission.CheckResult{Allowed: true}, nil)
                ps.EXPECT().
                    HasAllFieldPermissions(mock.Anything, mock.Anything, mock.Anything).
                    Return(permission.CheckResult{Allowed: true}, nil)
            },
            params: CompleteShipmentParams{
                ShipmentID: testShipmentID,
                RequesterID: testUserID,
            },
            wantErr: false,
        },
        {
            name: "missing completion permission",
            setup: func(ps *mocks.PermissionService) {
                ps.EXPECT().
                    HasAllPermissions(mock.Anything, mock.Anything).
                    Return(permission.CheckResult{Allowed: false}, nil)
            },
            params: CompleteShipmentParams{
                ShipmentID: testShipmentID,
                RequesterID: testUserID,
            },
            wantErr: true,
            errType: &validation.AuthorizationError{},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Setup mocks and service
            // Run test
            // Assert results
        })
    }
}
```
