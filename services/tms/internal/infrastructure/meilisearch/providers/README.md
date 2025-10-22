# Search Providers

This package provides search providers for converting domain entities into Meilisearch documents using the Fx dependency injection framework.

## Architecture

### The `Searchable` Interface

Domain entities can implement the `Searchable` interface to become searchable:

```go
type Searchable interface {
    GetID() string
    GetOrganizationID() pulid.ID
    GetBusinessUnitID() pulid.ID
    GetSearchTitle() string
    GetSearchSubtitle() string
    GetSearchContent() string
    GetSearchMetadata() map[string]any
    GetSearchEntityType() meilisearchtype.EntityType
    GetSearchTimestamps() (createdAt, updatedAt int64)
}
```

### Two Approaches

#### 1. Direct Interface Implementation (Recommended for Simple Cases)

If your entity implements the `Searchable` interface, use the `SearchHelper.Index()` method directly:

```go
// In your domain entity
func (s *Shipment) GetID() string {
    return s.ID.String()
}

func (s *Shipment) GetOrganizationID() pulid.ID {
    return s.OrganizationID
}

func (s *Shipment) GetBusinessUnitID() pulid.ID {
    return s.BusinessUnitID
}

func (s *Shipment) GetSearchTitle() string {
    return fmt.Sprintf("PRO: %s", s.ProNumber)
}

// ... implement other Searchable methods

// In your service
if s.searchHelper != nil {
    if err := s.searchHelper.Index(ctx, shipment); err != nil {
        // Error is logged, doesn't fail operation by default
    }
}
```

**Advantages:**

- No custom provider needed
- Clear contract via interface
- Works out of the box

**Disadvantages:**

- Adds methods to domain entities
- May not be ideal for complex conversion logic

#### 2. Custom Provider (Recommended for Complex Logic)

For complex conversion logic or when you need to compose data from multiple sources, create a custom provider:

```go
// shipment_provider.go
type ShipmentSearchProvider struct {
    BaseProvider
}

func NewShipmentSearchProvider() *ShipmentSearchProvider {
    return &ShipmentSearchProvider{}
}

func (p *ShipmentSearchProvider) EntityType() meilisearchtype.EntityType {
    return meilisearchtype.EntityTypeShipment
}

func (p *ShipmentSearchProvider) ToSearchDocument(
    shp *shipment.Shipment,
) (*meilisearchtype.SearchDocument, error) {
    // Custom conversion logic with complex formatting
    // ...
}
```

Then inject it via Fx and use it through the SearchHelper:

```go
// In your service (Fx will inject it)
type ServiceParams struct {
    fx.In
    SearchHelper *providers.SearchHelper `optional:"true"`
}

// Use the shipment-specific method
if err := s.searchHelper.IndexShipment(ctx, shipment); err != nil {
    // Handle error
}
```

**Advantages:**

- Separates complex conversion logic from domain
- Can inject dependencies if needed
- More flexible for special cases

**Disadvantages:**

- Requires additional code
- Must be registered in Fx module

## Fx Integration

The search infrastructure is provided via the `SearchModule`:

```go
// internal/bootstrap/modules/infrastructure/search.go
var SearchModule = fx.Module("search",
    fx.Provide(
        // Core infrastructure
        meilisearch.NewConnection,
        fx.Annotate(
            meilisearch.NewEngine,
            fx.As(new(ports.SearchEngine)),
        ),

        // Providers
        providers.NewShipmentSearchProvider,
        // Add more providers here as needed

        // Helper
        providers.NewSearchHelper,
    ),
)
```

### Using in Services

Inject the `SearchHelper` in your service:

```go
type ServiceParams struct {
    fx.In
    DB           *bun.DB
    Logger       *zap.Logger
    SearchHelper *providers.SearchHelper `optional:"true"` // Optional if search is disabled
}

type Service struct {
    db           *bun.DB
    logger       *zap.Logger
    searchHelper *providers.SearchHelper
}

func NewService(p ServiceParams) *Service {
    return &Service{
        db:           p.DB,
        logger:       p.Logger,
        searchHelper: p.SearchHelper,
    }
}
```

### Indexing Entities

```go
// After creating/updating an entity
func (s *Service) Create(ctx context.Context, data *CreateDTO) (*Entity, error) {
    entity := &Entity{/* ... */}
    
    if err := s.db.NewInsert().Model(entity).Scan(ctx); err != nil {
        return nil, err
    }
    
    // Index in search (fails silently by default)
    if s.searchHelper != nil {
        // Option 1: If entity implements Searchable
        if err := s.searchHelper.Index(ctx, entity); err != nil {
            // Error is already logged
        }
        
        // Option 2: If using custom provider (e.g., for shipments)
        if err := s.searchHelper.IndexShipment(ctx, shipment); err != nil {
            // Error is already logged
        }
    }
    
    return entity, nil
}

// After deleting
func (s *Service) Delete(ctx context.Context, id pulid.ID) error {
    // Delete from DB
    if _, err := s.db.NewDelete().Model((*Entity)(nil)).Where("id = ?", id).Exec(ctx); err != nil {
        return err
    }
    
    // Delete from search
    if s.searchHelper != nil {
        req := meilisearchtype.DeleteOperationRequest{
            EntityType: meilisearchtype.EntityTypeShipment,
            OrgID:      orgID.String(),
            BuID:       buID.String(),
            DocumentID: id.String(),
        }
        if err := s.searchHelper.engine.Delete(ctx, req); err != nil {
            s.logger.Warn("Failed to delete from search", zap.Error(err))
        }
    }
    
    return nil
}
```

### Batch Indexing

For bulk operations or reindexing:

```go
// Option 1: Batch index Searchable entities
entities := []providers.Searchable{entity1, entity2, entity3}
if err := s.searchHelper.BatchIndex(ctx, entities); err != nil {
    // Handle error
}

// Option 2: Batch index using custom provider
shipments := []*shipment.Shipment{shp1, shp2, shp3}
if err := s.searchHelper.IndexShipments(ctx, shipments); err != nil {
    // Handle error
}
```

## Configuration

By default, `SearchHelper` operates in "fail silently" mode - indexing errors are logged but don't fail the operation:

```go
// Make indexing errors propagate
helper.SetFailSilently(false)
```

This is useful during development or for critical indexing operations.

## Adding New Entity Types

1. **Option A: Implement Searchable Interface**
   - Add the interface methods to your domain entity
   - Use `SearchHelper.Index()` directly
   - No additional setup needed

2. **Option B: Create Custom Provider**
   - Create `{entity}_provider.go` implementing conversion logic
   - Add `New{Entity}SearchProvider` to Fx module in `search.go`
   - Optionally add convenience methods to `SearchHelper`

3. **In both cases:**
   - Add entity type to `meilisearchtype.EntityType` enum
   - Add index configuration to `index_config.go`

## Best Practices

1. **Keep search representations simple**: Title, subtitle, content should be human-readable
2. **Use metadata for structured data**: Store filterable fields in metadata
3. **Include searchable variations**: Add common abbreviations, codes, aliases
4. **Fail silently by default**: Search indexing shouldn't break core operations
5. **Use Fx for dependency injection**: Never create providers with `New*()` directly in services
6. **Test conversion logic**: Ensure search documents are valid and complete
7. **Optional injection**: Mark `SearchHelper` as `optional:"true"` to support disabling search

## Search Operations

### Basic Search

```go
req := &meilisearchtype.SearchRequest{
    Query:          "john",
    OrganizationID: orgID,
    BusinessUnitID: buID,
    Limit:          20,
    Offset:         0,
}

results, err := searchHelper.Search(req)
```

### Filtered Search

```go
req := &meilisearchtype.SearchRequest{
    Query:          "active",
    EntityTypes:    []meilisearchtype.EntityType{
        meilisearchtype.EntityTypeWorker,
        meilisearchtype.EntityTypeCustomer,
    },
    OrganizationID: orgID,
    BusinessUnitID: buID,
    Limit:          20,
}

results, err := searchHelper.Search(req)
```

### Entity-Specific Search

```go
req := &meilisearchtype.SearchRequest{
    Query:          "PRO12345",
    OrganizationID: orgID,
    BusinessUnitID: buID,
}

results, err := searchHelper.SearchByEntityType(req, meilisearchtype.EntityTypeShipment)
```
