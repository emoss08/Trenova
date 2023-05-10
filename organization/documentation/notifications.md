# Notification Management

## Introduction

Monta empowers users to customize notifications that can be triggered by a range of events, tailored to their specific
needs. For instance, a user may opt to receive a notification when a new user or project is created. The notification
system is exceptionally versatile, and extensible, providing developers with the flexibility to readily add new
notification types as required.

As one of its out-of-the-box features, Monta enables the seamless delivery of notifications for expired contracts, by
offering users the ability to define a notification template in Python String Formatting Syntax. Upon the expiration of
a contract, the template
can be executed to send an email, or carry out any other desired action. This customizable solution provides users with
enhanced control over their notification systems.

### API Endpoints

- GET /api/notification_types - Get all notification types
- GET /api/notification_types/{id} - Get notification type by id
- PUT /api/notification_types/{id} - Update notification type by id
- PATCH /api/notification_types/{id} - Partially update notification type by id
- GET /api/notification_settings - Get all notification settings
- GET /api/notification_settings/{id} - Get notification setting by id
- PUT /api/notification_settings/{id} - Update notification setting by id
- PATCH /api/notification_settings/{id} - Partially update notification setting by id

### Custom Subject / Content Example

#### Rate Expiration Notification

Custom Subject - Rate {rate.rate_number} has expired!

Custom Content - Rate {rate.rate_number} for {rate.customer.name} has expired on {rate.expiration_date}

