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

from movements import models, services
from utils.models import StatusChoices


def generate_initial_stops(
    created: bool, instance: models.Movement, **kwargs: Any
) -> None:
    """Generate initial movements stops.

    This hook should only be fired if the first movement is being added to the order.
    Its purpose is to create the initial stops for the movement, by taking the origin
    and destination from the order. This is done by calling the StopService. This
    service will then create the stops and sequence them.

    Returns:
        None
    """

    if (
        instance.order.status == StatusChoices.NEW
        and instance.order.movements.count() == 1
        and created
    ):
        services.create_initial_stops(movement=instance, order=instance.order)


def generate_ref_number(instance: models.Movement, **kwargs: Any) -> None:
    """Generate the ref_num before saving the Movement

    Returns:
        None
    """
    if not instance.ref_num:
        instance.ref_num = services.set_ref_number()


def update_order_status(instance: models.Movement, **kwargs: Any) -> None:
    movements = instance.order.movements.all()
    completed_movements = movements.filter(status=StatusChoices.COMPLETED)
    in_progress_movements = movements.filter(status=StatusChoices.IN_PROGRESS)

    if movements.count() == completed_movements.count():
        new_status = StatusChoices.COMPLETED
    elif completed_movements.count() > 0 or in_progress_movements.count() > 0:
        new_status = StatusChoices.IN_PROGRESS
    else:
        new_status = StatusChoices.NEW

    instance.order.status = new_status
    instance.order.save()


def check_movement_removal_policy(
    instance: models.Movement,
    **kwargs: Any,
) -> None:
    """Check if the organization allows order removal.

    If the organization does not allow order removal throw a ValidationError.

    Args:
        instance (models.Movement): The instance of the Movement model being saved.
        **kwargs: Additional keyword arguments.

    Returns:
        None: This function does not return anything.
    """

    if instance.organization.order_control.remove_orders is False:
        raise ValidationError(
            {
                "ref_num": _(
                    "Organization does not allow Movement removal. Please contact your administrator."
                )
            },
            code="invalid",
        )


def validate_movement_order(instance: models.Movement, **kwargs: Any) -> None:
    """Validate previous movement is completed before setting current movement in progress

    This method validates the given instance of the Movement model before saving it. Specifically,
    it checks if the instance's status is set to 'IN_PROGRESS' and if there are any previous movements
    with the same order that are not completed. If there are any previous movements that are not
    completed, it raises a validation error with a message indicating that the previous movement(s) must
    be completed before the current movement can be set to 'IN_PROGRESS'.

    Args:
        instance (models.Movement): An instance of Movement model to be validated.
        **kwargs (Any): Additional keyword arguments.

    Raises:
        ValidationError: If the instance status is set to IN_PROGRESS and the previous movement(s) with the same order are not completed.

    Returns:
        None: This function does not return anything.
    """

    if instance.status in [StatusChoices.IN_PROGRESS, StatusChoices.COMPLETED]:
        previous_movements = models.Movement.objects.filter(
            order=instance.order, id__lt=instance.id
        )

        for movement in previous_movements:
            if movement.status != StatusChoices.COMPLETED:
                raise ValidationError(
                    {
                        "status": _(
                            f"The previous movement (ID: {movement.ref_num}) must be completed before this movement can be set to `{instance.get_status_display()}` Please try again."
                        )
                    },
                    code="invalid",
                )
