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
import datetime
from typing import Tuple, Any

from django.db.models import Avg, F, ExpressionWrapper, fields
from location import models
from stops.models import Stop


def get_avg_wait_time(*, location: models.Location) -> datetime.timedelta | Any:
    """Calculates the average wait time for a given location and returns it as a `datetime.timedelta` object.

    Args:
        location (models.Location): The location for which to calculate the average wait time.

    Returns:
        datetime.timedelta: A `datetime.timedelta` object representing the average wait time for the given
        location. The object represents a duration as a combination of days, seconds,
        and microseconds.
    """
    return (
        Stop.objects.filter(location=location)
        .annotate(
            wait_time=ExpressionWrapper(
                F("departure_time") - F("arrival_time"),
                output_field=fields.DurationField(),
            )
        )
        .aggregate(avg_wait_time=Avg("wait_time"))["avg_wait_time"]
    )


def get_avg_wait_time_hours_minutes(*, location: models.Location) -> Tuple[int, int]:
    """Returns the average wait time in hours and minutes for a given location, formatted as a tuple of integers.

    Args:
        location (models.Location): The location for which to calculate the average wait time.

    Returns:
        Tuple[int, int]: A tuple of integers representing the average wait time in hours and minutes. The
        first element of the tuple is the number of hours in the average wait time, and
        the second element is the number of minutes in the average wait time.
    """
    avg_wait_time = get_avg_wait_time(location=location)

    total_seconds = int(avg_wait_time.total_seconds())
    hours, remainder = divmod(total_seconds, 3600)
    minutes, seconds = divmod(remainder, 60)

    return hours, minutes
