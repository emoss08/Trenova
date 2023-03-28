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

from typing import List, Any

from django.db import IntegrityError

from movements.models import Movement
from order.models import Order
from stops import models
from utils.models import StopChoices


class StopService:
    """Stop Service

    Service to manage all stop actions
    """

    @staticmethod
    def create_initial_stops(
        *, movement: Movement, order: Order
    ) -> tuple[models.Stop, models.Stop]:
        """Create Initial Stops for Orders

        Args:
            movement (Movement): The movement instance.
            order (Order): The order instance.

        Returns:
            None

        Raises:
            IntegrityError: If the stop cannot be created.
        """

        try:
            origin_stop: models.Stop = models.Stop.objects.create(
                organization=movement.organization,
                movement=movement,
                sequence=1,
                stop_type=StopChoices.PICKUP,
                location=order.origin_location,
                address_line=order.origin_address,
                appointment_time=order.origin_appointment,
            )
            destination_stop: models.Stop = models.Stop.objects.create(
                organization=movement.organization,
                movement=movement,
                sequence=2,
                stop_type=StopChoices.DELIVERY,
                location=order.destination_location,
                address_line=order.destination_address,
                appointment_time=order.destination_appointment,
            )

        except IntegrityError as stop_creation_error:
            raise stop_creation_error

        return origin_stop, destination_stop

    @staticmethod
    def sequence_stops(instance: models.Stop) -> None:
        """Sequence Stops

        Args:
            instance (Stop): The stop instance.

        Returns:
            None
        """

        # FIXME: Does not properly sequence stops. Need to figure out a way to sequence this properly.

        if instance.organization.order_control.auto_sequence_stops:
            stop_list: List[Any] = []
            stops = models.Stop.objects.filter(movement=instance.movement).order_by(
                "created"
            )

            for index, stop in enumerate(stops):
                stop.sequence = index + 1
                stop_list.append(stop)

            stop_list.sort(key=lambda x: x.stop_type, reverse=True)  # type: ignore
            models.Stop.objects.bulk_update(stop_list, ["sequence"])
