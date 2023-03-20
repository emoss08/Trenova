"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""

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

from backend.celery import app
from django.core.exceptions import ObjectDoesNotExist

from accounts.models import User
from billing.services import mass_order_billing, single_order_billing
from billing.services.transfer_to_billing import transfer_to_billing_queue_service
from order.models import Order
from order.services.consolidate_pdf import combine_pdfs
from organization.models import Organization
from utils.types import MODEL_UUID


@app.task(
    name="consolidate_order_documentation",
    bind=True,
    max_retries=3,
    default_retry_delay=60,
)
def consolidate_order_documentation(self, order_id: str) -> None:
    """Consolidate Order

    Query the database for the Order and call the consolidate_pdf
    service to combine the PDFs into a single PDF.

    Args:
        self (celery.app.task.Task): The task object
        order_id (str): Order ID

    Returns:
        None: None

    Raises:
        ObjectDoesNotExist: If the Order does not exist in the database.
    """

    try:
        order: Order = Order.objects.get(id=order_id)
        combine_pdfs(order=order)
    except ObjectDoesNotExist as exc:
        raise self.retry(exc=exc) from exc


@app.task(name="bill_order_task", bind=True, max_retries=3, default_retry_delay=60)
def bill_order_task(self, user_id: MODEL_UUID, order_id: str) -> None:
    """Bill Order

    Query the database for the Order and call the bill_order
    service to bill the order.

    Args:
        self (celery.app.task.Task): The task object
        user_id (str): User ID
        order_id (str): Order ID

    Returns:
        None: None

    Raises:
        ObjectDoesNotExist: If the Order does not exist in the database.
    """

    try:
        order: Order = Order.objects.get(pk=order_id)
        single_order_billing.bill_order(order=order, user_id=user_id)
    except ObjectDoesNotExist as exc:
        raise self.retry(exc=exc) from exc


@app.task(name="mass_order_bill_task", bind=True, max_retries=3, default_retry_delay=60)
def mass_order_bill_task(self, user_id: MODEL_UUID) -> None:
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
        mass_order_billing.mass_order_billing_service(
            user_id=user_id, task_id=self.request.id
        )
    except ObjectDoesNotExist as exc:
        raise self.retry(exc=exc) from exc


@app.task(name="transfer_to_billing_task", bind=True)
def transfer_to_billing_task(
    self, *, user_id: MODEL_UUID, order_pros: list[str]
) -> None:
    """
    Starts a Celery task to transfer the specified order(s) to billing for the logged in user.

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

    The function expects the following arguments:
    - self: The Celery task instance.
    - user_id: A string representing the ID of the user who initiated the transfer.
    - order_pros: A list of strings representing the order IDs to transfer.

    The `transfer_to_billing_queue_service` function is called to perform the actual transfer of the specified order(s).
    If this operation raises an ObjectDoesNotExist exception, the function will retry the task with an exponential
    backoff strategy until it succeeds.

    Finally, the function returns None.
    """
    try:
        transfer_to_billing_queue_service(
            user_id=user_id, order_pros=order_pros, task_id=self.request.id
        )
    except ObjectDoesNotExist as exc:
        raise self.retry(exc=exc) from exc


@app.task(
    name="automate_mass_order_billing", bind=True, max_retries=3, default_retry_delay=60
)
def automate_mass_order_billing(self) -> str:
    """Automated Mass Billing Tasks, that uses system user to bill orders.

    Args:
        self (celery.app.task.Task): The task object

    Returns:
        str: A string containing the results of the automated mass billing tasks.

    Raises:
        ObjectDoesNotExist: If the Order does not exist in the database.
    """
    system_user = User.objects.get(username="sys")
    organizations = Organization.objects.filter(billing_control__auto_bill_orders=True)
    results = []
    for organization in organizations:
        try:
            mass_order_billing.mass_order_billing_service(
                user_id=str(system_user.id), task_id=self.request.id
            )
            results.append(
                f"Automated Mass Billing Task for {organization.name} was successful."
            )
        except ObjectDoesNotExist as exc:
            raise self.retry(exc=exc) from exc
    if results:
        return "\n".join(results)
    return "No organizations have auto billing enabled."
