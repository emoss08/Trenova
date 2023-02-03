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


class TransferOrderDetails:
    """
    Class used to transfer order details to the BillingHistory
    & BillingQueue models.
    """

    def __init__(self, *, model):
        self.model = model
        self.save()

    def save(self):
        # If order has `pieces`, set `pieces` to order `pieces`
        if self.model.order.pieces and not self.model.pieces:
            self.model.pieces = self.model.order.pieces

        # Set order `order_type` to `order_type` if it is not set
        if not self.model.order_type:
            self.model.order_type = self.model.order.order_type

        # If order has `weight`, set `weight` to order `weight`
        if self.model.order.weight and not self.model.weight:
            self.model.weight = self.model.order.weight

        # If order has `mileage`, set `mileage` to order `mileage`
        if self.model.order.mileage and not self.model.weight:
            self.model.mileage = self.model.order.mileage

        # If order has `revenue_code`, set `revenue_code` to order `revenue_code`
        if self.model.order.revenue_code and not self.model.revenue_code:
            self.model.revenue_code = self.model.order.revenue_code

        if not self.model.commodity and self.model.order.commodity:
            self.model.commodity = self.model.order.commodity

        # If commodity `description` is set, set `commodity_descr` to the description of the commodity
        if self.model.commodity and self.model.commodity.description:
            self.model.commodity_descr = self.model.commodity.description

        # if order has `bol_number`, set `bol_number` to `bol_number`
        if self.model.order.bol_number and not self.model.bol_number:
            self.model.bol_number = self.model.order.bol_number

        # If `bill_type` is not set, set `bill_type` to `INVOICE`
        if not self.model.bill_type:
            self.model.bill_type = "INVOICE"

        # If not `bill_date`, set `bill_date` to `timezone.now().date()`
        if not self.model.bill_date:
            self.model.bill_date = timezone.now().date()

        # If order has `consignee_ref_number`
        if self.model.order.consignee_ref_number and not self.model.consignee_ref_number:
            self.model.consignee_ref_number = self.model.order.consignee_ref_number

        self.model.customer = self.model.order.customer
        self.model.other_charge_total = self.model.order.other_charge_amount
        self.model.freight_charge_amount = self.model.order.freight_charge_amount
        self.model.total_amount = self.model.order.sub_total
