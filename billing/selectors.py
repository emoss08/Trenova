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

from collections.abc import Iterable
from typing import Union, List, Optional

from django.db.models import Q

from billing.models import BillingControl, BillingQueue
from order.models import Order
from organization.models import Organization
from utils.models import StatusChoices


def get_billable_orders(
    *, organization: Organization, order_pros: Optional[List[str]] = None
) -> Union[Iterable[Order], None]:
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
        order_pros: A list of order pros to filter the billable orders by.

    Returns:
        An iterator of billable orders for the organization, or `None` if no billable
        orders are found.
    """

    # Map BillingControl.OrderTransferCriteriaChoices to the corresponding query
    criteria_to_query = {
        BillingControl.OrderTransferCriteriaChoices.READY_AND_COMPLETED: Q(
            status=StatusChoices.COMPLETED
        )
        & Q(ready_to_bill=True),
        BillingControl.OrderTransferCriteriaChoices.COMPLETED: Q(
            status=StatusChoices.COMPLETED
        ),
        BillingControl.OrderTransferCriteriaChoices.READY_TO_BILL: Q(
            ready_to_bill=True
        ),
    }

    query: Q = (
        Q(billed=False)
        & Q(transferred_to_billing=False)
        & Q(billing_transfer_date__isnull=True)
    )
    order_criteria_query: Q | None = criteria_to_query.get(
        organization.billing_control.order_transfer_criteria
    )

    if order_criteria_query is not None:
        query &= order_criteria_query

    if order_pros:
        query &= Q(pro_number__in=order_pros)

    orders = organization.orders.filter(query)

    return orders if orders.exists() else None


def get_billing_queue_information(*, order: Order) -> Union[BillingQueue, None]:
    """Returns the billing history for a given order.

    Args:
        order: The order for which the billing history should be returned.

    Returns:
        The billing history for the order, or `None` if no billing history is found.
    """
    return BillingQueue.objects.filter(order=order).last()
