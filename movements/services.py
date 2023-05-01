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
from django.db import IntegrityError
from typing import TYPE_CHECKING
from movements import models
from stops.models import Stop
from utils.models import StopChoices

if TYPE_CHECKING:
    from order.models import Order


def set_ref_number() -> str:
    """Generate a unique movement reference number.

    Returns:
        str: The generated reference number.
    """
    code = f"MOV{models.Movement.objects.count() + 1:06d}"
    return (
        "MOV000001" if models.Movement.objects.filter(ref_num=code).exists() else code
    )


def create_initial_stops(
    *, movement: models.Movement, order: "Order"
) -> tuple[Stop, Stop]:
    """Create Initial Stops for Orders

    Args:
        movement (Movement): The movement instance.
        order (Order): The order instance.

    Returns:
        tuple[Stop, Stop]: The origin and destination stop.

    Raises:
        IntegrityError: If the stop cannot be created.
    """

    try:
        origin_stop: Stop = Stop.objects.create(
            organization=movement.organization,
            movement=movement,
            sequence=1,
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

    except IntegrityError as stop_creation_error:
        raise stop_creation_error

    return origin_stop, destination_stop
