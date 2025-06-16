# Entity Update Notifications Feature

## Overview

This feature allows users to configure notifications when someone else updates records they own/created. The system uses a database-driven approach with an audit listener service that monitors audit logs and sends real-time notifications to owners when their records are modified.

## Architecture

### Domain Models

1. **NotificationPreference** (`internal/core/domain/notification/preferences.go`)
   - Stores user preferences for receiving notifications
   - Configurable by resource type (using existing permission.Resource types)
   - Supports filtering by update types (status changes, date changes, etc.)
   - Includes quiet hours and batching options
   - Uses existing permission resources instead of creating new entity types

### Services

1. **AuditListenerService** (`internal/core/services/notification/audit_listener_service.go`)
   - Runs as a background service alongside the main application
   - Polls the audit log every 5 seconds for new UPDATE actions
   - Determines record ownership by finding the original CREATE audit entry
   - Checks user notification preferences
   - Sends notifications through the existing notification service

2. **PreferenceService** (`internal/core/services/notification/preference_service.go`)
   - Manages CRUD operations for notification preferences
   - Validates that users have unique preferences per resource type
   - Ensures users can only manage their own preferences

### Database

1. **notification_preferences table**
   - Stores user notification configuration
   - Indexed on user_id, resource, and is_active for efficient queries
   - Uses the existing permission.Resource enum for resource types

### Key Benefits of Database-Driven Approach

1. **Scalability**: No need to modify each service when adding new entity types
2. **Decoupling**: Services don't need to know about notification logic
3. **Reliability**: Uses existing audit log infrastructure
4. **Flexibility**: Easy to add new resource types or update types
5. **Performance**: Efficient polling with timestamp-based queries

## Implementation Details

### Notification Flow

1. User performs an UPDATE operation on any resource
2. Audit service logs the action (already happens automatically)
3. AuditListenerService detects the new audit entry
4. Service finds the original creator of the resource
5. Checks if creator has notification preferences for this resource type
6. Validates preferences (quiet hours, excluded users, etc.)
7. Sends notification through WebSocket to the owner

### API Endpoints

- `GET /api/v1/notification-preferences` - List user's preferences
- `POST /api/v1/notification-preferences` - Create new preference
- `PUT /api/v1/notification-preferences/:id` - Update preference
- `DELETE /api/v1/notification-preferences/:id` - Delete preference
- `GET /api/v1/notification-preferences/user` - Get current user's preferences

### Configuration Example

```json
{
  "resource": "shipment",
  "updateTypes": ["status_change", "assignment", "date_change"],
  "notifyOnAllUpdates": false,
  "excludedUserIds": ["user_abc123"],
  "preferredChannels": ["user"],
  "quietHoursEnabled": true,
  "quietHoursStart": "22:00",
  "quietHoursEnd": "08:00",
  "timezone": "America/New_York",
  "batchNotifications": false
}
```

### Notification Example

When Bob Ross creates shipment #123 and Angie Ross updates the arrival time:

```json
{
  "eventType": "entity.shipment.updated",
  "title": "Shipment Updated",
  "message": "Angie Ross updated Shipment: arrival_time changed",
  "priority": "medium",
  "relatedEntities": [{
    "type": "shipment",
    "id": "shp_123",
    "name": "Shipment",
    "url": "/shipments/shp_123"
  }],
  "actions": [{
    "id": "view",
    "label": "View Details",
    "type": "link",
    "endpoint": "/shipments/shp_123"
  }]
}
```

## Future Enhancements

1. **Email Notifications**: Add email as a notification channel
2. **SMS Notifications**: Support SMS for critical updates
3. **Notification Templates**: Allow customization of notification messages
4. **Advanced Filtering**: Filter by specific fields or field values
5. **Webhooks**: Send notifications to external systems
6. **Batch Digests**: Email summaries of batched notifications
7. **Database Triggers**: Use PostgreSQL LISTEN/NOTIFY for real-time updates instead of polling

## Security Considerations

1. Users can only create/modify their own notification preferences
2. Audit log access is read-only for the listener service
3. Notifications are sent only through authenticated WebSocket connections
4. Resource-level permissions are respected (users only get notified about resources they have access to)