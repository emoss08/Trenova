# Workflow Versioning Implementation Status

## ✅ Completed

### 1. Design Documentation

**File**: `docs/workflow-versioning-design.md`

Comprehensive design covering:

- Template-version two-table pattern
- Version lifecycle management
- Rollback strategies
- JSON import/export structure
- UI considerations

### 2. Database Schema

**Files**:

- `migrations/20251118000001_workflow_system_v2.tx.up.sql`
- `migrations/20251118000001_workflow_system_v2.tx.down.sql`

**New Enums**:

- `workflow_version_status_enum`: Draft, Published, Archived

**New Tables**:

- `workflow_templates`: Stable workflow concept (name, description, published_version_id)
- `workflow_versions`: Version-specific configuration (version_number, trigger_type, status, version_status, schedule_config, trigger_config, change_description)

**Updated Tables**:

- `workflow_nodes`: Now references `workflow_version_id` instead of `workflow_id`
- `workflow_connections`: Now references `workflow_version_id`
- `workflow_instances`: Now references BOTH `workflow_template_id` AND `workflow_version_id`

**Key Features**:

- Composite PKs with org/bu isolation
- Unique constraint on version numbers per template
- Published version pointer in template table
- Full audit trail with created_by fields
- Search vector support on templates

### 3. Domain Models

**Package**: `internal/core/domain/workflow/`

**New Files**:

- `template.go`: Template entity (stable workflow concept)
- `version.go`: Version entity (version-specific configuration)

**Updated Files**:

- `enums.go`: Added `VersionStatus` enum with helper methods
- `node.go`: Renamed `WorkflowNode` → `Node`, now references `WorkflowVersionID`
- `connection.go`: Now references `WorkflowVersionID`
- `instance.go`: Now references both `WorkflowTemplateID` and `WorkflowVersionID`
- `nodeexecution.go`: Updated to reference `Node` instead of `WorkflowNode`

**Type Changes**:

- `Workflow` → `Template` (represents the stable concept)
- `WorkflowNode` → `Node` (cleaner naming)
- All entities follow existing patterns (Bun ORM, validation, multi-tenancy)

### 4. Key Relationships

```
Template (1) ----< (many) Version
   ↓
published_version_id → Version (currently active)

Version (1) ----< (many) Node
Version (1) ----< (many) Connection

Instance belongs_to Template (which workflow)
Instance belongs_to Version (which specific version was executed)
```

### 5. Repository Interfaces

**File**: `internal/core/ports/repositories/workflow.go`

**Completed**:

- Split `WorkflowRepository` → `TemplateRepository` + `VersionRepository`
- Added version-specific methods:
  - `Create(req *CreateVersionRequest)` - Create new version (empty or cloned)
  - `Publish(req *PublishVersionRequest)` - Publish draft version
  - `Archive(req *ArchiveVersionRequest)` - Archive version
  - `Rollback(req *RollbackVersionRequest)` - Rollback to previous version
  - `GetPublished(req *GetPublishedVersionRequest)` - Get published version
  - `GetNodes/GetConnections` - Get version nodes and connections
  - `CreateNode/UpdateNode/DeleteNode` - Manage nodes within version
  - `CreateConnection/UpdateConnection/DeleteConnection` - Manage connections
- Updated `WorkflowInstanceRepository` to reference both template and version
- Updated `StartWorkflowExecutionRequest` to support version selection (defaults to published)

### 6. Validators

**Files**: `pkg/validator/workflowvalidator/*`

**Completed**:

- **template.go**: Renamed from workflow.go, validates `*workflow.Template`
  - Template name uniqueness
  - ID validation on create
- **version.go**: NEW validator for `*workflow.Version`
  - Template existence validation
  - Version number uniqueness per template
  - Single published version constraint
  - Draft-only edit validation (Published/Archived versions immutable)
  - Trigger configuration validation (moved from template)
- **node.go**: Updated for versioning
  - Changed `*workflow.WorkflowNode` → `*workflow.Node`
  - Changed `WorkflowID` → `WorkflowVersionID`
  - Validates version existence instead of workflow
- **connection.go**: Updated for versioning
  - Changed `WorkflowNode` → `Node`
  - Changed `WorkflowID` → `WorkflowVersionID`
  - Validates nodes belong to same version
  - Circular dependency detection updated for versions

### 7. Repository Layer

**Directory**: `internal/infrastructure/postgres/repositories/workflowrepository/`

**Completed**:

- **template.go**: Template repository with CRUD, duplicate, import/export
  - List templates with optional version inclusion
  - Create/Update/Delete templates with audit fields (CreatedByID, UpdatedByID)
  - Duplicate template with published version cloning
  - Export to JSON (single version or all versions)
  - Import from JSON with version reconstruction
- **version.go**: Version repository with lifecycle management
  - CRUD operations for versions
  - Create version (empty or cloned from existing)
  - Publish: Archives current published, publishes draft, updates template pointer
  - Archive: Marks version as archived
  - Rollback: Re-publishes archived version, archives current
  - GetPublished: Retrieves currently published version
  - Node and Connection management within versions
- **instance.go**: Instance repository
  - List/Get instances with template and version relations
  - Create/Update instances
  - Get node executions for instance
- **nodeexecution.go**: Node execution repository
  - Create/Update node executions
  - Get executions by instance

## 📋 Next Steps

### 1. Service Layer

**Directory**: `internal/core/services/workflowsvc/`

Implement business logic:

- Template management
- Version creation from template
- Version publishing/rollback workflow
- Import/export with version handling
- Execution initiation (which version to run)

### 3. HTTP Handlers

**Directory**: `internal/api/handlers/workflowhandler/`

API endpoints:

- `GET /templates` - List templates
- `POST /templates` - Create template
- `GET /templates/{id}` - Get template
- `GET /templates/{id}/versions` - List versions
- `POST /templates/{id}/versions` - Create new version
- `PUT /templates/{id}/versions/{versionId}/publish` - Publish version
- `PUT /templates/{id}/versions/{versionId}/rollback` - Rollback to version
- `GET /templates/{id}/export` - Export (single version or full history)
- `POST /templates/import` - Import workflow

### 4. Temporal Integration

**Directory**: `internal/core/temporaljobs/workflowjobs/`

Update for versioning:

- Workflow execution activities reference specific version
- Pass version ID in execution payload
- Handle version-specific node execution

### 5. Dependency Injection

**File**: `internal/bootstrap/modules/*/module.go`

Wire up:

- Template validator
- Version validator
- Template repository
- Version repository
- Workflow service
- Workflow handlers

## Migration Path

For existing installations (if any workflows exist):

1. Run migration
2. Migration automatically:
   - Converts existing `workflows` → `workflow_templates`
   - Creates version 1 for each template
   - Sets version 1 as published
   - Updates all foreign keys

## Version Lifecycle Example

```
1. User creates template "Shipment Update Workflow"
   → Template created with ID wft_xxx

2. User configures nodes and connections
   → Version 1 created (status: Draft)

3. User publishes
   → Version 1 status: Draft → Published
   → Template.published_version_id = version_1_id

4. Workflow runs
   → Instance.workflow_template_id = wft_xxx
   → Instance.workflow_version_id = version_1_id

5. User wants to make changes
   → Version 2 created (status: Draft, cloned from v1)
   → Version 1 remains Published

6. User publishes version 2
   → Version 2 status: Draft → Published
   → Version 1 status: Published → Archived
   → Template.published_version_id = version_2_id

7. User rolls back to version 1
   → Version 1 status: Archived → Published
   → Version 2 status: Published → Archived
   → Template.published_version_id = version_1_id
```

## Benefits of This Design

✅ **Audit Trail**: Perfect tracking of what version was executed when
✅ **Rollback**: Easy rollback to any previous version
✅ **Testing**: Create drafts without affecting production
✅ **Stable References**: Foreign keys use template ID (never changes)
✅ **Extensibility**: Easy to add features like version tags, branching, A/B testing
✅ **Performance**: Indexed queries for published versions
✅ **Data Integrity**: Constraints ensure only one published version per template
