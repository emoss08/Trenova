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

from django.db.models import Sum

from order.models import Order
from stops.models import Stop


class OrderService:
    """Order Service

    Service to manage all order actions
    """

    @staticmethod
    def total_pieces(order: Order) -> int:
        """Get the total piece count for the order

        Returns:
            int: Total piece count for the order
        """
        return Stop.objects.filter(movement__order__exact=order).aggregate(
            Sum("pieces")
        )["pieces__sum"]

    @staticmethod
    def total_weight(order: Order) -> int:
        """Get the total weight for the order.

        Returns:
            int: Total weight for the order
        """
        return Stop.objects.filter(movement__order__exact=order).aggregate(
            Sum("weight")
        )["weight__sum"]

    @staticmethod
    def set_pro_number() -> str:
        """Set the pro_number for the order

        Returns:
            str: The pro_number for the order
        """
        code = f"ORD{Order.objects.count() + 1:06d}"
        return "ORD000001" if Order.objects.filter(pro_number=code).exists() else code
