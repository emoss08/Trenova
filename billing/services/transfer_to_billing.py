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

from typing import Iterable

from django.db import IntegrityError
from django.shortcuts import get_object_or_404
from django.utils import timezone
from notifications.signals import notify

from accounts.models import User
from billing import models
from billing.selectors import get_billable_orders
from billing.services.billing_service import create_billing_exception
from order.models import Order
from billing.exceptions import BillingException

def transfer_to_billing_queue(user_id: str) -> None:
    """
    Transfer billable orders to the billing queue for the current organization.

    This function retrieves the billable orders for the current organization and transfers them
    to the billing queue. The transferred orders are marked as `transferred_to_billing` and their
    `billing_transfer_date` is set to the current date and time.

    Returns:
        None: This function returns `None`.
    """

    user: User = get_object_or_404(User, id=user_id)

    billable_orders: Iterable[Order] | None = get_billable_orders(
        organization=user.organization
    )

    if not billable_orders:
        notify.send(
            user,
            recipient=user,
            level="info",
            verb="Billing Transfer Exception",
            description="No Billable Orders were found for the current organization.",
        )
        return

    order_ids = [order.id for order in billable_orders]
    now = timezone.now()

    Order.objects.filter(id__in=order_ids).update(
        transferred_to_billing=True, billing_transfer_date=now
    )

    try:
        for order in order_ids:
            models.BillingQueue.objects.create(
                organization=user.organization, order_id=order
            )

        bill_transfer_logs = [
            models.BillingTransferLog(
                organization=user.organization,
                order_id=order_id,
                transferred_at=now,
                transferred_by=user,
            )
            for order_id in order_ids
        ]
        models.BillingTransferLog.objects.bulk_create(bill_transfer_logs)

    except IntegrityError as int_err:
        create_billing_exception(
            user=user,
            exception_type="OTHER",
            order=None,
            exception_message=f"Error transferring orders to billing queue: {int_err}",
        )
    except BillingException as bill_err:
        create_billing_exception(
            user=user,
            exception_type="OTHER",
            order=None,
            exception_message=f"Error transferring orders to billing queue: {bill_err}",
        )
