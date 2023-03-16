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

from movements.services.generation import MovementService
from stops.services.generation import StopService
from utils.models import StatusChoices
from movements import models


def generate_initial_stops(
    sender: models.Movement, created: bool, instance: models.Movement, **kwargs: Any
) -> None:
    """Generate initial movements stops.

    This hook should only be fired if the first movement is being added to the order.
    Its purpose is to create the initial stops for the movement, by taking the origin
    and destination from the order. This is done by calling the StopService. This
    service will then create the stops and sequence them.

    Returns:
        None
    """

    if (
        instance.order.status == StatusChoices.NEW
        and instance.order.movements.count() == 1
    ):
        StopService.create_initial_stops(movement=instance, order=instance.order)


def generate_ref_number(
    sender: models.Movement, instance: models.Movement, **kwargs: Any
) -> None:
    """Generate the ref_num before saving the Movement

    Returns:
        None
    """
    if not instance.ref_num:
        instance.ref_num = MovementService.set_ref_number()
