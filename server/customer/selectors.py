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
import uuid

from billing.models import BillingHistory
from customer.types import (CustomerDiffResponse, CustomerMileageResponse,
                            CustomerOnTimePerfResponse, OrderDiffResponse)
from django.db.models import Case, F, FloatField, Max, Q, When
from django.db.models.aggregates import Sum
from django.utils import timezone
from order.models import Order
from stops.models import Stop
from utils.models import StatusChoices


def get_customer_order_diff(*, customer_id: uuid.UUID) -> OrderDiffResponse:
    """Calculate and return the total number of orders made and the percentage difference
    in counts for a customer between the current month, the previous month and the month
    before the previous month.

    This function first counts the total number of orders a customer has placed in the
    current month, the previous month, and the month before the previous month.

    It then calculates the percentage difference in order counts:
    - Percent change between the current month and the previous month
    - Percent change between the previous month and the month before the previous month

    Args:
        customer_id (uuid.UUID): The ID of the customer.

    Returns:
        OrderDiffResponse: A dictionary with the following structure:
            {
              "total_orders": int,
              "last_month_diff": float,
              "month_before_last_diff": float,
            }

    Note:
        - The "last_month_diff" and "month_before_last_diff" percentages are calculated with
          the order count of last month and month before last month as the base respectively.
          If there were no orders in the base month, the percentage difference is considered 0.

    Example:
        >>> get_customer_order_diff(customer_id=uuid.UUID("123"))
        >>> {
          >>> "total_orders": 50,
          >>> "last_month_diff": 25.0,
          >>> "month_before_last_diff": 33.33,
        >>> }

        The response signifies that the customer 123 has placed 50 orders in the current month,
        which is a 25% increase compared to the previous month and there was a 33.33% increase
        in the number of orders in the previous month compared to the month before the previous
        month.
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
    last_month_diff = (
        (this_month_orders_count - last_month_orders_count)
        / last_month_orders_count
        * 100
        if last_month_orders_count > 0
        else 0
    )

    month_before_last_diff = (
        (last_month_orders_count - month_before_last_orders_count)
        / month_before_last_orders_count
        * 100
        if month_before_last_orders_count > 0
        else 0
    )

    return {
        "total_orders": this_month_orders_count,
        "last_month_diff": round(last_month_diff, 1),
        "month_before_last_diff": round(month_before_last_diff, 1),
    }


def get_customer_revenue_diff(*, customer_id: uuid.UUID) -> CustomerDiffResponse:
    """Calculate and return the total revenue and its percentage difference a customer has generated
    between the current month, the previous month and the month before the previous month.

    This function first sums up the total revenue a customer has generated in the current, previous,
    and the month before the previous month.

    It then calculates the percentage difference between the current month's revenue and the previous
    month's revenue as well as between the previous month's revenue and the revenue of the month before
    previous month.

    Args:
        customer_id (uuid.UUID): The ID of the customer.

    Returns:
        CustomerDiffResponse: A dictionary with the following structure:
            {
              "total_revenue": float,
              "last_month_diff": float,
              "month_before_last_diff": float,
            }

    Note:
        - The "last_month_diff" and "month_before_last_diff" percentages are calculated with
          the revenue of last month and month before last month as the base respectively. If
          there was no revenue in the base month, the percentage difference is considered 0.

    Example:
        >>> get_customer_revenue_diff(customer_id=uuid.UUID("123"))
        >>> {
          >>> "total_revenue": 5000.00,
          >>> "last_month_diff": 25.0,
          >>> "month_before_last_diff": 33.33,
        >>> }

        The response signifies that customer 123 has generated 5000.00 in revenue in the current month,
        which is a 25% increase compared to the previous month and there was a 33.33% increase in the
        revenue in the previous month compared to the month before previous month.
    """
    now = timezone.now()
    this_month = now.month
    this_year = now.year
    last_month = this_month - 1 if this_month != 1 else 12
    last_month_year = this_year if this_month != 1 else this_year - 1
    month_before_last = last_month - 1 if last_month != 1 else 12
    month_before_last_year = last_month_year if last_month != 1 else last_month_year - 1

    this_month_revenue = (
        BillingHistory.objects.filter(
            customer_id=customer_id, created__month=this_month, created__year=this_year
        ).aggregate(total=Sum("total_amount"))["total"]
        or 0
    )

    last_month_revenue = (
        BillingHistory.objects.filter(
            customer_id=customer_id,
            created__month=last_month,
            created__year=last_month_year,
        ).aggregate(total=Sum("total_amount"))["total"]
        or 0
    )

    month_before_last_revenue = (
        BillingHistory.objects.filter(
            customer_id=customer_id,
            created__month=month_before_last,
            created__year=month_before_last_year,
        ).aggregate(total=Sum("total_amount"))["total"]
        or 0
    )

    last_month_diff = (
        (this_month_revenue - last_month_revenue) / last_month_revenue * 100
        if last_month_revenue
        else 0
    )
    month_before_last_diff = (
        (last_month_revenue - month_before_last_revenue)
        / month_before_last_revenue
        * 100
        if month_before_last_revenue
        else 0
    )

    return {
        "total_revenue": this_month_revenue,
        "last_month_diff": round(last_month_diff, 1),
        "month_before_last_diff": month_before_last_diff,
    }


def get_customer_on_time_performance_diff(
    *, customer_id: uuid.UUID
) -> CustomerOnTimePerfResponse:
    """Calculate and return the on-time performance metrics difference for a customer between
    the current month and the previous month. The function considers all stops the customer made.

    This function first determines the total number of stops the customer made in the given months.
    These stops are  then categorized into three categories namely: On-time, Early and Late stops
    based on appointment time windows.

    - Early if the arrival time was before the start of the appointment time window.
    - Late if the arrival time was after the end of the appointment time window.
    - On-time if the arrival time was within the appointment time window.

    The function finally calculates the monthly percentage for each stop type (on-time, early, late)
    for both months. It then calculates the difference in percentages between the current month and
    the previous month for each stop type.

    Args:
        customer_id (uuid.UUID): The ID of the customer.

    Returns:
        CustomerOnTimePerfResponse: A dictionary containing the structure as follows:
            {
              "this_month_on_time_percentage": float,
              "last_month_on_time_percentage": float,
              "on_time_diff": float,
              "this_month_early_percentage": float,
              "last_month_early_percentage": float,
              "early_diff": float,
              "this_month_late_percentage": float,
              "last_month_late_percentage": float,
              "late_diff": float,
            }

    Note:
        - The function does not handle corner cases like when the total stops is 0. In such cases, the
        percentages will be 0.
        - The function also does not handle cases where the last month does not have any on-time, early
        or late stops. In such cases, the difference will be 0.

    Example:
        >>> get_customer_on_time_performance_diff(customer_id=uuid.UUID("123"))
        >>> {
          >>> "this_month_on_time_percentage": 75.0,
          >>> "last_month_on_time_percentage": 80.0,
          >>> "on_time_diff": -6.25,
          >>> "this_month_early_percentage": 15.0,
          >>> "last_month_early_percentage": 10.0,
          >>> "early_diff": 50.0,
          >>> "this_month_late_percentage": 10.0,
          >>> "last_month_late_percentage": 10.0,
          >>> "late_diff": 0.0,
        >>> }

        The response signifies that customer 123 had an on-time performance of 75% in the current month,
        down 6.25% from 80% in the previous month.
    """

    now = timezone.now()
    this_month = now.month
    this_year = now.year
    last_month = this_month - 1 if this_month != 1 else 12
    last_month_year = this_year if this_month != 1 else this_year - 1

    months = [
        (this_year, this_month),
        (last_month_year, last_month),
    ]

    stop_percentages = {}

    for year, month in months:
        customer_stops = Stop.objects.filter(
            movement__order__customer_id=customer_id,
            movement__order__status__in=[StatusChoices.COMPLETED, StatusChoices.BILLED],
            arrival_time__year=year,
            arrival_time__month=month,
        )

        total_stops = customer_stops.count()

        if total_stops == 0:
            stop_percentages[month] = {
                "on_time_percentage": 0,
                "early_percentage": 0,
                "late_percentage": 0,
            }
            continue

        on_time_stops = customer_stops.filter(
            Q(arrival_time__gte=F("appointment_time_window_start"))
            & Q(arrival_time__lte=F("appointment_time_window_end"))
        ).count()

        early_stops = customer_stops.filter(
            arrival_time__lt=F("appointment_time_window_start")
        ).count()

        late_stops = customer_stops.filter(
            arrival_time__gt=F("appointment_time_window_end")
        ).count()

        stop_percentages[month] = {
            "on_time_percentage": on_time_stops / total_stops * 100,
            "early_percentage": early_stops / total_stops * 100,
            "late_percentage": late_stops / total_stops * 100,
        }

    this_month_data = stop_percentages[this_month]
    last_month_data = stop_percentages[last_month]

    return {
        "this_month_on_time_percentage": this_month_data["on_time_percentage"],
        "last_month_on_time_percentage": last_month_data["on_time_percentage"],
        "on_time_diff": (
            this_month_data["on_time_percentage"]
            - last_month_data["on_time_percentage"]
        )
        / last_month_data["on_time_percentage"]
        * 100
        if last_month_data["on_time_percentage"]
        else 0,
        "this_month_early_percentage": this_month_data["early_percentage"],
        "last_month_early_percentage": last_month_data["early_percentage"],
        "early_diff": (
            this_month_data["early_percentage"] - last_month_data["early_percentage"]
        )
        / last_month_data["early_percentage"]
        * 100
        if last_month_data["early_percentage"]
        else 0,
        "this_month_late_percentage": this_month_data["late_percentage"],
        "last_month_late_percentage": last_month_data["late_percentage"],
        "late_diff": (
            this_month_data["late_percentage"] - last_month_data["late_percentage"]
        )
        / last_month_data["late_percentage"]
        * 100
        if last_month_data["late_percentage"]
        else 0,
    }


def calculate_customer_total_miles(
    *, customer_id: uuid.UUID
) -> CustomerMileageResponse:
    """Calculate and return the total mileage and its percentage difference a customer has covered
    between the current month and the previous month.

    This function first sums up the total miles a customer has covered in their completed or billed
    orders in the current and the previous month.

    It then calculates the percentage difference in total miles covered between the current month
    and the previous month.

    Args:
        customer_id (uuid.UUID): The ID of the customer.

    Returns:
        CustomerMileageResponse: A dictionary with the following structure:
            {
              "this_month_miles": float,
              "last_month_miles": float,
              "mileage_diff": float,
            }

    Note:
        - The "mileage_diff" percentage is calculated taking the mileage of the previous month as the base.
          If there is no mileage covered in the previous month, the percentage difference is considered 0.

    Example:
        >>> calculate_customer_total_miles(customer_id=uuid.UUID("123"))
        >>> {
          >>> "this_month_miles": 1500.0,
          >>> "last_month_miles": 1200.0,
          >>> "mileage_diff": 25.0,
        >>> }

        The response signifies that the customer 123 has covered 1500.0 miles in the current month,
        which is a 25% increase compared to the 1200.0 miles covered in the previous month.
    """
    now = timezone.now()
    this_month = now.month
    this_year = now.year
    last_month = this_month - 1 if this_month != 1 else 12
    last_month_year = this_year if this_month != 1 else this_year - 1

    aggregated_miles = Order.objects.filter(
        customer_id=customer_id,
        status__in=[StatusChoices.COMPLETED, StatusChoices.BILLED],
    ).aggregate(
        current_month_miles=Sum(
            Case(
                When(
                    created__month=this_month,
                    created__year=this_year,
                    then=F("mileage"),
                ),
                default=0,
                output_field=FloatField(),
            ),
        ),
        previous_month_miles=Sum(
            Case(
                When(
                    created__month=last_month,
                    created__year=last_month_year,
                    then=F("mileage"),
                ),
                default=0,
                output_field=FloatField(),
            ),
        ),
    )

    this_month_miles = aggregated_miles["current_month_miles"] or 0
    last_month_miles = aggregated_miles["previous_month_miles"] or 0

    # Avoid divide by zero error
    if last_month_miles:
        mileage_diff = ((this_month_miles - last_month_miles) / last_month_miles) * 100
    else:
        mileage_diff = 0

    return {
        "this_month_miles": this_month_miles,
        "last_month_miles": last_month_miles,
        "mileage_diff": round(mileage_diff, 1),
    }


def get_customer_shipment_metrics(*, customer_id: uuid.UUID) -> dict:
    aggregated_dates = Order.objects.filter(
        customer_id=customer_id,
    ).aggregate(
        last_bill_date=Max("bill_date"),
        last_shipment_date=Max("ship_date"),
    )

    last_bill_date = aggregated_dates["last_bill_date"]
    last_shipment_date = aggregated_dates["last_shipment_date"]

    if last_bill_date:
        last_bill_date = last_bill_date.strftime("%b %d, %Y")
    if last_shipment_date:
        last_shipment_date = last_shipment_date.strftime("%b %d, %Y")

    return {
        "last_bill_date": last_bill_date,
        "last_shipment_date": last_shipment_date,
    }


def get_customer_credit_balance(*, customer_id: uuid.UUID) -> float:
    # TODO(Wolfred) Actually write validation using collections module to get total credit balance
    # Or add a status field to invoice to show amount due.
    credit_balance = BillingHistory.objects.filter(
        customer_id=customer_id,
    ).aggregate(
        credit_balance=Sum("total_amount"),
    )["credit_balance"]

    return credit_balance or 0
