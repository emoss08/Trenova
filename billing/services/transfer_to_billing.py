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

from datetime import datetime
from typing import List, Optional
from collections.abc import Iterable

from django.core.exceptions import ValidationError
from django.db import IntegrityError
from django.shortcuts import get_object_or_404
from django.utils import timezone

from accounts.models import User
from billing import models
from billing.exceptions import BillingException
from billing.selectors import get_billable_orders
from billing.services.billing_service import create_billing_exception
from order.models import Order
from utils.types import MODEL_UUID


def transfer_to_billing_queue_service(
    *, user_id: MODEL_UUID, order_pros: list[str], task_id: str
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

    user: User = get_object_or_404(User, id=user_id)
    orders: Iterable[Order] | None = get_billable_orders(
        organization=user.organization, order_pros=order_pros
    )

    if not orders:
        # Raise an exception if no orders are found to be eligible for transfer. This also will cause the task to fail.
        raise BillingException("No orders found to be eligible for transfer.")

    now: datetime = timezone.now()
    transfer_log = []
    for order in orders:
        try:
            models.BillingQueue.objects.create(
                organization=order.organization,
                order=order,
            )
            order.transferred_to_billing = True
            order.billing_transfer_date = now

            transfer_log.append(
                models.BillingTransferLog(
                    order=order,
                    organization=order.organization,
                    transferred_at=now,
                    task_id=task_id,
                    transferred_by=user,
                )
            )

            Order.objects.bulk_update(
                orders, ["transferred_to_billing", "billing_transfer_date"]
            )
        except (ValidationError, IntegrityError) as validation_error:
            create_billing_exception(
                user=user,
                exception_type="OTHER",
                order=order,
                exception_message=f"Order {order.pro_number} failed to transfer to billing queue: {validation_error}",
            )
            return f"Order {order.pro_number} failed to transfer to billing queue: {validation_error}"

    models.BillingTransferLog.objects.bulk_create(transfer_log)
    return "Successfully transferred orders to billing queue."
