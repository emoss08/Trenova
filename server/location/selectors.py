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

from typing import Any

from django.db.models import Avg, ExpressionWrapper, F, QuerySet
from django.db.models.fields import DurationField

from location import models

if TYPE_CHECKING:
    from utils.types import ModelUUID

def get_location_by_pk(*, location_id: "ModelUUID") -> models.Location | None:
    try:
        return models.Location.objects.get(pk=location_id)
    except models.Location.DoesNotExist:
        return None


def get_avg_wait_time(
    *, queryset: QuerySet[models.Location]
) -> QuerySet[models.Location] | Any:
    """Annotates the given queryset with the average wait time for each location.

    Args:
        queryset: The queryset to annotate.
    """
    if not isinstance(queryset, QuerySet):
        raise TypeError(
            f"Expected queryset to be of type {QuerySet}, got {type(queryset)}."
        )

    queryset: QuerySet[models.Location] = queryset.annotate(
        wait_time_avg=Avg(
            ExpressionWrapper(
                F("stop__departure_time") - F("stop__arrival_time"),
                output_field=DurationField(),
            )
        )
    )
    return queryset


def get_avg_wait_time_hours_minutes(
    *, queryset: QuerySet[models.Location]
) -> tuple[int, int]:
    """Returns the average wait time in hours and minutes for a given location, formatted as a tuple of integers.

    Args:
        queryset: The queryset to annotate.

    Returns:
        Tuple[int, int]: A tuple of integers representing the average wait time in hours and minutes. The
        first element of the tuple is the number of hours in the average wait time, and
        the second element is the number of minutes in the average wait time.
    """
    avg_wait_time = get_avg_wait_time(queryset=queryset)

    for location in avg_wait_time:
        total_seconds = location.wait_time_avg.total_seconds()
        hours, remainder = divmod(total_seconds, 3600)
        minutes, seconds = divmod(remainder, 60)
        return hours, minutes
    return 0, 0
