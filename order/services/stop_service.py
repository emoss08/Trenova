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
from order.models import Movement, Order, OrderControl, Stop, StopChoices


class StopService:
    """Stop Service

    Service to manage all stop actions
    """

    @staticmethod
    def create_initial_stops(movement: Movement, order: Order) -> tuple[Stop, Stop]:
        """Create Initial Stops for Orders

        Args:
            movement (Movement): The movement instance.
            order (Order): The order instance.

        Returns:
            None
        """
        origin_stop: Stop = Stop.objects.create(
            organization=movement.organization,
            movement=movement,
            stop_type=StopChoices.PICKUP,
            location=order.origin_location,
            address_line=order.origin_address,
            appointment_time=order.origin_appointment,
        )
        destination_stop: Stop = Stop.objects.create(
            organization=movement.organization,
            movement=movement,
            sequence=2,
            stop_type=StopChoices.DELIVERY,
            location=order.destination_location,
            address_line=order.destination_address,
            appointment_time=order.destination_appointment,
        )

        return origin_stop, destination_stop

    @staticmethod
    def sequence_stops(instance: Stop) -> None:
        """Sequence Stops

        Args:
            instance (Stop): The stop instance.

        Returns:
            None

        Raises:
            SequenceException: If the stop sequence is not valid.
        """
        order_control: OrderControl = OrderControl.objects.filter(
            organization=instance.organization
        ).get()
        if order_control.auto_sequence_stops:
            stop_list = []
            stops = Stop.objects.filter(movement=instance.movement).order_by("created")

            for index, stop in enumerate(stops):
                stop.sequence = index + 1
                stop_list.append(stop)

            stop_list.sort(key=lambda x: x.stop_type, reverse=True)
            Stop.objects.bulk_update(stop_list, ["sequence"])
