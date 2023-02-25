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
from django.core.exceptions import ValidationError
from django.db import transaction
from django.shortcuts import get_object_or_404
from django.utils import timezone

from accounts.models import User
from billing import models
from billing.selectors import get_transfer_to_billing_orders
from billing.services.billing_service import create_billing_exception
from order.models import Order


def transfer_to_billing_queue_service(
    *, user_id: str, order_pros: list[str], task_id: str
) -> str:
    """
    Creates a new BillingQueue object for each order and updates the order's transfer status and transfer date.

    Args:
        user_id: A string representing the ID of the user who initiated the transfer.
        order_pros: A list of strings representing the order IDs to transfer.
        task_id: A string representing the ID of the Celery task that initiated the transfer.

    Returns:
        A string representing the result of the transfer operation.

    Raises:
        N/A

    This function is responsible for creating a new BillingQueue object for each order in the provided list of
    order IDs, and updating the transfer status and transfer date of each order. It is typically called as a Celery
    task, and is intended to run in the background.

    If a provided order ID is invalid, this function will log an error message and continue processing the
    remaining orders. If all orders are processed successfully, the function will return None.

    The function expects the following arguments:
    - user_id: A string representing the ID of the user who initiated the transfer.
    - order_pros: A list of strings representing the order IDs to transfer.

    The function retrieves the corresponding User object based on the provided user ID, and retrieves the Order
    objects based on the provided order IDs using the `get_transfer_to_billing_orders` helper function.

    For each order, the function attempts to create a new BillingQueue object with the order's organization and ID.
    If this operation fails due to a ValidationError, the function logs an error message and continues processing
    the remaining orders.

    If the BillingQueue object is created successfully, the function updates the `transferred_to_billing` and
    `billing_transfer_date` fields of the Order object to indicate that the order has been successfully transferred
    to billing.

    Finally, the function uses bulk_create and bulk_update operations to efficiently create the BillingTransferLog
    objects for all successfully transferred orders and update the Order objects with the new transfer status and
    transfer date.
    """
    user = get_object_or_404(User, id=user_id)
    orders = get_transfer_to_billing_orders(order_pros=order_pros)

    now = timezone.now()
    transfer_log = []
    result = "Task completed successfully. See billing transfer log for details."

    with transaction.atomic():
        for order in orders:
            try:
                models.BillingQueue.objects.create(
                    organization=order.organization,
                    order=order,
                )

            except ValidationError as db_error:
                create_billing_exception(
                    user=user,
                    exception_type="OTHER",
                    order=order,
                    exception_message=f"Order {order.pro_number} failed to transfer to billing queue: {db_error}",
                )

                transfer_log.append(
                    models.BillingTransferLog(
                        order=order,
                        organization=order.organization,
                        transferred_at=now,
                        task_id=task_id,
                        transferred_by=user,
                    )
                )

                order.transferred_to_billing = True
                order.billing_transfer_date = now

        models.BillingTransferLog.objects.bulk_create(transfer_log)
        Order.objects.bulk_update(
            orders, ["transferred_to_billing", "billing_transfer_date"]
        )

    return result
