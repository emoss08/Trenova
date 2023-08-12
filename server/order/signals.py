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

from django.core.exceptions import ValidationError
from django.utils.translation import gettext_lazy as _
from movements.models import Movement
from order import models, services
from route.services import get_order_mileage


def set_order_pro_number(instance: models.Order, **kwargs: Any) -> None:
    """Set a pro number for an order.

    This function sets a pro number for the passed Order instance if it does not already have one. It calls the set_pro_number
    function from the services module, passing in the organization associated with the order.

    Args:
        instance: An instance of the Order model.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if not instance.pro_number:
        instance.pro_number = services.set_pro_number(
            organization=instance.organization
        )


def create_order_initial_movement(
    instance: models.Order, created: bool, **kwargs: Any
) -> None:
    """Create the initial movement of an Order model instance.

    This function is called as a signal when an Order model instance is saved.
    If the order does not have any associated Movement model instances, it creates
    the initial movement using the MovementService.

    Args:
        instance (models.Order): The instance of the Order model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if not created:
        return

    if not Movement.objects.filter(order=instance).exists():
        services.create_initial_movement(order=instance)


def check_order_removal_policy(
    instance: models.Order,
    **kwargs: Any,
) -> None:
    """Check if the organization allows order removal.

    If the organization does not allow order removal, throw a ValidationError.

    Args:
        instance (models.Order): The instance of the Order model being saved.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if instance.organization.order_control.remove_orders is False:
        raise ValidationError(
            {
                "pro_number": _(
                    "Organization does not allow order removal. Please contact your administrator."
                )
            },
            code="invalid",
        )


def set_order_mileage_and_create_route(instance: models.Order, **kwargs: Any) -> None:
    """Set the mileage for an order and create a route.

    This function is called as a signal when an Order model instance is saved.
    If the order has an origin and destination location, it sets the mileage
    for the order and creates a route using the generate_route().

    Args:
        instance (models.Order): The instance of the Order model being saved.
        **kwargs (Any): Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """
    if instance.origin_location and instance.destination_location:
        instance.mileage = get_order_mileage(order=instance)
