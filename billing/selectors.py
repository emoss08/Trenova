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

from billing.models import BillingControl, BillingQueue
from order.models import Order
from organization.models import Organization
from utils.models import StatusChoices


def get_billable_orders(*, organization: Organization) -> Iterable[Order] | None:
    """Returns an iterator of orders that are billable for a given organization.

    The billable orders are determined based on the `order_transfer_criteria`
    set on the organization's `BillingControl` instance. If the
    `order_transfer_criteria` is set to `READY_AND_COMPLETED`, orders that are
    both ready to bill and have a status of `COMPLETED` will be returned. If the
    `order_transfer_criteria` is set to `COMPLETED`, orders that have a status of
    `COMPLETED` will be returned. If the `order_transfer_criteria` is set to
    `READY_TO_BILL`, orders that are ready to bill will be returned.

    Args:
        organization: The organization for which billable orders should be returned.

    Returns:
        An iterator of billable orders for the organization, or `None` if no billable
        orders are found.
    """

    if (
        organization.billing_control.order_transfer_criteria
        == BillingControl.OrderTransferCriteriaChoices.READY_AND_COMPLETED
    ):
        return organization.orders.filter(
            billed=False,
            status=StatusChoices.COMPLETED,
            ready_to_bill=True,
            transferred_to_billing=False,
            billing_transfer_date__isnull=True,
        )
    elif (
        organization.billing_control.order_transfer_criteria
        == BillingControl.OrderTransferCriteriaChoices.COMPLETED
    ):
        return organization.orders.filter(
            billed=False,
            status=StatusChoices.COMPLETED,
            transferred_to_billing=False,
            billing_transfer_date__isnull=True,
        )
    elif (
        organization.billing_control.order_transfer_criteria
        == BillingControl.OrderTransferCriteriaChoices.READY_TO_BILL
    ):
        return organization.orders.filter(
            billed=False,
            ready_to_bill=True,
            transferred_to_billing=False,
            billing_transfer_date__isnull=True,
        )
    return None


def get_billing_queue_information(*, order: Order) -> BillingQueue | None:
    """Returns the billing history for a given order.

    Args:
        order: The order for which the billing history should be returned.

    Returns:
        The billing history for the order, or `None` if no billing history is found.
    """
    return BillingQueue.objects.filter(order=order).last()
