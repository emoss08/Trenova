# Workflow Versioning Design

## Overview

The workflow system needs versioning support to allow users to:

- Create new versions when modifying workflows
- Maintain a complete audit trail of changes
- Roll back to previous versions
- Test changes without affecting production workflows
- Compare versions to see what changed

## Design Approach

### Two-Table Design: Templates + Versions

We use a **template-version pattern** where:

1. **Workflow Templates** represent the logical workflow concept
2. **Workflow Versions** represent specific configurations of that workflow

```
workflow_templates (1) ----< (many) workflow_versions
                     ↓
                published_version_id (points to active version)
```

### Key Benefits

- **Stable References**: Foreign keys reference the template ID, which never changes
- **Clear Separation**: Template = "what workflow is this?", Version = "how is it configured?"
- **Easy Querying**: `SELECT * FROM workflow_versions WHERE template_id = X ORDER BY version_number`
- **Atomic Publishing**: Change `published_version_id` to activate a different version

## Database Schema

### workflow_templates

Contains stable metadata about the workflow:

| Field | Type | Description |
|-------|------|-------------|
| id | VARCHAR(100) | Stable identifier (never changes) |
| organization_id | VARCHAR(100) | Multi-tenancy |
| business_unit_id | VARCHAR(100) | Multi-tenancy |
| name | VARCHAR(255) | Base workflow name (can be overridden in versions) |
| description | TEXT | Base description |
| is_template | BOOLEAN | Whether this is a shareable template |
| published_version_id | VARCHAR(100) | FK to workflow_versions (currently active version) |
| created_by_id | VARCHAR(100) | User who created the workflow |
| updated_by_id | VARCHAR(100) | User who last updated |
| created_at | BIGINT | Unix timestamp |
| updated_at | BIGINT | Unix timestamp |
| version | BIGINT | Optimistic locking version |

### workflow_versions

Contains version-specific configuration:

| Field | Type | Description |
|-------|------|-------------|
| id | VARCHAR(100) | Unique version identifier |
| organization_id | VARCHAR(100) | Multi-tenancy |
| business_unit_id | VARCHAR(100) | Multi-tenancy |
| workflow_template_id | VARCHAR(100) | FK to workflow_templates |
| version_number | INTEGER | Auto-incrementing per template (1, 2, 3...) |
| name | VARCHAR(255) | Version-specific name override |
| description | TEXT | Version-specific description |
| trigger_type | ENUM | Manual, Scheduled, Event |
| status | ENUM | Active, Inactive, Draft |
| version_status | ENUM | Draft, Published, Archived |
| schedule_config | JSONB | Cron schedule configuration |
| trigger_config | JSONB | Event trigger configuration |
| change_description | TEXT | What changed in this version |
| created_by_id | VARCHAR(100) | User who created this version |
| created_at | BIGINT | Unix timestamp |
| version | BIGINT | Optimistic locking version |

### workflow_nodes

**CHANGE**: Now references `workflow_version_id` instead of `workflow_id`

```sql
workflow_version_id VARCHAR(100) NOT NULL
FOREIGN KEY (workflow_version_id, organization_id, business_unit_id)
  REFERENCES workflow_versions(id, organization_id, business_unit_id)
```

### workflow_connections

**CHANGE**: Now references `workflow_version_id` instead of `workflow_id`

```sql
workflow_version_id VARCHAR(100) NOT NULL
```

### workflow_instances

**CHANGE**: References both template and version:

```sql
workflow_template_id VARCHAR(100) NOT NULL  -- Which workflow concept
workflow_version_id VARCHAR(100) NOT NULL   -- Which specific version was executed
```

## Version Lifecycle

### Creating a New Version

1. User edits a workflow
2. System creates new `workflow_versions` record with incremented `version_number`
3. New version starts as `version_status = 'Draft'`
4. User can continue editing the draft
5. When ready, user publishes the draft

### Publishing a Version

1. Set `version_status = 'Published'` on the new version
2. Set `version_status = 'Archived'` on the previously published version
3. Update `workflow_templates.published_version_id` to point to the new version
4. **Constraint**: Only one version can have `version_status = 'Published'` per template

### Rolling Back

1. User selects a previous version
2. System sets `version_status = 'Published'` on the old version
3. Updates `workflow_templates.published_version_id`
4. Previous "published" version becomes "Archived"

### Creating a New Draft from Published

When user wants to edit:

1. Clone the currently published version
2. Create new record with incremented `version_number`
3. Set `version_status = 'Draft'`
4. User edits the draft
5. Published version remains active until draft is published

## Version Numbering

- **Simple integers**: 1, 2, 3, 4... (easier than semantic versioning for this use case)
- **Auto-increment per template**: Version 1 for template A is independent of version 1 for template B
- **Never reuse numbers**: Deleted versions don't free up their numbers
- **Unique constraint**: `UNIQUE(workflow_template_id, organization_id, business_unit_id, version_number)`

## Execution Behavior

### Which Version Runs?

When a workflow is triggered:

1. **Manual execution**: User can choose which version to run (default: published)
2. **Scheduled execution**: Always uses the published version
3. **Event-triggered execution**: Always uses the published version

### Instance Tracking

`workflow_instances` records:

- `workflow_template_id`: Which workflow concept
- `workflow_version_id`: Exactly which version was executed
- This provides perfect audit trail: "Instance X ran using version 5 of workflow Y"

## JSON Import/Export

### Export Options

**Export Single Version** (default):

```json
{
  "templateName": "Update Shipment Workflow",
  "version": 3,
  "versionStatus": "Published",
  "changeDescription": "Added email notification step",
  "nodes": [...],
  "connections": [...]
}
```

**Export Full History** (optional):

```json
{
  "template": {
    "name": "Update Shipment Workflow",
    "description": "...",
    "publishedVersion": 3
  },
  "versions": [
    { "version": 1, "nodes": [...], "connections": [...] },
    { "version": 2, "nodes": [...], "connections": [...] },
    { "version": 3, "nodes": [...], "connections": [...] }
  ]
}
```

### Import Behavior

- **Import as new template**: Creates new template + version 1
- **Import as new version**: Adds to existing template with next version number
- **Never overwrites**: Imports always create new records

## Migration Strategy

Since this is a breaking change to the schema, we need a migration path:

1. **Rename** `workflows` → `workflow_templates`
2. **Create** `workflow_versions` table
3. **Migrate data**: For each existing workflow:
   - Keep template record
   - Create version 1 record with workflow configuration
   - Update `published_version_id` to point to version 1
4. **Update foreign keys** in nodes/connections/instances
5. **Update indexes** and constraints

## UI Considerations

### Version Selector

- Dropdown showing all versions with labels: "v3 (Published)", "v4 (Draft)", "v2 (Archived)"
- Show version number, status, created date, created by

### Version Comparison

- Side-by-side diff showing:
  - Configuration changes
  - Node additions/removals
  - Connection changes

### Version History Panel

- Table showing all versions with:
  - Version number
  - Status
  - Created by
  - Created date
  - Change description
  - Actions: View, Publish, Rollback

## Performance Considerations

- **Index on template_id**: Fast retrieval of all versions for a template
- **Index on published versions**: `WHERE version_status = 'Published'` is a common query
- **Partial index**: Only index published/draft versions (not archived) if archive table grows large
- **Soft delete archived versions**: Keep audit trail without cluttering active queries

## Future Enhancements

1. **Version Tags**: Allow users to tag versions (e.g., "Stable", "Pre-release", "Rollback Point")
2. **Scheduled Publishing**: Schedule a draft to go live at a specific time
3. **A/B Testing**: Run different versions simultaneously with traffic splitting
4. **Version Branching**: Create branches from a version (like git branches)
5. **Version Merging**: Merge changes from one version into another
