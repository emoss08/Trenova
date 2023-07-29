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

from typing import TYPE_CHECKING

from django.db.models.aggregates import Sum
from django.utils import timezone

from billing.models import BillingHistory
from movements.models import Movement
from order import models
from order.types import OrderDiffResponse
from stops.models import Stop

if TYPE_CHECKING:
    from django.db.models import QuerySet
    from utils.types import ModelUUID


def get_order_by_id(*, order_id: "ModelUUID") -> models.Order | None:
    """Get an Order model instance by its ID.

    Args:
        order_id (str): The ID of the order.

    Returns:
        models.Order: The Order model instance.
    """
    try:
        return models.Order.objects.get(pk__exact=order_id)
    except models.Order.DoesNotExist:
        return None


def get_order_movements(*, order: models.Order) -> "QuerySet[Movement]":
    """Get the movements of an order.

    Args:
        order (models.Order): The order.

    Returns:
        QuerySet[Movement]: QuerySet of the movements of the order.
    """
    return Movement.objects.filter(order=order)


def get_order_stops(*, order: models.Order) -> "QuerySet[Stop]":
    """Get the stops of an order.

    Args:
        order (models.Order): The order.

    Returns:
        QuerySett[Stop]: QuerySet of the stops of the order.
    """
    movements = get_order_movements(order=order)
    return Stop.objects.filter(movement__in=movements).select_related("movement")


def sum_order_additional_charges(*, order: models.Order) -> float:
    """Sum the additional charges of an order.

    Args:
        order (models.Order): The order.

    Returns:
        float: The sum of the additional charges.
    """
    # Calculate the sum of sub_total for each additional charge associated with the order
    additional_charges_total = models.AdditionalCharge.objects.filter(
        order=order
    ).aggregate(total=Sum("sub_total"))["total"]

    # If there are no additional charges associated with the order, return 0
    return additional_charges_total or 0


def get_customer_order_diff(*, customer_id: str) -> OrderDiffResponse:
    """Calculates and returns the total order count for a customer for the current month,
    the percentage difference in order count from the last month, and the month before last.

    Function takes a customer's ID, filters out the orders placed by the customer in each month,
    and calculates the percentage difference between the order counts of these months.

    Args:
        customer_id (str): A unique identifier of a customer, used to filter the orders for the specific customer.

    Returns:
        OrderDiffResponse:
            total_orders (int): The total order count for the current month.
            last_month_diff (int): The percentage difference in order count from the last month.
            month_before_last_diff (int): The percentage difference in order count from the month before last.
    """
    now = timezone.now()
    this_month = now.month
    this_year = now.year
    last_month = this_month - 1 if this_month != 1 else 12
    last_month_year = this_year if this_month != 1 else this_year - 1
    month_before_last = last_month - 1 if last_month != 1 else 12
    month_before_last_year = last_month_year if last_month != 1 else last_month_year - 1

    this_month_orders_count = models.Order.objects.filter(
        customer_id=customer_id, created__month=this_month, created__year=this_year
    ).count()

    last_month_orders_count = models.Order.objects.filter(
        customer_id=customer_id,
        created__month=last_month,
        created__year=last_month_year,
    ).count()

    month_before_last_orders_count = models.Order.objects.filter(
        customer_id=customer_id,
        created__month=month_before_last,
        created__year=month_before_last_year,
    ).count()

    # Calculate differences
    if last_month_orders_count > 0:
        last_month_diff = (
            (this_month_orders_count - last_month_orders_count)
            / last_month_orders_count
            * 100
        )
    else:
        last_month_diff = 0

    if month_before_last_orders_count > 0:
        month_before_last_diff = (
            (last_month_orders_count - month_before_last_orders_count)
            / month_before_last_orders_count
            * 100
        )
    else:
        month_before_last_diff = 0

    return {
        "total_orders": this_month_orders_count,
        "last_month_diff": last_month_diff,
        "month_before_last_diff": month_before_last_diff,
    }


def get_customer_revenue_diff(*, customer_id: str) -> tuple[float, float, float]:
    """Calculates the current month's revenue difference, and the percentage difference from the
    last month and the month before for a customer based on the given customer id.

    The function works by extracting the revenue for current month, last month and the month
    before from BillingHistory and calculates the difference.

    Args:
        customer_id (str): The unique identifier of a customer.

    Returns:
        tuple: Returns a tuple with three elements:
               1. `float`: The revenue difference of the customer for the current month.
               2. `float`: The percentage difference in revenue from the last month.
               3. `float`: The percentage difference in revenue from the month before last.

    """
    now = timezone.now()
    this_month = now.month
    this_year = now.year
    last_month = this_month - 1 if this_month != 1 else 12
    last_month_year = this_year if this_month != 1 else this_year - 1
    month_before_last = last_month - 1 if last_month != 1 else 12
    month_before_last_year = last_month_year if last_month != 1 else last_month_year - 1

    this_month_revenue = BillingHistory.objects.filter(
        customer_id=customer_id, created__month=this_month, created__year=this_year
    ).aggregate(total=Sum("total_amount"))["total"]

    last_month_revenue = BillingHistory.objects.filter(
        customer_id=customer_id,
        created__month=last_month,
        created__year=last_month_year,
    ).aggregate(total=Sum("total_amount"))["total"]

    month_before_last_revenue = BillingHistory.objects.filter(
        customer_id=customer_id,
        created__month=month_before_last,
        created__year=month_before_last_year,
    ).aggregate(total=Sum("total_amount"))["total"]

    # Calculate differences
    if last_month_revenue:
        last_month_diff = (
            (this_month_revenue - last_month_revenue) / last_month_revenue * 100
        )
    else:
        last_month_diff = 0

    if month_before_last_revenue:
        month_before_last_diff = (
            (last_month_revenue - month_before_last_revenue)
            / month_before_last_revenue
            * 100
        )
    else:
        month_before_last_diff = 0

    return (
        this_month_revenue,
        last_month_diff,
        month_before_last_diff,
    )
