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

from typing import TYPE_CHECKING

from django.db.models.aggregates import Sum

from movements.models import Movement
from shipment import models
from stops.models import Stop

if TYPE_CHECKING:
    from django.db.models import QuerySet

    from utils.types import ModelUUID


def get_shipment_by_id(*, shipment_id: "ModelUUID") -> models.Shipment | None:
    """Get an shipment model instance by its ID.

    Args:
        shipment_id (str): The ID of the shipment.

    Returns:
        models.Shipment: The shipment model instance.
    """
    try:
        return models.Shipment.objects.get(pk__exact=shipment_id)
    except models.Shipment.DoesNotExist:
        return None


def get_shipment_movements(*, shipment: models.Shipment) -> "QuerySet[Movement]":
    """Get the movements of an shipment.

    Args:
        shipment (models.Shipment): The shipment.

    Returns:
        QuerySet[Movement]: QuerySet of the movements of the shipment.
    """
    return Movement.objects.filter(shipment=shipment)


def get_shipment_stops(*, shipment: models.Shipment) -> "QuerySet[Stop]":
    """Get the stops of an shipment.

    Args:
        shipment (models.Shipment): The shipment.

    Returns:
        QuerySett[Stop]: QuerySet of the stops of the shipment.
    """
    movements = get_shipment_movements(shipment=shipment)
    return Stop.objects.filter(movement__in=movements).select_related("movement")


def sum_shipment_additional_charges(*, shipment: models.Shipment) -> float:
    """Sum the additional charges of an shipment.

    Args:
        shipment (models.Shipment): The shipment.

    Returns:
        float: The sum of the additional charges.
    """
    # Calculate the sum of sub_total for each additional charge associated with the order
    additional_charges_total = models.AdditionalCharge.objects.filter(
        shipment=shipment
    ).aggregate(total=Sum("sub_total"))["total"]

    # If there are no additional charges associated with the order, return 0
    return additional_charges_total or 0
