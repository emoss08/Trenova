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
import uuid
from typing import TYPE_CHECKING

from celery import shared_task
from celery_singleton import Singleton
from django.db.models import QuerySet

from accounts.models import User
from backend.celery import app
from billing import selectors, services
from core.exceptions import ServiceException
from organization.models import Organization
from utils.types import ModelUUID

if TYPE_CHECKING:
    from celery.app.task import Task


@app.task(
    name="auto_mass_billing_for_all_orgs",
    bind=True,
    max_retries=3,
    default_retry_delay=60,
    base=Singleton,
)
def automate_mass_order_billing(self: "Task") -> str:
    """Automated Mass Billing Tasks that uses system user to bill orders.

    Filter the database for the Organizations that have auto bill orders enabled and call the mass_order_billing_service
    service to bill the orders.

    Args:
        self (celery.app.task.Task): The task object

    Returns:
        str: A string containing the results of the automated mass billing tasks.

    Raises:
        ServiceException: If the Order does not exist in the database.
    """

    # TODO: Remove this once we have a better way to get the system user
    system_user = User.objects.get(username="sys")

    # Get the organizations that have auto bill orders enabled
    organizations: QuerySet[Organization] = Organization.objects.filter(
        billing_control__auto_bill_orders=True
    )
    results = []

    # For each organization, call the mass_order_billing_service service to bill the orders
    for organization in organizations:
        try:
            services.mass_order_billing_service(
                user_id=system_user.id, task_id=str(self.request.id)
            )
            results.append(
                f"Automated Mass Billing Task for {organization.name} was successful."
            )
        except ServiceException as exc:
            raise self.retry(exc=exc) from exc
    if results:
        return "\n".join(results)
    return "No organizations have auto billing enabled."


@shared_task(name="transfer_to_billing_task", bind=True, base=Singleton)
def transfer_to_billing_task(
    self: "Task", *, user_id: str, order_pros: list[str]
) -> None:
    """
    Starts a Celery task to transfer the specified order(s) to billing for the logged-in user.

    Args:
        self: The Celery task instance.
        user_id: A string representing the ID of the user who initiated the transfer.
        order_pros: A list of strings representing the order IDs to transfer.

    Returns:
        None.

    Raises:
        self.retry: If an ObjectDoesNotExist exception is raised while processing the task.

    This Celery task function calls the `transfer_to_billing_queue_service` function to create BillingQueue objects
    for each order in the provided list of order IDs, and updates the transfer status and transfer date of each order.

    If an ObjectDoesNotExist exception is raised while processing the task, the Celery task will automatically retry
    the task until it succeeds, with an exponential backoff strategy.

    The `transfer_to_billing_queue_service` function is called to perform the actual transfer of the specified order(s).
    If this operation raises an ObjectDoesNotExist exception, the function will retry the task with an exponential
    backoff strategy until it succeeds.

    Finally, the function returns None.
    """

    try:
        services.transfer_to_billing_queue_service(
            user_id=user_id, order_pros=order_pros, task_id=str(self.request.id)
        )
    except ServiceException as exc:
        raise self.retry(exc=exc) from exc


@shared_task(
    name="bill_invoice_task",
    bind=True,
    max_retries=3,
    default_retry_delay=60,
    base=Singleton,
)
def bill_invoice_task(self: "Task", user_id: ModelUUID, invoice_id: ModelUUID) -> None:
    """Bill Order

    Query the database for the Order and call the bill_order
    service to bill the order.

    Args:
        self (celery.app.task.Task): The task object
        user_id (str): User ID
        invoice_id (ModelUUID): Invoice ID

    Returns:
        None: None

    Raises:
        ObjectDoesNotExist: If the Order does not exist in the database.
    """

    try:
        if invoice := selectors.get_invoice_by_id(invoice_id=invoice_id):
            services.bill_orders(
                invoices=invoice, user_id=user_id, task_id=str(self.request.id)
            )
        else:
            return None
    except ServiceException as exc:
        raise self.retry(exc=exc) from exc


@shared_task(
    name="mass_order_billing_task",
    bind=True,
    max_retries=3,
    default_retry_delay=60,
    base=Singleton,
)
def mass_order_bill_task(self: "Task", *, user_id: ModelUUID) -> None:
    """Bill Order

    Args:
        self (celery.app.task.Task): The task object
        user_id (str): User ID

    Returns:
        None: None

    Raises:
        ObjectDoesNotExist: If the Order does not exist in the database.
    """
    try:
        services.mass_order_billing_service(user_id=user_id, task_id=str(self.request.id))
    except ServiceException as exc:
        raise self.retry(exc=exc) from exc
