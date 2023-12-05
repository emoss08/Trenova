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

from typing import Any

from movements.models import Movement
from route.services import get_shipment_mileage
from shipment import models, services


def create_shipment_initial_movement(
    instance: models.Shipment, created: bool, **kwargs: Any
) -> None:
    """Create the initial movement of a shipment model instance.

    This function is called as a signal when a shipment model instance is saved.
    If the shipment does not have any associated Movement model instances, it creates
    the initial movement using the MovementService.

    Args:
        instance (models.Shipment): The instance of the shipment model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if not created:
        return

    if not Movement.objects.filter(shipment=instance).exists():
        services.create_initial_movement(shipment=instance)


def set_shipment_mileage_and_create_route(
    instance: models.Shipment, **kwargs: Any
) -> None:
    """Set the mileage for a shipment and create a route.

    This function is called as a signal when a shipment model instance is saved.
    If the shipment has an origin and destination location, it sets the mileage
    for the shipment and creates a route using the generate_route().

    Args:
        instance (models.Shipment): The instance of the shipment model being saved.
        **kwargs (Any): Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if instance.origin_location and instance.destination_location:
        instance.mileage = get_shipment_mileage(shipment=instance)
