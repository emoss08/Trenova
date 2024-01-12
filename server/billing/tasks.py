# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
from typing import TYPE_CHECKING

from celery import shared_task
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
    # queue="high_priority",
)
def automate_mass_shipments_billing(self: "Task") -> str:
    """Automated Mass Billing Tasks that uses system user to bill shipments.

    Filter the database for the Organizations that have auto bill shipments enabled and call the mass_shipments_billing_service
    service to bill the shipments.

    Args:
        self (celery.app.task.Task): The task object

    Returns:
        str: A string containing the results of the automated mass billing tasks.

    Raises:
        ServiceException: If the shipment does not exist in the database.
    """

    # TODO: Remove this once we have a better way to get the system user
    system_user = User.objects.get(username="Trenova")

    # Get the organizations that have auto bill shipments enabled
    organizations: QuerySet[Organization] = Organization.objects.filter(
        billing_control__auto_bill_shipment=True
    )
    results = []

    # For each organization, call the mass_shipments_billing_service service to bill the shipments
    for organization in organizations:
        try:
            services.mass_shipments_billing_service(
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


@shared_task(
    name="transfer_to_billing_task",
    bind=True,
    # queue="high_priority"
)
def transfer_to_billing_task(
    self: "Task", *, user_id: str, shipment_pros: list[str]
) -> None:
    """
    Starts a Celery task to transfer the specified Shipment(s) to billing for the logged-in user.

    Args:
        self: The Celery task instance.
        user_id: A string representing the ID of the user who initiated the transfer.
        shipment_pros: A list of strings representing the order IDs to transfer.

    Returns:
        None.

    Raises:
        self.retry: If an ObjectDoesNotExist exception is raised while processing the task.

    This Celery task function calls the `transfer_to_billing_queue_service` function to create BillingQueue objects
    for each order in the provided list of order IDs, and updates the transfer status and transfer date of each shipment.

    If an ObjectDoesNotExist exception is raised while processing the task, the Celery task will automatically retry
    the task until it succeeds, with an exponential backoff strategy.

    The `transfer_to_billing_queue_service` function is called to perform the actual transfer of the specified Shipment(s).
    If this operation raises an ObjectDoesNotExist exception, the function will retry the task with an exponential
    backoff strategy until it succeeds.

    Finally, the function returns None.
    """

    try:
        services.transfer_to_billing_queue_service(
            user_id=user_id, shipment_pros=shipment_pros, task_id=str(self.request.id)
        )
    except ServiceException as exc:
        raise self.retry(exc=exc) from exc


# @shared_task(
#     name="transfer_to_billing_task",
#     bind=True,
#     base=Singleton,
#     # queue="high_priority"
# )
# def transfer_to_billing_task(
#     self: "Task", *, user_id: str, shipment_pros: list[str] | None
# ) -> None:
#     try:
#         services.transfer_to_billing_queue_service(
#             user_id=user_id, shipment_pros=shipment_pros, task_id=str(self.request.id)
#         )
#     except ServiceException as exc:
#         raise self.retry(exc=exc) from exc


@shared_task(
    name="bill_invoice_task",
    bind=True,
    max_retries=3,
    default_retry_delay=60,
    # queue="high_priority",
)
def bill_invoice_task(self: "Task", user_id: ModelUUID, invoice_id: ModelUUID) -> None:
    """Bill Order

    Query the database for the Order and call the bill_shipments
    service to bill the shipment.

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
            services.bill_shipments(
                invoices=invoice, user_id=user_id, task_id=str(self.request.id)
            )
        else:
            return None
    except ServiceException as exc:
        raise self.retry(exc=exc) from exc


@shared_task(
    name="mass_shipments_billing_task",
    bind=True,
    max_retries=3,
    default_retry_delay=60,
    # queue="high_priority",
)
def mass_shipments_bill_task(self: "Task", *, user_id: ModelUUID) -> None:
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
        services.mass_shipments_billing_service(
            user_id=user_id, task_id=str(self.request.id)
        )
    except ServiceException as exc:
        raise self.retry(exc=exc) from exc


@shared_task(
    name="mark_shipment_as_paid_task",
    bind=True,
    max_retries=3,
    default_retry_delay=60,
)
def mark_shipment_as_paid_task(self: "Task") -> None:
    try:
        # Get all paid invoice numbers from InvoicePaymentDetail
        paid_invoice_numbers = selectors.get_paid_invoices().values_list(
            "invoice", flat=True
        )

        # Get all unpaid invoices from BillingHistory
        unpaid_invoices = selectors.get_unpaid_invoices()

        # Update the payment_received field to True for all unpaid invoices that are in the paid_invoice_numbers list
        unpaid_invoices.filter(id__in=paid_invoice_numbers).update(
            payment_received=True
        )

    except ServiceException as exc:
        raise self.retry(exc=exc) from exc
