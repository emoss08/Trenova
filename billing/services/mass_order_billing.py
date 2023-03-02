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
from django.db import transaction
from django.shortcuts import get_object_or_404

from accounts.models import User
from billing.services.billing_service import (
    check_billing_control,
    check_billing_requirements,
    create_billing_exception,
    create_billing_history,
    delete_billing_queue,
    get_billing_queue,
    send_billing_email,
    set_billing_requirements,
    set_order_billed,
    set_order_documents,
)
from utils.types import MODEL_UUID


def order_billing_actions(*, invoice, user: User) -> None:
    """Perform billing actions for a single order.

    This function performs the billing actions for a single order, including
    setting the order as billed, creating a billing history, and deleting
    the invoice from the billing queue.

    Args:
        user (User): The user performing the billing actions.
        invoice (models.BillingQueue): The invoice to perform the billing actions for.

    Returns:
        None: None
    """
    with transaction.atomic():
        set_order_billed(order=invoice.order)
        create_billing_history(order=invoice.order, user=user)
        delete_billing_queue(billing_queue=invoice)


def mass_order_billing_service(*, user_id: MODEL_UUID, task_id: str) -> None:
    """Bill a list of orders to their respective customers

    This function bills a list of orders to their respective customers.
    For each order in the billing queue, the function first checks if
    the billing control is enabled. If it is, it sets the billing requirements
    for the customer and order, sets the order document IDs, and checks if the
    billing requirements are met. If the requirements are met, it sets the order
    as billed, sends a billing email to the customer, and creates a billing history
    record. If the requirements are not met, a billing exception is created. If the
    billing control is not enabled, the order is simply set as billed. Finally, the
    invoice is deleted from the billing queue.

    Args:
        user_id (str): The user performing the billing actions.
        task_id (str): The task ID of the billing task.

    Returns:
        None: None
    """

    user: User = get_object_or_404(User, id=user_id)

    for invoice in get_billing_queue(user=user, task_id=task_id):
        if check_billing_control(user=user):
            set_billing_requirements(customer=invoice.order.customer)
            set_order_documents(order=invoice.order)
            if check_billing_requirements(order=invoice.order, user=user):
                order_billing_actions(user=user, invoice=invoice)
                send_billing_email(user=user, order=invoice.order)
            else:
                create_billing_exception(
                    user=user,
                    exception_type="PAPERWORK",
                    order=invoice.order,
                    exception_message=f"Paperwork requirements not met for order {invoice.order.id}.",
                )
        else:
            order_billing_actions(user=user, invoice=invoice)
            send_billing_email(order=invoice.order, user=user)
