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
from order.models import Order


def transfer_order_details(obj: Union[BillingHistory, BillingQueue]) -> None:
    """Save the obj of either `BillingHistory` or `BillingQueue`.

    The method transfers the details from `obj.order` to the obj,
    if the obj does not have the information.

    Returns:
        None
    """
    order = Order.objects.select_related(
        "order_type", "revenue_code", "commodity", "customer"
    ).get(pk=obj.order.pk)

    obj.pieces = obj.pieces or order.pieces
    obj.order_type = obj.order_type or order.order_type
    obj.weight = obj.weight or order.weight
    obj.mileage = obj.mileage or order.mileage
    obj.revenue_code = obj.revenue_code or order.revenue_code
    obj.commodity = obj.commodity or order.commodity
    obj.bol_number = obj.bol_number or order.bol_number
    obj.bill_type = (
        obj.bill_type or models.BillingQueue.BillTypeChoices.INVOICE
    )
    obj.bill_date = obj.bill_date or timezone.now().date()
    obj.consignee_ref_number = (
        obj.consignee_ref_number or order.consignee_ref_number
    )

    if obj.commodity and not obj.commodity_descr:
        obj.commodity_descr = obj.commodity.description

    obj.customer = order.customer
    obj.other_charge_total = order.other_charge_amount
    obj.freight_charge_amount = order.freight_charge_amount
    obj.total_amount = order.sub_total
