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

from order.models import Order, OrderControl


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
    def _calculate_total(instance: Order) -> decimal.Decimal:
        """Calculate the sub_total for a given order.

        Returns:
            decimal.Decimal: The sub_total for the order.
        """

        # Calculate the sub_total if Rating Method is Flat.
        if instance.rate_method == instance.RatingMethodChoices.FLAT:
            if instance.other_charge_amount and instance.freight_charge_amount:
                return instance.other_charge_amount + instance.freight_charge_amount
            elif instance.freight_charge_amount:
                return decimal.Decimal(instance.freight_charge_amount)

        # Calculate the sub_total if Rating Method is Per Mile.
        elif instance.rate_method == instance.RatingMethodChoices.PER_MILE:
            if (
                instance.other_charge_amount
                and instance.mileage
                and instance.freight_charge_amount
            ):
                return (
                    instance.mileage * instance.freight_charge_amount
                    + instance.other_charge_amount
                )
            elif instance.mileage and instance.freight_charge_amount:
                return instance.freight_charge_amount * instance.mileage

        if instance.freight_charge_amount:
            return instance.freight_charge_amount

        return decimal.Decimal(0)

    @staticmethod
    def calculate_order_total(instance: Order) -> Order:
        """Calculate the sub_total for a given order.

        Args:
            instance (Order): The order instance.

        Returns:
            None
        """

        order_control = OrderControl.objects.filter(
            organization=instance.organization
        ).get()

        if instance.ready_to_bill and order_control.auto_order_total:
            instance.sub_total = OrderGenerationService._calculate_total(instance)
            return Order.objects.filter(id=instance.id).update(
                sub_total=instance.sub_total
            )
