# Entity Update Notifications Feature

## Overview

This feature allows users to configure notifications when someone else updates records they own/created. The system tracks record ownership through audit logs and sends real-time notifications to owners when their records are modified.

## Architecture

### Domain Models

1. **NotificationPreference** (`internal/core/domain/notification/preferences.go`)
   - Stores user preferences for receiving notifications
   - Configurable by entity type (shipment, worker, customer, etc.)
   - Supports filtering by update types (status changes, date changes, etc.)
   - Includes quiet hours and batching options

### Services

1. **EntityUpdateService** (`internal/core/services/notification/entity_update_service.go`)
   - Handles the logic for determining when to send notifications
   - Looks up record creators from audit logs
   - Checks user preferences before sending notifications

2. **PreferenceService** (`internal/core/services/notification/preference_service.go`)
   - Manages CRUD operations for notification preferences
   - Handles permission checks for user preferences

### Database Schema

New table: `notification_preferences`
```sql
CREATE TABLE notification_preferences (
    id VARCHAR(100) PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL,
    organization_id VARCHAR(100) NOT NULL,
    business_unit_id VARCHAR(100) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    update_types TEXT[] NOT NULL,
    notify_on_all_updates BOOLEAN DEFAULT FALSE,
    notify_only_owned_records BOOLEAN DEFAULT TRUE,
    excluded_user_ids VARCHAR(100)[],
    preferred_channels VARCHAR(20)[],
    is_active BOOLEAN DEFAULT TRUE,
    -- Additional fields for timing, batching, etc.
);
```

## Integration Example

The feature is integrated into the shipment update process:

```go
// In shipment service Update method
if s.euns != nil {
    updateType := notification.UpdateTypeAny
    if original.Status != updatedEntity.Status {
        updateType = notification.UpdateTypeStatusChange
    }
    
    err = s.euns.HandleEntityUpdate(ctx, &notificationservice.EntityUpdateRequest{
        EntityType:      notification.EntityTypeShipment,
        EntityID:        updatedEntity.ID.String(),
        EntityCode:      updatedEntity.ProNumber,
        EntityName:      fmt.Sprintf("Shipment %s", updatedEntity.ProNumber),
        UpdateType:      updateType,
        UpdatedByUserID: userID,
        OrganizationID:  updatedEntity.OrganizationID,
        BusinessUnitID:  updatedEntity.BusinessUnitID,
    })
}
```

## How It Works

1. **Record Creation**: When a user creates a record (e.g., shipment), an audit entry is created with their user ID
2. **Preference Configuration**: Users configure their notification preferences through the preference service
3. **Record Update**: When another user updates the record:
   - The system checks audit logs to find the original creator
   - Checks the creator's notification preferences
   - If preferences match, sends a real-time notification via WebSocket

## API Endpoints Needed

To complete the feature, you'll need to create API endpoints for:

1. **Notification Preferences**:
   - `GET /api/notification-preferences` - List user's preferences
   - `POST /api/notification-preferences` - Create new preference
   - `PUT /api/notification-preferences/:id` - Update preference
   - `DELETE /api/notification-preferences/:id` - Delete preference

2. **User Settings Page**: Add UI for users to configure their notification preferences

## Extending to Other Entities

To add notification support for other entities:

1. Add the entity type to `notification.EntityType` enum
2. Add corresponding event type to `notification.EventType` enum
3. Integrate `EntityUpdateService.HandleEntityUpdate()` call in the entity's update method
4. Map entity type to permission resource in `mapEntityTypeToResource()`

## Configuration Options

Users can configure:
- **Entity Types**: Which types of records to receive notifications for
- **Update Types**: Specific types of updates (status changes, date changes, etc.)
- **Excluded Users**: List of users whose updates won't trigger notifications
- **Quiet Hours**: Time periods when notifications won't be sent
- **Batching**: Group notifications together instead of real-time delivery

## Next Steps

1. Create API handlers for notification preferences
2. Add UI components for preference configuration
3. Create integration tests
4. Add more entity types beyond shipments
5. Implement email notifications (currently only WebSocket)
6. Add notification history/log viewing