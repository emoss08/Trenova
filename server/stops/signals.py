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

from stops import models, services
from utils.models import StatusChoices


def update_movement_status(instance: models.Stop, **kwargs: Any) -> None:
    # If the stop is VOIDED, skip the logic below
    if instance.status == models.StatusChoices.VOIDED:
        return

    stops = instance.movement.stops.all()

    completed_stops = stops.filter(status=StatusChoices.COMPLETED)
    in_progress_stops = stops.filter(status=StatusChoices.IN_PROGRESS)

    # Determine the new status for the Movement
    if stops.count() == completed_stops.count():
        new_status = StatusChoices.COMPLETED
    elif completed_stops.count() > 0 or in_progress_stops.count() > 0:
        new_status = StatusChoices.IN_PROGRESS
    else:
        new_status = StatusChoices.NEW

    instance.movement.status = new_status
    instance.movement.save()


def create_service_incident(instance: models.Stop, **kwargs: Any) -> None:
    if instance.arrival_time:
        services.create_service_incident_if_needed(obj=instance)
