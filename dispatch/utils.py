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
from typing import TYPE_CHECKING

from dispatch import exceptions, models, services
from movements.models import Movement
from organization.models import Organization
from worker.models import Worker, WorkerHOS

if TYPE_CHECKING:
    from django.db.models import QuerySet


def calculate_worker_miles_per_week(*, worker: Worker) -> float:
    worker_weekly_hos = WorkerHOS.objects.filter(worker=worker)

    return sum(hos.miles_driven for hos in worker_weekly_hos)


def calculate_worker_miles_per_day(
    *, worker: Worker, reference_date: datetime.date
) -> float:
    week_start = reference_date - timedelta(days=reference_date.weekday())
    week_end = reference_date  # Update the week_end to be the reference_date

    worker_weekly_hos = WorkerHOS.objects.filter(worker=worker)

    total_miles_driven = sum(hos.miles_driven for hos in worker_weekly_hos)
    total_days = (week_end - week_start).days + 1

    return total_miles_driven / total_days if total_days > 0 else 0


def evaluate_criteria(
    *, driver_value: float, operator: str, criteria_value: float
) -> bool:
    if operator == "gte":
        return driver_value >= criteria_value
    elif operator == "gt":
        return driver_value > criteria_value
    elif operator == "lte":
        return driver_value <= criteria_value
    elif operator == "lt":
        return driver_value < criteria_value
    elif operator == "eq":
        return driver_value == criteria_value
    else:
        raise exceptions.OperatorNotFound(f"Invalid operator: {operator}")


def calculate_worker_otp(*, worker: Worker) -> float:
    on_time_stops = 0
    total_stops = 0

    movements = Movement.objects.filter(primary_worker=worker)

    if not movements:
        raise exceptions.WorkerOTPCalculationError(
            f"Worker {worker.code} has no movements."
        )

    for movement in movements:
        stops = movement.stops.all()
        for stop in stops:
            if stop.arrival_time <= stop.appointment_time_window_end:
                on_time_stops += 1
            total_stops += 1

    return on_time_stops / total_stops if total_stops > 0 else 0.0


def get_eligible_drivers(
    *,
    workers_hos: "QuerySet[WorkerHOS]",
    origin_appointment: datetime,
    destination_appointment: datetime,
    travel_time: int,
    organization: Organization,
    total_order_miles: int,
    pickup_time_window_start: datetime,
    pickup_time_window_end: datetime,
    delivery_time_window_start: datetime,
) -> tuple[list[WorkerHOS], list[WorkerHOS]]:
    eligible_workers_hos = []
    ineligible_workers_hos = []

    # Get the feasibility tool control settings
    feasibility_control = models.FeasibilityToolControl.objects.filter(
        organization=organization
    ).first()

    for worker_hos in workers_hos:
        worker = worker_hos.worker

        # Calculate worker's miles per week (MPW)
        worker_mpw = calculate_worker_miles_per_week(worker=worker)

        # Calculate worker's miles per day (MPD)
        worker_mpd = calculate_worker_miles_per_day(
            worker=worker, reference_date=worker_hos.log_date
        )

        # Check if worker meets the feasibility criteria
        if evaluate_criteria(
            driver_value=worker_mpw,
            operator=feasibility_control.mpw_operator,
            criteria_value=feasibility_control.mpw_criteria,
        ) and evaluate_criteria(
            driver_value=worker_mpd,
            operator=feasibility_control.mpd_operator,
            criteria_value=feasibility_control.mpd_criteria,
        ):
            print(f"Worker {worker} meets the feasibility criteria")

            # Check if worker is eligible for the order
            driver_info = services.feasibility_tool(
                origin_appointment=origin_appointment,
                destination_appointment=destination_appointment,
                travel_time=travel_time,
                driver_daily_miles=int(worker_mpd),
                total_order_miles=total_order_miles,
                seventy_hour_time=worker_hos.seventy_hour_time,
                drive_time=worker_hos.drive_time,
                on_duty_time=worker_hos.on_duty_time,
                pickup_time_window_start=pickup_time_window_start,
                pickup_time_window_end=pickup_time_window_end,
                delivery_time_window_start=delivery_time_window_start,
            )

            # If driver_info is not None, the worker is eligible
            if driver_info is not None:
                print(f"Worker {worker} is eligible for the order")
                eligible_workers_hos.append(worker_hos)
            else:
                ineligible_workers_hos.append(worker_hos)
                print(f"Worker {worker} is NOT eligible for the order")
        else:
            ineligible_workers_hos.append(worker_hos)
            print(f"Worker {worker} does not meet the feasibility criteria")

    return eligible_workers_hos, ineligible_workers_hos
