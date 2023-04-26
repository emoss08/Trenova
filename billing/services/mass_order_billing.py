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
from django.db import transaction
from django.shortcuts import get_object_or_404

from accounts.models import User
from billing.models import BillingQueue
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
from utils.types import ModelUUID


def order_billing_actions(*, invoice: BillingQueue, user: User) -> None:
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


def mass_order_billing_service(*, user_id: ModelUUID, task_id: str) -> None:
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
