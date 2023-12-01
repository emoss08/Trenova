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

from django.db.models import Case, F, FloatField, Max, Q, When
from django.db.models.aggregates import Sum
from django.shortcuts import get_object_or_404
from django.utils import timezone

from billing.models import BillingHistory
from customer import models, types
from shipment.models import Shipment
from stops.models import Stop
from utils.models import StatusChoices


def get_customer_shipments_diff(
    *, customer_id: uuid.UUID
) -> types.shipmentDiffResponse:
    """Calculate and return the total number of shipments made and the percentage difference
    in counts for a customer between the current month, the previous month and the month
    before the previous month.

    This function first counts the total number of shipments a customer has placed in the
    current month, the previous month, and the month before the previous month.

    It then calculates the percentage difference in shipment counts:
    - Percent change between the current month and the previous month
    - Percent change between the previous month and the month before the previous month

    Args:
        customer_id (uuid.UUID): The ID of the customer.

    Returns:
        shipmentDiffResponse: A dictionary with the following structure:
            {
              "total_shipments": int,
              "last_month_diff": float,
              "month_before_last_diff": float,
            }

    Note:
        - The "last_month_diff" and "month_before_last_diff" percentages are calculated with
          the shipment count of last month and month before last month as the base respectively.
          If there were no shipments in the base month, the percentage difference is considered 0.
    """
    now = timezone.now()
    this_month = now.month
    this_year = now.year
    last_month = this_month - 1 if this_month != 1 else 12
    last_month_year = this_year if this_month != 1 else this_year - 1
    month_before_last = last_month - 1 if last_month != 1 else 12
    month_before_last_year = last_month_year if last_month != 1 else last_month_year - 1

    this_month_shipments_count = Shipment.objects.filter(
        customer_id=customer_id, created__month=this_month, created__year=this_year
    ).count()

    last_month_shipments_count = Shipment.objects.filter(
        customer_id=customer_id,
        created__month=last_month,
        created__year=last_month_year,
    ).count()

    month_before_last_shipments_count = Shipment.objects.filter(
        customer_id=customer_id,
        created__month=month_before_last,
        created__year=month_before_last_year,
    ).count()

    last_month_diff = (
        abs(last_month_shipments_count - this_month_shipments_count)
        / ((last_month_shipments_count + this_month_shipments_count) / 2)
        * 100
        if this_month_shipments_count > 0
        else 0
    )

    month_before_last_diff = (
        abs(last_month_shipments_count - month_before_last_shipments_count)
        / month_before_last_shipments_count
        * 100
        if month_before_last_shipments_count > 0
        else 0
    )

    return {
        "total_shipments": this_month_shipments_count,
        "last_month_diff": round(last_month_diff, 1),
        "month_before_last_diff": round(month_before_last_diff, 1),
    }


def get_customer_revenue_diff(*, customer_id: uuid.UUID) -> types.CustomerDiffResponse:
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
        abs(this_month_revenue - last_month_revenue)
        / ((last_month_revenue + this_month_revenue) / 2)
        * 100
        if this_month_revenue > 0
        else 0
    )

    month_before_last_diff = (
        abs(last_month_revenue - month_before_last_revenue)
        / ((last_month_revenue + month_before_last_revenue) / 2)
        * 100
        if month_before_last_revenue > 0
        else 0
    )

    return {
        "total_revenue": this_month_revenue,
        "last_month_diff": round(last_month_diff, 1),
        "month_before_last_diff": month_before_last_diff,
    }


def get_customer_on_time_performance_diff(
    *, customer_id: uuid.UUID
) -> types.CustomerOnTimePerfResponse:
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
            movement__shipment__customer_id=customer_id,
            movement__shipment__status__in=[
                StatusChoices.COMPLETED,
                StatusChoices.BILLED,
            ],
            arrival_time__year=year,
            arrival_time__month=month,
        )

        total_stops = customer_stops.count()

        if total_stops == 0:
            stop_percentages[month] = {
                "on_time_percentage": 0.0,
                "early_percentage": 0.0,
                "late_percentage": 0.0,
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
        else 0.0,
        "this_month_early_percentage": this_month_data["early_percentage"],
        "last_month_early_percentage": last_month_data["early_percentage"],
        "early_diff": (
            this_month_data["early_percentage"] - last_month_data["early_percentage"]
        )
        / last_month_data["early_percentage"]
        * 100
        if last_month_data["early_percentage"]
        else 0.0,
        "this_month_late_percentage": this_month_data["late_percentage"],
        "last_month_late_percentage": last_month_data["late_percentage"],
        "late_diff": (
            this_month_data["late_percentage"] - last_month_data["late_percentage"]
        )
        / last_month_data["late_percentage"]
        * 100
        if last_month_data["late_percentage"]
        else 0.0,
    }


def calculate_customer_total_miles(
    *, customer_id: uuid.UUID
) -> types.CustomerMileageResponse:
    """Calculate and return the total mileage and its percentage difference a customer has covered
    between the current month and the previous month.

    This function first sums up the total miles a customer has covered in their completed or billed
    shipments in the current and the previous month.

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

        The response signifies that the customer 123 has covered 1500.0 miles in the current month,
        which is a 25% increase compared to the 1200.0 miles covered in the previous month.
    """
    now = timezone.now()
    this_month = now.month
    this_year = now.year
    last_month = this_month - 1 if this_month != 1 else 12
    last_month_year = this_year if this_month != 1 else this_year - 1

    aggregated_miles = Shipment.objects.filter(
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


def get_customer_shipment_metrics(
    *, customer_id: uuid.UUID
) -> types.CustomerShipmentMetricsResponse:
    """Retrieves and aggregates the last bill date and the last shipment date for a customer
    by their customer_id.

    Args:
        customer_id (uuid.UUID): The unique identifier for the customer.

    Returns:
        types.CustomerShipmentMetricsResponse: Dictionary containing the last_bill_date
        and last_shipment_date in "Month day, Year" format. None is returned
        if the customer doesn't have any shipments yet.

    Raises:
        Do not have any exception raises
    """
    aggregated_dates = Shipment.objects.filter(
        customer_id=customer_id,
    ).aggregate(
        last_bill_date=Max("bill_date"),
        last_shipment_date=Max("ship_date"),
    )

    last_bill_date = aggregated_dates["last_bill_date"]
    last_shipment_date = aggregated_dates["last_shipment_date"]

    return {
        "last_bill_date": last_bill_date,
        "last_shipment_date": last_shipment_date,
    }


def get_customer_email_profile_by_id(
    *, customer_id: str
) -> models.CustomerEmailProfile | None:
    """Returns a customer's email profile by their customer_id. If there's no
    customer with that id, it returns None.

    Args:
        customer_id (str): The unique identifier for the customer.

    Returns:
        models.CustomerEmailProfile | None: A customer's email profile or None if
        no customer with the provided id exists.

    Raises:
        Do not have any exception raises
    """
    try:
        customer = get_object_or_404(models.CustomerEmailProfile, id=customer_id)
    except models.Customer.DoesNotExist:
        return None

    return customer


def get_customer_credit_balance(*, customer_id: uuid.UUID) -> float:
    """Retrieves and aggregates the total credit balance for a customer by their
    customer_id.

    Args:
        customer_id (uuid.UUID): The unique identifier for the customer.

    Returns:
        float: The total credit balance for a given customer. The function will
        return 0 if the customer doesn't have any billing history.

    Raises:
        Do not have any exception raises
    """
    # TODO(Wolfred) Actually write validation using collections module to get total credit balance
    # Or add a status field to invoice to show amount due.
    credit_balance = BillingHistory.objects.filter(
        customer_id=customer_id,
    ).aggregate(
        credit_balance=Sum("total_amount"),
    )["credit_balance"]

    return credit_balance or 0
