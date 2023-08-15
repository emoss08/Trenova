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

from django.utils import timezone

from dispatch import models, selectors
from order.models import AdditionalCharge, Order
from order.selectors import sum_order_additional_charges

if TYPE_CHECKING:
    from datetime import datetime


def get_rate(*, order: Order) -> models.Rate | None:
    """Get the rate for the order.

    Args:
        order(Order): The order to get the rate for.

    Returns:
        models.Rate | None: The rate for the order or None if no rate is found.
    """
    today = timezone.now().date()
    rates = models.Rate.objects.filter(
        customer=order.customer,
        commodity=order.commodity,
        order_type=order.order_type,
        equipment_type=order.equipment_type,
        origin_location=order.origin_location,
        destination_location=order.destination_location,
        effective_date__lte=today,
        expiration_date__gte=today,
    )
    return rates.first() if rates.exists() else None


def transfer_rate_details(order: Order) -> None:
    """Transfer rate details to the order.

    Args:
        order (Order): The order to transfer rate details to.

    Returns:
        None: This function does not return anything.
    """

    if rate := get_rate(order=order):
        order.freight_charge_amount = rate.rate_amount
        order.mileage = rate.distance_override

        for billing_item in selectors.get_rate_billing_table_by_rate(rate=rate):
            # Check if the charge already exists on the order
            additional_charge_exists = AdditionalCharge.objects.filter(
                organization=order.organization,
                order=order,
                accessorial_charge=billing_item.accessorial_charge,
            ).exists()

            if not additional_charge_exists:
                AdditionalCharge.objects.create(
                    organization=order.organization,
                    order=order,
                    accessorial_charge=billing_item.accessorial_charge,
                    charge_amount=billing_item.charge_amount,
                    unit=billing_item.unit,
                    description=billing_item.description,
                    entered_by=order.entered_by,
                )

        order.other_charge_amount = sum_order_additional_charges(order=order)


def feasibility_tool(
    *,
    drive_time: int,
    on_duty_time: int,
    seventy_hour_time: int,
    origin_appointment: "datetime",
    destination_appointment: "datetime",
    travel_time: int,
    driver_daily_miles: int,
    total_order_miles: int,
    pickup_time_window_start: "datetime",
    pickup_time_window_end: "datetime",
    delivery_time_window_start: "datetime",
    last_reset_date: "datetime",
) -> tuple[int, float] | None:
    # Calculate the number of days between the origin and destination appointments
    days_between_appointments = (destination_appointment - origin_appointment).days

    # Calculate the maximum possible miles the driver can drive based on their daily average
    max_possible_miles = days_between_appointments * driver_daily_miles

    time_since_last_reset = (origin_appointment.date() - last_reset_date).days  # type: ignore
    eligible_for_restart = time_since_last_reset >= 8

    if eligible_for_restart:
        # Update the seventy_hour_time to the maximum allowed value after the restart
        seventy_hour_time = 70 * 60

    # Check if the driver can cover the total order miles within the available days
    if total_order_miles >= max_possible_miles:
        return None

    # Calculate the number of breaks required to complete the order
    breaks_required = (travel_time - 1) // (11 * 60)

    # Calculate the total driving time required to complete the order, including breaks
    total_driving_time_required = travel_time + breaks_required * 10 * 60

    # Calculate the total on-duty time required to complete the order, including breaks
    total_on_duty_time_required = total_driving_time_required + breaks_required * 3 * 60

    if (
        drive_time < total_driving_time_required
        or on_duty_time < total_on_duty_time_required
    ):
        return None

    # Calculate the breaks duration
    breaks_duration = breaks_required * 10 * 60

    # Calculate the time left on the driver's 70-hour clock after taking the order
    time_left_after_order = seventy_hour_time - total_on_duty_time_required

    # Check if the driver can reach the pickup location within the pickup time window
    time_until_pickup_start = (
        pickup_time_window_start - origin_appointment
    ).total_seconds() / 60
    can_reach_pickup = (
        time_left_after_order >= time_until_pickup_start
        and time_until_pickup_start <= travel_time
    )

    # Calculate the time spent at the pickup location
    time_spent_at_pickup = (
        pickup_time_window_end - pickup_time_window_start
    ).total_seconds() / 60

    # Check if the driver can reach the destination within the delivery time window
    time_until_delivery_start = (
        delivery_time_window_start - destination_appointment
    ).total_seconds() / 60
    total_time_required = travel_time + breaks_duration + time_spent_at_pickup

    can_reach_delivery = (
        time_left_after_order >= time_until_delivery_start
        and time_until_delivery_start <= total_time_required
    )

    if can_reach_pickup and can_reach_delivery:
        if time_left_after_order < total_time_required:
            sleeper_berth_time = on_duty_time - drive_time
            can_use_sleeper_berth = (
                sleeper_berth_time >= 8 * 60
                and total_on_duty_time_required - drive_time <= sleeper_berth_time
            )
            if not can_use_sleeper_berth:
                return None

        return (
            (breaks_required, time_left_after_order)
            if time_left_after_order >= 0
            else None
        )
    return None
