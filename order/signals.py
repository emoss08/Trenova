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

from dispatch.services.transfer_rate_details import transfer_rate_details
from movements.models import Movement
from movements.services.generation import MovementService

from order import models
from route.services import get_order_mileage
from stops.selectors import total_weight_for_order, total_piece_count_for_order
from utils.models import StatusChoices
from order.services.pro_number_service import set_pro_number


def set_total_piece_and_weight(
    instance: models.Order,
    created: bool,
    **kwargs: Any,
) -> None:
    """
    Set total pieces and weight of a completed order.

    This function is called as a signal when an Order model instance is saved.
    If the order status is COMPLETED, it sets the total pieces and weight of the
    order using the respective helper functions.

    Args:
        instance (models.Order): The instance of the Order model being saved.
        created (bool): True if a new record was created, False otherwise.
        **kwargs: Additional keyword arguments.
    """
    if instance.status == StatusChoices.COMPLETED:
        instance.pieces = total_piece_count_for_order(order=instance)
        instance.weight = total_weight_for_order(order=instance)


def set_field_values(
    sender: models.Order, instance: models.Order, **kwargs: Any
) -> None:
    """Set various field values of an Order model instance.

    This function is called as a signal when an Order model instance is saved.
    It sets values for pro_number, sub_total, origin_address, destination_address,
    and hazmat fields based on certain conditions.

    Args:
        sender (models.Order): The class of the sending instance.
        instance (models.Order): The instance of the Order model being saved.
        **kwargs: Additional keyword arguments.
    """
    _set_pro_number(instance)
    _set_sub_total(instance)
    _set_origin_and_destination_addresses(instance)
    _set_hazmat(instance)


def _set_pro_number(instance: models.Order) -> None:
    if not instance.pro_number:
        instance.pro_number = set_pro_number(organization=instance.organization)


def _set_sub_total(instance: models.Order) -> None:
    if instance.ready_to_bill and instance.organization.order_control.auto_order_total:
        instance.sub_total = instance.calculate_total()


def _set_origin_and_destination_addresses(instance: models.Order) -> None:
    if instance.origin_location and not instance.origin_address:
        instance.origin_address = instance.origin_location.get_address_combination

    if instance.destination_location and not instance.destination_address:
        instance.destination_address = (
            instance.destination_location.get_address_combination
        )


def _set_hazmat(instance: models.Order) -> None:
    if instance.commodity and instance.commodity.hazmat:
        instance.hazmat = instance.commodity.hazmat


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
        MovementService.create_initial_movement(order=instance)


def check_order_removal_policy(
    instance: models.Order,
    **kwargs: Any,
) -> None:
    """Check if the organization allows order removal.

    If the organization does not allow order removal throw a ValidationError.

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


def transfer_rate_information(instance: models.Order, **kwargs: Any) -> None:
    """Transfer rate information from the order to the movement.

    Args:
        instance (models.Order): The instance of the Order model being saved.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """

    if instance.rate:
        transfer_rate_details(order=instance)


def set_order_mileage_and_create_route(instance: models.Order, **kwargs) -> None:
    instance.mileage = get_order_mileage(order=instance)
