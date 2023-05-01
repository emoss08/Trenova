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


from datetime import timedelta

from django.shortcuts import get_object_or_404

from dispatch.models import DispatchControl
from stops import models
from utils.models import StopChoices


class StopServiceIncidentHandler:
    def __init__(
        self,
        instance: models.Stop,
        dc_object: DispatchControl,
    ) -> None:
        """Initialize the Create Service Incident class

        Assign instances to get around circular imports.

        Args:
            instance (Stop): The stop to create service incident
            dc_object (DispatchControl): The dispatch object

        Returns:
            None
        """

        self.instance = instance
        self.dispatch_control = get_object_or_404(
            dc_object, organization=self.instance.organization
        )

    def should_create_service_incident(self, stop_type: str) -> bool:
        is_late = (
            self.instance.arrival_time
            and self.instance.arrival_time
            > self.instance.appointment_time
            + timedelta(minutes=self.dispatch_control.grace_period)
        )
        if not self.instance.arrival_time or not is_late:
            return False

        control_choice = self.dispatch_control.record_service_incident

        if control_choice == self.dispatch_control.ServiceIncidentControlChoices.PICKUP:
            return stop_type == StopChoices.PICKUP
        elif (
            control_choice
            == self.dispatch_control.ServiceIncidentControlChoices.DELIVERY
        ):
            return stop_type == StopChoices.DELIVERY
        elif (
            control_choice
            == self.dispatch_control.ServiceIncidentControlChoices.PICKUP_DELIVERY
        ):
            return True
        elif (
            control_choice
            == self.dispatch_control.ServiceIncidentControlChoices.ALL_EX_SHIPPER
        ):
            return stop_type != StopChoices.PICKUP and self.instance.sequence != 1

        return False

    def create_service_incident(self) -> None:
        if self.instance.arrival_time:
            models.ServiceIncident.objects.create(
                organization=self.instance.organization,
                movement=self.instance.movement,
                stop=self.instance,
                delay_time=self.instance.arrival_time - self.instance.appointment_time,
            )

    def create_service_incident_if_needed(self) -> None:
        if self.should_create_service_incident(self.instance.stop_type):
            self.create_service_incident()
