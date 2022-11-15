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


class OrderGenerationService:
    """Order Generation Service

    Generate a unique code for the order.
    """

    @staticmethod
    def pro_number() -> str:
        """Generate a unique code for the order

        Returns:
            str: Generated Pro Number
        """
        code = f"ORD{Order.objects.count() + 1:06d}"
        return "ORD000001" if Order.objects.filter(pro_number=code).exists() else code

    @staticmethod
    def movement_ref_number() -> str:
        """Generate a unique code for the movement

        Returns:
            str: Generated Movement Reference Number
        """
        code = f"MOV{Movement.objects.count() + 1:06d}"
        return (
            "MOV000001"
            if Movement.objects.filter(movement_ref_number=code).exists()
            else code
        )
