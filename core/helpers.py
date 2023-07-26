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
from accounting.models import DivisionCode
from accounting.serializers import DivisionCodeSerializer
from order.models import Order
from order.serializers import OrderSerializer


def order_display(order: Order) -> str:
    return f"Order {order.pro_number}, status: {order.get_status_display()}, from {order.origin_address} to {order.destination_address}"


def division_code_display(division_code: DivisionCode) -> str:
    return f"Division Code {division_code.code}, status: {division_code.get_status_display()}"


searchable_models = {
    "Order": {
        "app": "order",
        "serializer": OrderSerializer,
        "search_fields": [
            "pro_number",
            "origin_address",
            "destination_address",
            "bol_number",
            "status",
        ],
        "display": order_display,
    },
    "DivisionCode": {
        "app": "accounting",
        "serializer": DivisionCodeSerializer,
        "search_fields": [
            "code",
            "status",
            "description",
        ],
        "display": division_code_display,
    },
}
