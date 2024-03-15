# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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


from datetime import timedelta

from dispatch.models import DispatchControl
from stops import models


def is_late(*, obj: models.Stop) -> bool:
    if not obj.arrival_time:
        return False
    grace_period = timedelta(minutes=obj.organization.dispatch_control.grace_period)
    return obj.arrival_time > obj.appointment_time_window_end + grace_period


def control_choice_matches_stop(
    *, control_choice: DispatchControl.ServiceIncidentControlChoices, obj: models.Stop
) -> bool:
    stop_type = obj.stop_type
    si_choices = DispatchControl.ServiceIncidentControlChoices

    control_mapping = {
        si_choices.PICKUP: models.StopChoices.PICKUP,
        si_choices.DELIVERY: models.StopChoices.DELIVERY,
        si_choices.PICKUP_DELIVERY: None,
        si_choices.ALL_EX_SHIPPER: models.StopChoices.PICKUP,
    }

    return stop_type == control_mapping.get(control_choice, None)


def should_create_service_incident(*, obj: models.Stop) -> bool:
    dispatch_control = obj.organization.dispatch_control
    if not is_late(obj=obj):
        return False
    return control_choice_matches_stop(
        control_choice=dispatch_control.record_service_incident, obj=obj
    )


def create_service_incident(*, obj: models.Stop) -> None:
    if obj.arrival_time:
        delay_time = obj.arrival_time - obj.appointment_time_window_end
        models.ServiceIncident.objects.create(
            organization=obj.organization,
            business_unit=obj.business_unit,
            movement=obj.movement,
            stop=obj,
            delay_time=delay_time,
        )


def create_service_incident_if_needed(obj: models.Stop) -> None:
    if should_create_service_incident(obj=obj):
        create_service_incident(obj=obj)
