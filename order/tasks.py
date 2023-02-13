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

from celery import shared_task
from django.core.exceptions import ObjectDoesNotExist

from accounts.models import User
from order.models import Order
from order.services.consolidate_pdf import combine_pdfs
from organization.models import Organization
from billing.services import mass_order_billing, single_order_billing

@shared_task(bind=True)
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

@shared_task(bind=True)
def bill_order_task(self, user_id: str, order_id: str) -> None:
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


@shared_task(bind=True)
def mass_order_bill_task(self, user_id: str) -> None:
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
        mass_order_billing.mass_order_billing_service(user_id=user_id, task_id=self.request.id)
    except ObjectDoesNotExist as exc:
        raise self.retry(exc=exc) from exc

@shared_task(bind=True)
def automate_mass_order_billing(self) -> str:
    """Automated Mass Billing Tasks, that uses system user to bill orders.

    Args:
        self (celery.app.task.Task): The task object

    Returns:
        None: None
    """
    system_user = User.objects.get(username="sys")
    organizations = Organization.objects.filter(billing_control__auto_bill_orders=True)
    results = []
    for organization in organizations:
        try:
            mass_order_billing.mass_order_billing_service(user_id=str(system_user.id), task_id=self.request.id)
            results.append(f"Automated Mass Billing Task for {organization.name} was successful.")
        except ObjectDoesNotExist as exc:
            raise self.retry(exc=exc) from exc
    if results:
        return "\n".join(results)
    return "No organizations have auto billing enabled."


