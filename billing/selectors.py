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

from collections.abc import Iterable

from django.db.models import Prefetch, Q

from billing.models import BillingControl, BillingQueue
from commodities.models import Commodity
from customer.models import Customer
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

    query = (
        Q(billed=False)
        & Q(transferred_to_billing=False)
        & Q(billing_transfer_date__isnull=True)
    )
    order_criteria_query = criteria_to_query.get(
        organization.billing_control.order_transfer_criteria
    )
    if order_criteria_query is not None:
        query &= order_criteria_query

    return organization.orders.filter(query) or None


def get_billing_queue_information(*, order: Order) -> BillingQueue | None:
    """Returns the billing history for a given order.

    Args:
        order: The order for which the billing history should be returned.

    Returns:
        The billing history for the order, or `None` if no billing history is found.
    """
    return BillingQueue.objects.filter(order=order).last()


def get_transfer_to_billing_orders(*, order_pros: list[str]) -> Iterable[Order]:
    return (
        Order.objects.filter(pro_number__in=order_pros)
        .only(
            "id",
            "organization_id",
            "organization__name",
            "pro_number",
            "order_type_id",
            "pieces",
            "weight",
            "ready_to_bill",
            "status",
            "billed",
            "mileage",
            "consignee_ref_number",
            "other_charge_amount",
            "other_charge_amount_currency",
            "freight_charge_amount",
            "freight_charge_amount_currency",
            "sub_total",
            "sub_total_currency",
            "bol_number",
            "transferred_to_billing",
            "billing_transfer_date",
            "revenue_code",
            "customer_id",
            "commodity_id",
            "entered_by",
        )
        .select_related("revenue_code")
        .prefetch_related(
            Prefetch("customer", queryset=Customer.objects.only("id", "name")),
            Prefetch(
                "commodity",
                queryset=Commodity.objects.only("id", "description"),
            ),
            Prefetch(
                "organization",
                queryset=Organization.objects.only("id", "name"),
            ),
        )
    )
