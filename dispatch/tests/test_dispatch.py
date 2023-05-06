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

from datetime import datetime, timedelta

import pytest
from django.utils import timezone

from dispatch import models, utils
from dispatch.factories import FleetCodeFactory
from equipment.tests.factories import TractorFactory
from order.selectors import get_order_movements, get_order_stops
from order.tests.factories import OrderFactory
from organization.models import Organization
from worker.factories import WorkerFactory
from worker.models import WorkerHOS

pytestmark = pytest.mark.django_db


def test_feasibility_tool_eligible_driver(organization: Organization) -> None:
    """Tests worker is eligible for order based on feasibility tool.

    Args:
        organization (Organization): Organization object.

    Returns:
        None: this function does not return anything.
    """
    fleet = FleetCodeFactory(organization=organization)
    worker = WorkerFactory(organization=organization, fleet=fleet)
    tractor = TractorFactory(
        organization=organization, fleet=fleet, primary_worker=worker
    )

    order = OrderFactory()
    movements = get_order_movements(order=order)

    for movement in movements:
        movement.primary_worker = worker
        movement.tractor = tractor
        movement.save()

    stops = get_order_stops(order=order)

    for stop in stops:
        if stop.sequence == 1:
            stop.appointment_time_window_start = timezone.now() - timedelta(hours=1)
            stop.appointment_time_window_end = timezone.now() - timedelta(hours=4)
            stop.arrival_time = timezone.now() - timedelta(hours=2)
            stop.departure_time = timezone.now() - timedelta(hours=2)
        else:
            stop.appointment_time_window_start = timezone.now() + timedelta(hours=1)
            stop.appointment_time_window_end = timezone.now() + timedelta(hours=4)
            stop.arrival_time = timezone.now() + timedelta(minutes=1)
            stop.departure_time = timezone.now() + timedelta(minutes=1)
        stop.save()
        stop.refresh_from_db()

    # Query all available WorkerHOS instances
    worker_hos = WorkerHOS.objects.create(
        organization=organization,
        worker=worker,
        drive_time=11 * 60,
        off_duty_time=10 * 60,
        sleeper_berth_time=0,
        on_duty_time=14 * 60,
        violation_time=0,
        miles_driven=400,
        seventy_hour_time=70 * 60,
        current_status="On Duty",
        current_location="Test Location",
        log_date=timezone.now().date(),
        last_reset_date=timezone.now().date() - timedelta(days=7),
    )

    # Create a FeasibilityControl instance
    models.FeasibilityToolControl.objects.create(
        organization=organization,
        mpw_operator=models.FeasibilityToolControl.OperatorChoices.GREATER_THAN_OR_EQUAL_TO,
        mpw_criteria=2,
        mpd_operator=models.FeasibilityToolControl.OperatorChoices.GREATER_THAN_OR_EQUAL_TO,
        mpd_criteria=2,
        otp_operator=models.FeasibilityToolControl.OperatorChoices.GREATER_THAN_OR_EQUAL_TO,
        otp_criteria=0.1,
    )

    workers_hos = WorkerHOS.objects.all()

    # Example values
    origin_appointment = datetime.now()
    destination_appointment = datetime.now() + timedelta(days=5)
    travel_time = 8 * 60  # 8 hours in minutes
    pickup_time_window_start = datetime.now() + timedelta(hours=1)
    pickup_time_window_end = datetime.now() + timedelta(hours=4)
    delivery_time_window_start = datetime.now() + timedelta(days=5, hours=1)

    # Call the get_eligible_drivers function
    eligible_workers_hos, ineligible_workers_hos = utils.get_eligible_drivers(
        delivery_time_window_start=delivery_time_window_start,
        destination_appointment=destination_appointment,
        organization=organization,
        origin_appointment=origin_appointment,
        pickup_time_window_end=pickup_time_window_end,
        pickup_time_window_start=pickup_time_window_start,
        travel_time=travel_time,
        workers_hos=workers_hos,
        total_order_miles=100,
        last_reset_date=worker_hos.last_reset_date,
    )

    assert worker_hos in eligible_workers_hos


def test_feasibility_tool_not_eligible(organization: Organization) -> None:
    """Test Driver not eligible for order.

    Args:
        organization (Organization): Organization object

    Returns:
        None: This function does not return anything.
    """
    fleet = FleetCodeFactory(organization=organization)
    worker = WorkerFactory(organization=organization, fleet=fleet)
    tractor = TractorFactory(
        organization=organization, fleet=fleet, primary_worker=worker
    )

    order = OrderFactory()
    movements = get_order_movements(order=order)

    for movement in movements:
        movement.primary_worker = worker
        movement.tractor = tractor
        movement.save()

    stops = get_order_stops(order=order)

    for stop in stops:
        if stop.sequence == 1:
            stop.appointment_time_window_start = timezone.now() - timedelta(hours=1)
            stop.appointment_time_window_end = timezone.now() - timedelta(hours=4)
            stop.arrival_time = timezone.now() - timedelta(hours=2)
            stop.departure_time = timezone.now() - timedelta(hours=2)
        else:
            stop.appointment_time_window_start = timezone.now() + timedelta(hours=1)
            stop.appointment_time_window_end = timezone.now() + timedelta(hours=4)
            stop.arrival_time = timezone.now() + timedelta(minutes=1)
            stop.departure_time = timezone.now() + timedelta(minutes=1)
        stop.save()
        stop.refresh_from_db()

    # Query all available WorkerHOS instances
    worker_hos = WorkerHOS.objects.create(
        organization=organization,
        worker=worker,
        drive_time=11,
        off_duty_time=10,
        sleeper_berth_time=0,
        on_duty_time=14,
        violation_time=0,
        miles_driven=100,
        seventy_hour_time=70,
        current_status="On Duty",
        current_location="Test Location",
        log_date=timezone.now().date(),
        last_reset_date=timezone.now().date() - timedelta(days=7),
    )

    # Create a FeasibilityControl instance
    models.FeasibilityToolControl.objects.create(
        organization=organization,
        mpw_operator=models.FeasibilityToolControl.OperatorChoices.GREATER_THAN_OR_EQUAL_TO,
        mpw_criteria=100,
        mpd_operator=models.FeasibilityToolControl.OperatorChoices.GREATER_THAN_OR_EQUAL_TO,
        mpd_criteria=100,
        otp_operator=models.FeasibilityToolControl.OperatorChoices.GREATER_THAN_OR_EQUAL_TO,
        otp_criteria=1.0,
    )

    workers_hos = WorkerHOS.objects.all()

    # Example values
    travel_time = 8 * 60  # 8 hours in minutes

    # Call the get_eligible_drivers function
    eligible_workers_hos, ineligible_workers_hos = utils.get_eligible_drivers(
        delivery_time_window_start=order.destination_appointment_window_start,
        destination_appointment=order.destination_appointment_window_start,
        organization=organization,
        origin_appointment=order.origin_appointment_window_start,
        pickup_time_window_end=order.origin_appointment_window_start,
        pickup_time_window_start=order.origin_appointment_window_end,
        travel_time=travel_time,
        workers_hos=workers_hos,
        total_order_miles=100,
        last_reset_date=worker_hos.last_reset_date,
    )

    assert worker_hos in ineligible_workers_hos
