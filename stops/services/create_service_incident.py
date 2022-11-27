"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""
from datetime import timedelta

from utils.models import StopChoices


class CreateServiceIncident:
    """
    Create service incident
    """

    def __init__(self, stop, dc_object, si_object) -> None:
        """Initialize the Create Service Incident class

        Assign instances to get around circular imports.

        Args:
            stop (Stop): The stop to create service incident
            dc_object (DispatchControl): The dispatch object
            si_object (ServiceIncident): The service incident object

        Returns:
            None
        """

        self.stop = stop
        self.dc_object = dc_object
        self.si_object = si_object

    def create(self) -> None:
        """Create service incident

        Create service incident based on the organization's service incident control settings.

        Returns:
            None
        """

        self.create_pickup_service_incident()
        self.create_delivery_service_incident()
        self.create_pick_and_delivery_service_incident()
        self.create_all_exc_shipper_service_incident()

    def create_pickup_service_incident(self) -> None:
        """Create pickup service incident

        If the stop is a pickup, and the organization has pickup service incident control
        enabled, create a service incident for the stop.

        Returns:
            None
        """

        dispatch_control = self.dc_object.objects.get(
            organization=self.stop.organization
        )

        if (
            self.stop.arrival_time
            and dispatch_control.record_service_incident
            == self.dc_object.ServiceIncidentControlChoices.PICKUP
            and self.stop.stop_type == StopChoices.PICKUP
            and self.stop.arrival_time
            > self.stop.appointment_time
            + timedelta(minutes=dispatch_control.grace_period)
        ):
            self.si_object.objects.create(
                organization=self.stop.movement.order.organization,
                movement=self.stop.movement,
                stop=self,
                delay_time=self.stop.arrival_time - self.stop.appointment_time,
            )

    def create_delivery_service_incident(self) -> None:
        """Create delivery service incident

        If the stop is a delivery, and the organization has delivery service incident control
        enabled, create a service incident for the stop.

        Returns:
            None
        """

        dispatch_control = self.dc_object.objects.get(
            organization=self.stop.organization
        )

        if (
            dispatch_control.record_service_incident
            == self.dc_object.ServiceIncidentControlChoices.DELIVERY
            and self.stop.stop_type == StopChoices.DELIVERY
            and self.stop.arrival_time
            > self.stop.appointment_time
            + timedelta(minutes=dispatch_control.grace_period)
        ):
            self.si_object.objects.create(
                organization=self.stop.movement.order.organization,
                movement=self.stop.movement,
                stop=self,
                delay_time=self.stop.arrival_time - self.stop.appointment_time,
            )

    def create_pick_and_delivery_service_incident(self) -> None:
        """Create pickup and delivery service incident

        If the stop is a pickup or delivery, and the organization has pickup and delivery service incident control
        enabled, create a service incident for the stop.

        Returns:
            None
        """

        dispatch_control = self.dc_object.objects.get(
            organization=self.stop.organization
        )

        if (
            dispatch_control.record_service_incident
            == self.dc_object.ServiceIncidentControlChoices.PICKUP_DELIVERY
            and self.stop.arrival_time
            > self.stop.appointment_time
            + timedelta(minutes=dispatch_control.grace_period)
        ):
            self.si_object.objects.create(
                organization=self.stop.movement.order.organization,
                movement=self.stop.movement,
                stop=self.stop,
                delay_time=self.stop.arrival_time - self.stop.appointment_time,
            )

    def create_all_exc_shipper_service_incident(self) -> None:
        """Create all except shipper service incident

        If the stop is not a shipper, and the organization has all except shipper service incident control
        enabled, create a service incident for the stop.

        Returns:
            None
        """

        dispatch_control = self.dc_object.objects.get(
            organization=self.stop.organization
        )

        if (
            dispatch_control.record_service_incident
            == self.dc_object.ServiceIncidentControlChoices.ALL_EX_SHIPPER
            and self.stop.stop_type != StopChoices.PICKUP
            and self.stop.sequence != 1
            and self.stop.arrival_time
            > self.stop.appointment_time
            + timedelta(minutes=dispatch_control.grace_period)
        ):
            self.si_object.objects.create(
                organization=self.stop.movement.order.organization,
                movement=self.stop.movement,
                stop=self.stop,
                delay_time=self.stop.arrival_time - self.stop.appointment_time,
            )
