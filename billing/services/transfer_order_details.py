# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------

from typing import Union
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
        instance (Union[models.BillingHistory, models.BillingQueue]): An instance of either the
        `BillingHistory` or `BillingQueue` model.

    Returns:
        None
    """

    def __init__(self, *, instance: Union[BillingQueue, BillingHistory]) -> None:
        self.instance = instance
        self._save()

    def _save(self) -> None:
        """Save the instance of either `BillingHistory` or `BillingQueue`.

        The method transfers the details from `instance.order` to the instance,
        if the instance does not have the information.

        Returns:
            None
        """
        order = self.instance.order
        instance = self.instance

        instance.pieces = instance.pieces or order.pieces
        instance.order_type = instance.order_type or order.order_type
        instance.weight = instance.weight or order.weight
        instance.mileage = instance.mileage or order.mileage
        instance.revenue_code = instance.revenue_code or order.revenue_code
        instance.commodity = instance.commodity or order.commodity
        instance.bol_number = instance.bol_number or order.bol_number
        instance.bill_type = instance.bill_type or models.BillingQueue.BillTypeChoices.INVOICE
        instance.bill_date = instance.bill_date or timezone.now().date()
        instance.consignee_ref_number = instance.consignee_ref_number or order.consignee_ref_number

        if instance.commodity and not instance.commodity_descr:
            instance.commodity_descr = instance.commodity.description

        instance.customer = order.customer
        instance.other_charge_total = order.other_charge_amount
        instance.freight_charge_amount = order.freight_charge_amount
        instance.total_amount = order.sub_total