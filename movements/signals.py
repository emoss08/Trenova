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

from typing import Any

from django.db.models.signals import post_save, pre_save
from django.dispatch import receiver

from movements import models


@receiver(post_save, sender=models.Movement)
def generate_movement_stops(
    sender: models.Movement, instance: models.Movement, created: bool, **kwargs: Any
):
    """Generate the movement stops

    Args:
        sender (Movement): Movement
        instance (Movement): The Movement instance.
        created (bool): if the Movement was created
        **kwargs (Any): Keyword Arguments

    Returns:
        None
    """
    if created and not instance.stops.exists():
        stop_service.StopService.create_initial_stops(instance, instance.order)


@receiver(pre_save, sender=models.Movement)
def set_movement_ref_number(
    sender: models.Movement, instance: models.Movement, **kwargs: Any
) -> None:
    """Set the Movement Reference Number

    Args:
        sender (Movement): Movement
        instance (Movement): The Movement instance.
        **kwargs (Any): Keyword Arguments

    Returns:
        None
    """
    if not instance.ref_num:
        instance.ref_num = movement_service.MovementService.set_ref_number()
