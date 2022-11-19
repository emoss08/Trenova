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

import decimal

from order.exceptions import RatingException
from order.models import Order


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


class OrderTotalService:
    """
    Generate the total amount of an order.
    """
    rating_method: type[Order.RatingMethodChoices] = Order.RatingMethodChoices

    def __init__(self, order: Order):
        self.order = order

    def _calculate_flat_rate(self) -> decimal.Decimal:
        try:
            if self.order.rate_method == self.rating_method.FLAT:
                if self.order.other_charge_amount and self.order.freight_charge_amount:
                    return self.order.other_charge_amount + self.order.freight_charge_amount
                elif self.order.freight_charge_amount:
                    return decimal.Decimal(self.order.freight_charge_amount)
            
        except RatingException as rating_exception:
            raise rating_exception
