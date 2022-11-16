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

from order.models import Movement, Order


class MovementService:
    """Movement Service

    Service to manage all movement actions
    """

    @staticmethod
    def movement_ref_number() -> str:
        """Generate a unique code for the movement

        Returns:
            str: Generated Movement Reference Number
        """
        code = f"MOV{Movement.objects.count() + 1:06d}"
        return "MOV000001" if Movement.objects.filter(ref_num=code).exists() else code

    @staticmethod
    def create_initial_movement(instance: Order) -> None:
        """Create Initial Movements

        Create the initial movements for the order.

        Args:
            instance (Order): The order instance.

        Returns:
            None
        """
        Movement.objects.create(
            organization=instance.organization,
            order=instance,
        )
