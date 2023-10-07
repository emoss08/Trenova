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
import decimal
import typing

from django.db.models import Sum

from stops.models import Stop

if typing.TYPE_CHECKING:
    from shipment.models import Shipment


def get_total_piece_count_by_shipment(*, shipment: "Shipment") -> int:
    """Return the total piece count for an order

    Args:
        shipment (Shipment): shipment instance

    Returns:
        int: Total piece counts for an order
    """
    value: int = Stop.objects.filter(movement__shipment__exact=shipment).aggregate(
        Sum("pieces")
    )["pieces__sum"]
    return value or 0


def get_total_weight_by_shipment(*, shipment: "Shipment") -> decimal.Decimal | int:
    """Return the total weight for an order

    Args:
        shipment (Shipment): shipment instance

    Returns:
        decimal.Decimal: Total weight for an order
    """
    value: decimal.Decimal = Stop.objects.filter(
        movement__shipment__exact=shipment
    ).aggregate(Sum("weight"))["weight__sum"]
    return value or 0
