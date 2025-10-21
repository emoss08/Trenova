# Pagination Package

A comprehensive pagination solution for Gin-based REST APIs that supports filtering, sorting, and complex query parameters with a clean, fluent API.

## Features

- ✅ **Builder Pattern API** - Fluent, chainable methods for configuration
- ✅ **Generic Type Safety** - Full type safety with Go generics
- ✅ **Complex Query Support** - Handle domain-specific filters alongside pagination
- ✅ **Field Configuration** - Optional field validation for frontend table filtering
- ✅ **Automatic Query Binding** - Uses Gin's native form tag binding
- ✅ **Repository Integration** - Works seamlessly with QueryBuilder for database operations

## Table of Contents

- [Installation](#installation)
- [Basic Usage](#basic-usage)
- [Advanced Usage](#advanced-usage)
- [Repository Integration](#repository-integration)
- [API Reference](#api-reference)

## Installation

```go
import "github.com/emoss08/trenova/pkg/pagination"
```

## Basic Usage

### Simple Pagination

The simplest use case - paginating a list without any additional filters:

```go
func (h *OrganizationHandler) getUserOrganizations(c *gin.Context) {
    pagination.Handle[*tenant.Organization](c, context.GetAuthContext(c)).
        WithErrorHandler(h.eh).
        Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*tenant.Organization], error) {
            return h.service.GetUserOrganizations(c.Request.Context(), opts)
        })
}
```

This automatically handles:

- Parsing `limit` and `offset` from query parameters
- Setting tenant context (organization, business unit, user)
- Building next/previous page URLs
- Returning standardized JSON response

### Query Parameters

The handler automatically parses these query parameters:

```text
GET /api/organizations?limit=20&offset=0&query=search_term
```

## Advanced Usage

### Domain-Specific Query Parameters

For endpoints that need additional filtering beyond basic pagination:

```go
// Define your domain-specific filter options with form tags
type WorkerFilterOptions struct {
    Status      string `form:"status"`
    Type        string `form:"type"`
    Department  string `form:"department"`
    IsActive    bool   `form:"isActive"`
}

func (h *WorkerHandler) list(c *gin.Context) {
    var workerFilters WorkerFilterOptions
    
    pagination.Handle[*worker.Worker](c, context.GetAuthContext(c)).
        WithErrorHandler(h.eh).
        WithExtraParams(&workerFilters).  // Automatically binds query params
        Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*worker.Worker], error) {
            // Build request with both pagination and domain filters
            req := &repositories.ListWorkerRequest{
                Filter: opts,
                WorkerFilterOptions: workerFilters,
            }
            return h.service.List(c.Request.Context(), req)
        })
}
```

Query example:

```text
GET /api/workers?limit=20&offset=0&status=active&type=driver&department=logistics
```

### Complex Filtering with Date Ranges

```go
type PTOFilterOptions struct {
    Status      string `form:"status"`
    Type        string `form:"type"`
    StartDate   int64  `form:"startDate"`   // Unix timestamp
    EndDate     int64  `form:"endDate"`     // Unix timestamp
    WorkerID    string `form:"workerId"`
    FleetCodeID string `form:"fleetCodeId"`
}

func (h *WorkerHandler) listUpcomingPTO(c *gin.Context) {
    var ptoFilters PTOFilterOptions
    
    pagination.Handle[*worker.WorkerPTO](c, context.GetAuthContext(c)).
        WithErrorHandler(h.eh).
        WithExtraParams(&ptoFilters).
        Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*worker.WorkerPTO], error) {
            req := &repositories.ListUpcomingWorkerPTORequest{
                Filter: opts,
                PTOFilterOptions: ptoFilters,
            }
            return h.service.ListUpcomingPTO(c.Request.Context(), req)
        })
}
```

### Field Configuration for Frontend Tables

When you need to validate which fields can be filtered or sorted (useful for dynamic frontend tables):

```go
// Define field configuration for your domain
var WorkerFieldConfig = &pagination.FieldConfiguration{
    FilterableFields: map[string]bool{
        "status":    true,
        "firstName": true,
        "lastName":  true,
        "type":      true,
    },
    SortableFields: map[string]bool{
        "status":    true,
        "firstName": true,
        "lastName":  true,
        "createdAt": true,
    },
    FieldMap: map[string]string{
        "firstName": "first_name",  // Map API field to DB column
        "lastName":  "last_name",
        "createdAt": "created_at",
    },
}

func (h *WorkerHandler) list(c *gin.Context) {
    var workerFilters WorkerFilterOptions
    
    pagination.Handle[*worker.Worker](c, context.GetAuthContext(c)).
        WithErrorHandler(h.eh).
        WithFieldConfig(WorkerFieldConfig).  // Validates filter/sort fields
        WithExtraParams(&workerFilters).
        Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*worker.Worker], error) {
            return h.service.List(c.Request.Context(), opts)
        })
}
```

This accepts advanced filter parameters from the frontend:

```text
GET /api/workers?filters[0][field]=status&filters[0][operator]=eq&filters[0][value]=active&sort[0][field]=createdAt&sort[0][direction]=desc
```

## Repository Integration

### Using FilteredRequest Builder

The `request.go` provides a builder for creating structured requests:

```go
// In your handler
func (h *ShipmentHandler) list(c *gin.Context) {
    var shipmentFilters repositories.ShipmentFilterOptions
    
    pagination.Handle[*shipment.Shipment](c, context.GetAuthContext(c)).
        WithErrorHandler(h.eh).
        WithExtraParams(&shipmentFilters).
        Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*shipment.Shipment], error) {
            // Use the request builder
            req := pagination.BuildRequest(opts, shipmentFilters)
            return h.service.List(c.Request.Context(), req)
        })
}

// In your service
func (s *ShipmentService) List(
    ctx context.Context, 
    req *pagination.FilteredRequest[repositories.ShipmentFilterOptions],
) (*pagination.ListResult[*shipment.Shipment], error) {
    return s.repo.List(ctx, req)
}

// In your repository
func (r *ShipmentRepository) List(
    ctx context.Context,
    req *pagination.FilteredRequest[repositories.ShipmentFilterOptions],
) (*pagination.ListResult[*shipment.Shipment], error) {
    // Access pagination options
    filter := req.Filter
    
    // Access domain-specific options
    options := req.Options
    
    // Build query...
}
```

### Repository with QueryBuilder

Example of how pagination integrates with the QueryBuilder:

```go
func (r *workerRepository) filterQuery(
    q *bun.SelectQuery,
    req *repositories.ListWorkerRequest,
) *bun.SelectQuery {
    qb := querybuilder.NewWithPostgresSearch(
        q,
        "wrk",
        repositories.WorkerFieldConfig,
        (*worker.Worker)(nil),
    )

    // Apply tenant filters
    qb.ApplyTenantFilters(req.Filter.TenantOpts)

    // Apply field filters from frontend
    if req.Filter != nil {
        qb.ApplyFilters(req.Filter.FieldFilters)
        
        if len(req.Filter.Sort) > 0 {
            qb.ApplySort(req.Filter.Sort)
        }
        
        if req.Filter.Query != "" {
            qb.ApplyTextSearch(req.Filter.Query, []string{"first_name", "last_name"})
        }
    }
    
    q.Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
        return r.applyDomainFilters(sq, req.WorkerFilterOptions)
    })
    
    return q.Limit(req.Filter.Limit).Offset(req.Filter.Offset)
}

func (r *workerRepository) applyDomainFilters(
    q *bun.SelectQuery,
    opts repositories.WorkerFilterOptions,
) *bun.SelectQuery {
    if opts.Status != "" {
        q = q.Where("wrk.status = ?", opts.Status)
    }
    if opts.Type != "" {
        q = q.Where("wrk.type = ?", opts.Type)
    }
    if opts.IsActive {
        q = q.Where("wrk.is_active = ?", true)
    }
    return q
}
```

## Real-World Scenarios

### 1. Search with Autocomplete

```go
type SearchOptions struct {
    Query       string   `form:"q"`
    Types       []string `form:"types[]"`
    MaxResults  int      `form:"maxResults"`
}

func (h *SearchHandler) search(c *gin.Context) {
    var searchOpts SearchOptions
    
    pagination.Handle[*search.Result](c, context.GetAuthContext(c)).
        WithErrorHandler(h.eh).
        WithExtraParams(&searchOpts).
        Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*search.Result], error) {
            // Override limit if maxResults is specified
            if searchOpts.MaxResults > 0 {
                opts.Limit = searchOpts.MaxResults
            }
            
            return h.service.Search(c.Request.Context(), opts, searchOpts)
        })
}
```

### 2. Report Generation with Filters

```go
type ReportFilterOptions struct {
    StartDate    int64    `form:"startDate"`
    EndDate      int64    `form:"endDate"`
    GroupBy      string   `form:"groupBy"`
    Metrics      []string `form:"metrics[]"`
    IncludeEmpty bool     `form:"includeEmpty"`
}

func (h *ReportHandler) generateReport(c *gin.Context) {
    var reportFilters ReportFilterOptions
    
    pagination.Handle[*report.Entry](c, context.GetAuthContext(c)).
        WithErrorHandler(h.eh).
        WithExtraParams(&reportFilters).
        WithFieldConfig(report.FieldConfig).
        Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*report.Entry], error) {
            req := &repositories.GenerateReportRequest{
                Filter:        opts,
                ReportOptions: reportFilters,
            }
            return h.service.GenerateReport(c.Request.Context(), req)
        })
}
```

### 3. Nested Resource Filtering

```go
// List comments for a specific post with filtering
func (h *CommentHandler) listByPost(c *gin.Context) {
    postID := c.Param("postId")
    
    var commentFilters struct {
        AuthorID  string `form:"authorId"`
        Since     int64  `form:"since"`
        ParentID  string `form:"parentId"`  // For threaded comments
    }
    
    pagination.Handle[*comment.Comment](c, context.GetAuthContext(c)).
        WithErrorHandler(h.eh).
        WithExtraParams(&commentFilters).
        Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*comment.Comment], error) {
            return h.service.ListByPost(c.Request.Context(), postID, opts, commentFilters)
        })
}
```

## API Reference

### PaginationHandler Methods

| Method | Description |
|--------|-------------|
| `Handle[T](c *gin.Context, authCtx *AuthContext)` | Creates a new pagination handler |
| `WithErrorHandler(eh *ErrorHandler)` | Sets error handler for automatic error handling |
| `WithFieldConfig(config *FieldConfiguration)` | Sets field configuration for validation |
| `WithExtraParams(params any)` | Binds additional query parameters |
| `Execute(handler func)` | Executes the handler and returns paginated response |

### QueryOptions Structure

```go
type QueryOptions struct {
    TenantOpts   TenantOptions // Organization context
    Query        string        // Search query
    FieldFilters []FieldFilter // Advanced filters
    Sort         []SortField   // Sort options
    Limit        int          // Page size (default: 20, max: 100)
    Offset       int          // Skip count (default: 0)
}
```

### Response Format

```json
{
    "count": 150,
    "results": [...],
    "next": "https://api.example.com/resources?limit=20&offset=20",
    "previous": "https://api.example.com/resources?limit=20&offset=0"
}
```

## Best Practices

1. **Always use error handlers** - Ensures consistent error responses
2. **Define field configurations for public APIs** - Prevents unauthorized field access
3. **Use FilteredRequest for complex scenarios** - Maintains clean separation of concerns
4. **Validate domain-specific parameters** - Add validation tags to your filter structs
5. **Keep handlers thin** - Move business logic to service layer
6. **Use proper HTTP status codes** - The handler returns 200 by default, customize in error handler

## Migration from Old Pattern

### Before (Old Pattern)

```go
func (h *Handler) list(c *gin.Context) {
    authCtx := context.GetAuthContext(c)
    
    handler := func(c *gin.Context, filter *pagination.QueryOptions) (*pagination.ListResult[*Entity], error) {
        if err := c.ShouldBindQuery(filter); err != nil {
            h.eh.HandleError(c, err)
            return nil, err
        }
        return h.service.List(c.Request.Context(), filter)
    }
    
    pagination.HandlePaginatedRequest(c, h.eh, authCtx, handler)
}
```

### After (New Pattern)

```go
func (h *Handler) list(c *gin.Context) {
    pagination.Handle[*Entity](c, context.GetAuthContext(c)).
        WithErrorHandler(h.eh).
        Execute(func(c *gin.Context, opts *pagination.QueryOptions) (*pagination.ListResult[*Entity], error) {
            return h.service.List(c.Request.Context(), opts)
        })
}
```

## Contributing

When adding new features to the pagination package:

1. Maintain backward compatibility
2. Add comprehensive tests
3. Update this documentation
4. Follow the existing patterns and conventions
