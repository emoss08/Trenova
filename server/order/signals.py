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

from movements.models import Movement
from movements.services.generation import MovementService
from order import models
from order.services.order_service import OrderService


@receiver(pre_save, sender=models.Order)
def generate_pro_number(
    sender: models.Order, instance: models.Order, **kwargs: Any
) -> None:
    """Generate Pro Number

    Generate a pro number when a new order is added.

    Args:
        sender (Order): Order
        instance (Order): The order instance.
        **kwargs (Any): Keyword arguments.

    Returns:
        None
    """
    if not instance.pro_number:
        instance.pro_number = OrderService.set_pro_number()


@receiver(post_save, sender=models.Order)
def generate_order_movement(
    sender: models.Order, instance: models.Order, created: bool, **kwargs: Any
) -> None:
    """Generate the initial movement for the order

    Args:
        sender (Order): Order
        instance (Order): The Order instance.
        created (bool): if the Order was created
        **kwargs (Any): Keyword Arguments

    Returns:
        None
    """
    if created and not Movement.objects.filter(order=instance).exists():
        MovementService.create_initial_movement(instance)
