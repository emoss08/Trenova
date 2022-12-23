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

from order.models import Order


class OrderService:
    """Order Service

    Service to manage all order actions
    """

    @staticmethod
    def set_pro_number() -> str:
        """Generate a unique pro number for an order.

        Returns:
            str: The pro number for the order.
        """
        count = Order.objects.count() + 1
        pro_number = f"ORD{count:06d}"

        # Check if pro number already exists and generate a new one if it does.
        while Order.objects.filter(pro_number=pro_number).exists():
            count += 1
            pro_number = f"ORD{count:06d}"

        return pro_number
