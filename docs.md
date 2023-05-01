# Monta Documentation

## Django Admin Commands

### Setup Celery Beat

```bash
py manage.py setupcelerybeat
```

#### Description

`setupcelerybeat` command creates the necessary interval schedules for period tasks.

### Install Plugin

```bash
py manage.py install_plugin
```

#### Description

`install_plugin` command installs a plugin from a zip file.

### PSQL Listener

```bash
py manage.py psql_listener
```

#### Description

`psql_listener` command listens to PostgreSQL notifications and executes the corresponding tasks.

### Wait For DB

```bash
py manage.py wait_for_db
```

#### Description

`wait_for_db` command waits for the database to be available.

### Create Test Users

```bash
py manage.py createtestusers
```

#### Description

`createtestusers` command creates test users.

### Create System User

```bash
py manage.py createsystemuser
```

#### Description

`createsystemuser` command creates a system user.

### Setup PSQL Triggers

```postgresql
CREATE OR REPLACE FUNCTION notify_table_change_alert_update()
    RETURNS trigger
    LANGUAGE 'plpgsql'
AS
$BODY$
DECLARE
BEGIN
    IF (TG_OP = 'INSERT' OR TG_OP = 'UPDATE') THEN
        PERFORM pg_notify('table_change_alert_updated', NEW.id::text);
    END IF;
    RETURN NULL;
END
$BODY$;
```

```postgresql
CREATE OR REPLACE TRIGGER table_change_alert_update_trigger
    AFTER INSERT OR UPDATE
    ON public.table_change_alert
    FOR EACH ROW
EXECUTE PROCEDURE notify_table_change_alert_update();
```

#### Description

`notify_table_change_alert_update` function notifies the `table_change_alert_updated` channel when a row is inserted or
updated in the `table_change_alert` table.

## Celery Task Schedules

### Send Expired Rates Notification

Name: `send_expired_rates_notification`

Task: `dispatch.tasks.send_expired_rates_notification`

Interval Schedule: `Every Day`

Start Datetime: `now`

### Auto Mass Order Billing

Name: `auto_mass_order_billing`

Task: `auto_mass_billing_for_all_orgs`

Interval Schedule: `Every 15 minutes`

Start Datetime: `now`

    Note: This task should only be enabled if the customer has purchased the billing module, and auto-billing package.


