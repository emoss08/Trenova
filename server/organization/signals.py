# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------

from typing import Any

from django.conf import settings
from django.db import connections
from django.db.models.signals import post_migrate
from django.dispatch import Signal, receiver

from accounting.models import AccountingControl
from billing.models import BillingControl
from dispatch.models import DispatchControl, FeasibilityToolControl
from invoicing.models import InvoiceControl
from organization import models
from organization.models import EmailControl
from organization.services.psql_triggers import drop_trigger_and_function
from organization.services.table_change import (
    create_trigger_based_on_db_action,
    drop_trigger_and_create,
    set_trigger_name_requirements,
)
from route.models import RouteControl
from shipment.models import ShipmentControl

restart_psql_listener_signal = Signal()


def create_dispatch_control(
    sender: models.Organization,
    instance: models.Organization,
    created: bool,
    **kwargs: Any,
) -> None:
    """Create a DispatchControl model instance for a new Organization model instance.

    This function is called as a signal when an Organization model instance is saved.
    If a new Organization instance is created, it creates a DispatchControl model
    instance with the organization reference.

    Args:
        sender (models.Organization): The class of the sending instance.
        instance (models.Organization): The instance of the Organization model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """

    if created:
        DispatchControl.objects.create(
            organization=instance, business_unit=instance.business_unit
        )


def create_accounting_control(
    sender: models.Organization,
    instance: models.Organization,
    created: bool,
    **kwargs: Any,
) -> None:
    """Create a AccountingControl model instance for a new Organization model instance.

    This function is called as a signal when an Organization model instance is saved.
    If a new Organization instance is created, it creates a AccountingControl model
    instance with the organization reference.

    Args:
        sender (models.Organization): The class of the sending instance.
        instance (models.Organization): The instance of the Organization model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """

    if created:
        AccountingControl.objects.create(
            organization=instance, business_unit=instance.business_unit
        )


def create_shipment_control(
    sender: models.Organization,
    instance: models.Organization,
    created: bool,
    **kwargs: Any,
) -> None:
    """Create an ShipmentControl model instance for a new Organization model instance.

    This function is called as a signal when an Organization model instance is saved.
    If a new Organization instance is created, it creates an ShipmentControl model
    instance with the organization reference.

    Args:
        sender (models.Organization): The class of the sending instance.
        instance (models.Organization): The instance of the Organization model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if created:
        ShipmentControl.objects.create(
            organization=instance, business_unit=instance.business_unit
        )


def create_route_control(
    instance: models.Organization, created: bool, **kwargs: Any
) -> None:
    """Create a RouteControl model instance for a new Organization model instance.

    This function is called as a signal when an Organization model instance is saved.
    If a new Organization instance is created, it creates a RouteControl model
    instance with the organization reference.

    Args:
        instance (models.Organization): The instance of the Organization model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if created:
        RouteControl.objects.create(
            organization=instance, business_unit=instance.business_unit
        )


def create_billing_control(
    sender: models.Organization,
    instance: models.Organization,
    created: bool,
    **kwargs: Any,
) -> None:
    """Create a BillingControl model instance for a new Organization model instance.

    This function is called as a signal when an Organization model instance is saved.
    If a new Organization instance is created, it creates a BillingControl model
    instance with the organization reference.

    Args:
        sender (models.Organization): The class of the sending instance.
        instance (models.Organization): The instance of the Organization model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if created:
        BillingControl.objects.create(
            organization=instance, business_unit=instance.business_unit
        )


def create_email_control(
    sender: models.Organization,
    instance: models.Organization,
    created: bool,
    **kwargs: Any,
) -> None:
    """Create an EmailControl model instance for a new Organization model instance.

    This function is called as a signal when an Organization model instance is saved.
    If a new Organization instance is created, it creates an EmailControl model
    instance with the organization reference.

    Args:
        sender (models.Organization): The class of the sending instance.
        instance (models.Organization): The instance of the Organization model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if created:
        EmailControl.objects.create(
            organization=instance, business_unit=instance.business_unit
        )


def create_invoice_control(
    sender: models.Organization,
    instance: models.Organization,
    created: bool,
    **kwargs: Any,
) -> None:
    """Create an InvoiceControl model instance for a new Organization model instance.

    This function is called as a signal when an Organization model instance is saved.
    If a new Organization instance is created, it creates an InvoiceControl model
    instance with the organization reference.

    Args:
        sender (models.Organization): The class of the sending instance.
        instance (models.Organization): The instance of the Organization model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if created:
        InvoiceControl.objects.create(
            organization=instance, business_unit=instance.business_unit
        )


def create_feasibility_tool_control(
    sender: models.Organization,
    instance: models.Organization,
    created: bool,
    **kwargs: Any,
) -> None:
    """Create an FeasibilityToolControl model instance for a new Organization model instance.

    This function is called as a signal when an Organization model instance is saved.
    If a new Organization instance is created, it creates an FeasibilityToolControl model
    instance with the organization reference.

    Args:
        sender (models.Organization): The class of the sending instance.
        instance (models.Organization): The instance of the Organization model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if created:
        FeasibilityToolControl.objects.create(
            organization=instance, business_unit=instance.business_unit
        )


def create_depot_detail(
    sender: models.Depot, instance: models.Depot, created: bool, **kwargs: Any
) -> None:
    """Create a DepotDetail model instance for a new Depot model instance.

    This function is called as a signal when a Depot model instance is saved.
    If a new Depot instance is created, it creates a DepotDetail model
    instance with the organization and depot references.

    Args:
        sender (models.Depot): The class of the sending instance.
        instance (models.Depot): The instance of the Depot model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if created:
        models.DepotDetail.objects.create(
            organization=instance.organization,
            depot=instance,
            business_unit=instance.organization.business_unit,
        )


def save_trigger_name_requirements(
    sender: models.TableChangeAlert, instance: models.TableChangeAlert, **kwargs: Any
) -> None:
    """Save trigger name requirements for a TableChangeAlert model instance.

    This function is called as a signal when a TableChangeAlert model instance is saved.
    It sets trigger name requirements for the instance using the set_trigger_name_requirements
    function.

    Args:
        sender (models.TableChangeAlert): The class of the sending instance.
        instance (models.TableChangeAlert): The instance of the TableChangeAlert model being saved.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if instance.source == models.TableChangeAlert.SourceChoices.KAFKA:
        return

    set_trigger_name_requirements(instance=instance)


def create_trigger_signal(
    sender: models.TableChangeAlert,
    instance: models.TableChangeAlert,
    created: bool,
    **kwargs: Any,
) -> None:
    """Create a trigger for a new TableChangeAlert model instance.

    This function is called as a signal when a TableChangeAlert model instance is saved.
    If a new TableChangeAlert instance is created, it creates a trigger based on the
    database action using the create_trigger_based_on_db_action function.

    Args:
        sender (models.TableChangeAlert): The class of the sending instance.
        instance (models.TableChangeAlert): The instance of the TableChangeAlert model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if (
        created
        and instance.source == models.TableChangeAlert.SourceChoices.POSTGRES
        and instance.table
    ):
        create_trigger_based_on_db_action(
            instance=instance,
            organization_id=instance.organization_id,
        )


def drop_trigger_and_function_signal(
    sender: models.TableChangeAlert, instance: models.TableChangeAlert, **kwargs: Any
) -> None:
    """Drop the trigger and associated function for a TableChangeAlert model instance.

    This function is called as a signal before a TableChangeAlert model instance is deleted.
    It drops the trigger and associated function using the drop_trigger_and_function function.

    Args:
        sender (models.TableChangeAlert): The class of the sending instance.
        instance (models.TableChangeAlert): The instance of the TableChangeAlert model being deleted.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if instance.source == models.TableChangeAlert.SourceChoices.KAFKA:
        return

    drop_trigger_and_function(
        trigger_name=instance.trigger_name,
        function_name=instance.function_name,
        table_name=instance.table,
    )


def delete_and_add_new_trigger(
    sender: models.TableChangeAlert, instance: models.TableChangeAlert, **kwargs: Any
) -> None:
    """Delete and add a new trigger for a TableChangeAlert model instance.

    This function is called as a signal when a TableChangeAlert model instance is saved.
    If the table attribute of the instance has changed, it deletes the existing trigger
    and creates a new one using the drop_trigger_and_create function.

    Args:
        sender (models.TableChangeAlert): The class of the sending instance.
        instance (models.TableChangeAlert): The instance of the TableChangeAlert model being saved.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if instance.source == models.TableChangeAlert.SourceChoices.KAFKA:
        return

    try:
        old_instance = sender.objects.get(pk__exact=instance.pk)
    except sender.DoesNotExist:
        return

    if old_instance.table != instance.table:
        drop_trigger_and_create(instance=instance)

    if old_instance.database_action != instance.database_action:
        drop_trigger_and_create(instance=instance)


def delete_and_recreate_trigger_and_function(
    sender: models.TableChangeAlert, instance: models.TableChangeAlert, **kwargs: Any
) -> None:
    """
    If the database action on a trigger has changed, drop the trigger,
    and function and recreate it to reflect the new changes.

    Args:
        sender (models.TableChangeAlert): The class of the sending instance.
        instance (models.TableChangeAlert): The instance of the TableChangeAlert model being saved.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if instance.source == models.TableChangeAlert.SourceChoices.KAFKA:
        return

    try:
        old_instance = sender.objects.get(pk__exact=instance.pk)
    except sender.DoesNotExist:
        return

    if old_instance.database_action != instance.database_action:
        drop_trigger_and_function(
            trigger_name=old_instance.trigger_name,
            function_name=old_instance.function_name,
            table_name=instance.table,
        )
        create_trigger_based_on_db_action(
            instance=instance, organization_id=instance.organization_id
        )

    if old_instance.table != instance.table:
        drop_trigger_and_function(
            trigger_name=old_instance.trigger_name,
            function_name=old_instance.function_name,
            table_name=old_instance.table,
        )
        create_trigger_based_on_db_action(
            instance=instance, organization_id=instance.organization_id
        )


def create_notification_settings(
    sender: models.NotificationType,
    instance: models.NotificationType,
    created: bool,
    **kwargs: Any,
) -> None:
    if created:
        models.NotificationSetting.objects.get_or_create(
            business_unit=instance.business_unit,
            organization=instance.organization,
            notification_type=instance,
        )


@receiver(post_migrate)
def set_replica_identity(sender: Any, **kwargs: Any) -> None:
    """
    Signal receiver function to set REPLICA IDENTITY FULL on all tables in a PostgreSQL database after a migration event.

    In the context of logical decoding and change data capture, the REPLICA IDENTITY configuration of a table
    determines what information is included in the WAL (Write-Ahead Log) for DELETE and UPDATE operations. The
    FULL option specifies that the entire row should be logged.

    This function is triggered after a Django migration event. It checks if the database engine of the migrated
    database is PostgreSQL. If it is, for each table in the public schema that doesn't already have
    REPLICA IDENTITY set to FULL, it will issue an ALTER TABLE statement to change the REPLICA IDENTITY setting to FULL.

    This function makes use of the Django database connection and cursor to execute raw SQL commands. The function
    gets the list of table names from the system catalog table, pg_class, and then loops over this list to execute
    the ALTER TABLE command.

    Args:
        sender (Any): The signal sender.
        **kwargs (Any): Additional keyword arguments. In this case, we're interested in 'using' which indicates
                  the alias of the database on which the migration was performed.

    Returns:
        None: this function does not return anything.
    """
    db_alias = kwargs["using"]
    # Check if the database engine is Postgres
    if "postgresql" in settings.DATABASES[db_alias]["ENGINE"]:
        connection = connections[db_alias]
        cursor = connection.cursor()

        # get all table names where replica identity is not full
        cursor.execute(
            """
            SELECT c.relname
            FROM pg_catalog.pg_class c
            JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
            WHERE c.relkind = 'r' AND n.nspname = 'public' AND c.relreplident != 'f'
        """
        )
        table_names = [row[0] for row in cursor.fetchall()]

        # set REPLICA IDENTITY FULL for each table
        for table_name in table_names:
            cursor.execute(f'ALTER TABLE "{table_name}" REPLICA IDENTITY FULL;')
