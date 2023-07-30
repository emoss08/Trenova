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
from django.db.models import Q, F
from django.utils import timezone
from billing.models import BillingHistory
from customer.types import (
    CustomerOnTimePerfResponse,
    CustomerDiffResponse,
    OrderDiffResponse,
    CustomerMileageResponse,
)
from order.models import Order
from django.db.models.aggregates import Sum

from stops.models import Stop
from utils.models import StatusChoices


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

    this_month_orders_count = Order.objects.filter(
        customer_id=customer_id, created__month=this_month, created__year=this_year
    ).count()

    last_month_orders_count = Order.objects.filter(
        customer_id=customer_id,
        created__month=last_month,
        created__year=last_month_year,
    ).count()

    month_before_last_orders_count = Order.objects.filter(
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


def get_customer_revenue_diff(*, customer_id: str) -> CustomerDiffResponse:
    """Calculates the current month's revenue difference, and the percentage difference from the
    last month and the month before for a customer based on the given customer id.

    The function works by extracting the revenue for current month, last month and the month
    before from BillingHistory and calculates the difference.

    Args:
        customer_id (str): The unique identifier of a customer.

    Returns:
        CustomerDiffResponse:
            this_month_revenue (float): The revenue for the current month.
            last_month_diff (float): The percentage difference in revenue from the last month.
            month_before_last_diff (float): The percentage difference in revenue from the month before last.
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

    return {
        "total_revenue": this_month_revenue,
        "last_month_diff": last_month_diff,
        "month_before_last_diff": month_before_last_diff,
    }


def get_customer_on_time_performance_diff(
    *, customer_id: str
) -> CustomerOnTimePerfResponse:
    now = timezone.now()
    this_month = now.month
    this_year = now.year
    last_month = this_month - 1 if this_month != 1 else 12
    last_month_year = this_year if this_month != 1 else this_year - 1

    def _calculate_stop_percentages(year, month):
        # Get the stops for orders delivered for the particular customer in the given month and year.
        customer_stops = Stop.objects.filter(
            movement__order__customer_id=customer_id,
            movement__order__status__in=[StatusChoices.COMPLETED, StatusChoices.BILLED],
            arrival_time__year=year,
            arrival_time__month=month,
        )

        # Get the total number of stops
        total_stops = customer_stops.count()

        if total_stops == 0:
            return 0, 0, 0

        # Get the number of stops that are on time
        on_time_stops = customer_stops.filter(
            Q(arrival_time__gte=F("appointment_time_window_start"))
            & Q(arrival_time__lte=F("appointment_time_window_end"))
        ).count()

        # Get the number of stops that are early
        early_stops = customer_stops.filter(
            arrival_time__lt=F("appointment_time_window_start")
        ).count()

        # Get the number of stops that are late
        late_stops = customer_stops.filter(
            arrival_time__gt=F("appointment_time_window_end")
        ).count()

        # Calculate the percentage of on time stops, early stops and late stops
        return (
            on_time_stops / total_stops * 100,
            early_stops / total_stops * 100,
            late_stops / total_stops * 100,
        )

    # Calculate the on-time, early and late percentage for this month and the last month
    (
        this_month_on_time_percentage,
        this_month_early_percentage,
        this_month_late_percentage,
    ) = _calculate_stop_percentages(this_year, this_month)

    (
        last_month_on_time_percentage,
        last_month_early_percentage,
        last_month_late_percentage,
    ) = _calculate_stop_percentages(last_month_year, last_month)

    # Calculate the difference in on-time, early and late performance
    on_time_diff = (
        (this_month_on_time_percentage - last_month_on_time_percentage)
        / last_month_on_time_percentage
        * 100
        if last_month_on_time_percentage
        else 0
    )

    early_diff = (
        (this_month_early_percentage - last_month_early_percentage)
        / last_month_early_percentage
        * 100
        if last_month_early_percentage
        else 0
    )

    late_diff = (
        (this_month_late_percentage - last_month_late_percentage)
        / last_month_late_percentage
        * 100
        if last_month_late_percentage
        else 0
    )

    return {
        "this_month_on_time_percentage": this_month_on_time_percentage,
        "last_month_on_time_percentage": last_month_on_time_percentage,
        "on_time_diff": on_time_diff,
        "this_month_early_percentage": this_month_early_percentage,
        "last_month_early_percentage": last_month_early_percentage,
        "early_diff": early_diff,
        "this_month_late_percentage": this_month_late_percentage,
        "last_month_late_percentage": last_month_late_percentage,
        "late_diff": late_diff,
    }


def calculate_customer_total_miles(*, customer_id: str) -> CustomerMileageResponse:
    """Calculates and returns a customer's total mileage for the current and previous month, and the difference
    in percentage.

    The function first determines the current month and year. It calculates the mileage for the customer's
    completed or billed orders for this month. This process is repeated to calculate the mileage for the
    last month as well.

    This function subsequently calculates the percentage difference in mileage of this month and last month.
    If no miles were logged last month, the function safely avoids a divide-by-zero error and sets the percentage difference as zero.

    Args:
        customer_id (str): The unique identifier for a customer.

    Returns:
        CustomerMileageResponse: A dictionary representing the customer's mileage data,
        including total mileage for this month and last month, and the percentage difference in mileage.
    """
    now = timezone.now()
    this_month = now.month
    this_year = now.year
    last_month = this_month - 1 if this_month != 1 else 12
    last_month_year = this_year if this_month != 1 else this_year - 1

    this_month_miles = (
        Order.objects.filter(
            customer_id=customer_id,
            status__in=[StatusChoices.COMPLETED, StatusChoices.BILLED],
            created__month=this_month,
            created__year=this_year,
        ).aggregate(total=Sum("mileage"))["total"]
        or 0
    )

    last_month_miles = (
        Order.objects.filter(
            customer_id=customer_id,
            status__in=[StatusChoices.COMPLETED, StatusChoices.BILLED],
            created__month=last_month,
            created__year=last_month_year,
        ).aggregate(total=Sum("mileage"))["total"]
        or 0
    )

    # Avoid divide by zero error
    if last_month_miles:
        mileage_diff = ((this_month_miles - last_month_miles) / last_month_miles) * 100
    else:
        mileage_diff = 0

    return {
        "this_month_miles": this_month_miles,
        "last_month_miles": last_month_miles,
        "mileage_diff": mileage_diff,
    }
