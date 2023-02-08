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
from django.utils import timezone

from billing import models
from billing.models import BillingHistory, BillingQueue


class TransferOrderDetails:
    """
    Class to transfer details from `instance` to billing queue and billing history.
    If the instance is a `models.BillingHistory`, it will transfer details from the `instance.order`
    to the instance, if the instance does not have the information.

    If the instance is a `models.BillingQueue`, it will transfer details from the `instance.order`
    to the instance, if the instance does not have the information.

    Args:
        instance (models.BillingHistory | models.BillingQueue): An instance of either the
        `BillingHistory` or `BillingQueue` model.

    Returns:
        None
    """

    def __init__(self, *, instance: BillingQueue | BillingHistory) -> None:
        self.instance = instance
        self.save()

    def save(self) -> None:
        """Save the instance of either `BillingHistory` or `BillingQueue`.

        The method transfers the details from `instance.order` to the instance,
        if the instance does not have the information.

        Returns:
            None
        """
        # If order has `pieces`, set `pieces` to order `pieces`
        if self.instance.order.pieces and not self.instance.pieces:
            self.instance.pieces = self.instance.order.pieces

        # Set order `order_type` to `order_type` if it is not set
        if not self.instance.order_type:
            self.instance.order_type = self.instance.order.order_type

        # If order has `weight`, set `weight` to order `weight`
        if self.instance.order.weight and not self.instance.weight:
            self.instance.weight = self.instance.order.weight

        # If order has `mileage`, set `mileage` to order `mileage`
        if self.instance.order.mileage and not self.instance.weight:
            self.instance.mileage = self.instance.order.mileage

        # If order has `revenue_code`, set `revenue_code` to order `revenue_code`
        if self.instance.order.revenue_code and not self.instance.revenue_code:
            self.instance.revenue_code = self.instance.order.revenue_code

        if not self.instance.commodity and self.instance.order.commodity:
            self.instance.commodity = self.instance.order.commodity

        # If commodity `description` is set, set `commodity_descr` to the description of the commodity
        if self.instance.commodity and self.instance.commodity.description:
            self.instance.commodity_descr = self.instance.commodity.description

        # if order has `bol_number`, set `bol_number` to `bol_number`
        if self.instance.order.bol_number and not self.instance.bol_number:
            self.instance.bol_number = self.instance.order.bol_number

        # If `bill_type` is not set, set `bill_type` to `INVOICE`
        if not self.instance.bill_type:
            self.instance.bill_type = models.BillingQueue.BillTypeChoices.INVOICE

        # If not `bill_date`, set `bill_date` to `timezone.now().date()`
        if not self.instance.bill_date:
            self.instance.bill_date = timezone.now().date()

        # If order has `consignee_ref_number`, set `consignee_ref_number` to order `consignee_ref_number`
        if (
            self.instance.order.consignee_ref_number
            and not self.instance.consignee_ref_number
        ):
            self.instance.consignee_ref_number = (
                self.instance.order.consignee_ref_number
            )

        self.instance.customer = self.instance.order.customer
        self.instance.other_charge_total = self.instance.order.other_charge_amount
        self.instance.freight_charge_amount = self.instance.order.freight_charge_amount
        self.instance.total_amount = self.instance.order.sub_total
