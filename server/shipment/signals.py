# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

from django.db import transaction

from movements.models import Movement
from shipment import models, services, selectors
from stops.models import Stop


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


def update_stops_on_shipment_change(
    sender: models.Shipment, instance: models.Shipment, **kwargs: Any
) -> None:
    """Update the stops for a shipment when the shipment is changed.

    This function is called as a signal when a shipment model instance is saved.
    If the origin or destination of the shipment is changed, this function updates
    the stops for the shipment's movements.

    Args:
        sender (models.Shipment): The class of the model sending the signal.
        instance (models.Shipment): The instance of the shipment model being saved.

    Returns:
        None: This function does not return anything.
    """
    if not instance.pk or not models.Shipment.objects.filter(pk=instance.pk).exists():
        # If it's a new instance or doesn't exist in the database, there's nothing to do
        return

    movements = selectors.get_shipment_movements(shipment=instance)

    print("movements", movements)

    if not movements:
        return

    old_instance = models.Shipment.objects.get(pk=instance.pk)

    origin_changed = any(
        [
            old_instance.origin_location != instance.origin_location,
            old_instance.origin_address != instance.origin_address,
            old_instance.origin_appointment_window_start
            != instance.origin_appointment_window_start,
            old_instance.origin_appointment_window_end
            != instance.origin_appointment_window_end,
        ]
    )

    destination_changed = any(
        [
            old_instance.destination_location != instance.destination_location,
            old_instance.destination_address != instance.destination_address,
            old_instance.destination_appointment_window_start
            != instance.destination_appointment_window_start,
            old_instance.destination_appointment_window_end
            != instance.destination_appointment_window_end,
        ]
    )

    if origin_changed or destination_changed:
        with transaction.atomic():
            for movement in movements:
                if origin_changed:
                    # Update the first stop (pickup) for this movement
                    Stop.objects.filter(movement=movement, sequence=1).update(
                        location=instance.origin_location,
                        address_line=instance.origin_address,
                        appointment_time_window_start=instance.origin_appointment_window_start,
                        appointment_time_window_end=instance.origin_appointment_window_end,
                    )

                if destination_changed:
                    # Update the last stop (delivery) for this movement
                    last_sequence = (
                        movement.stops.order_by("-sequence").first().sequence
                    )
                    print("last_sequence", last_sequence)
                    Stop.objects.filter(
                        movement=movement, sequence=last_sequence
                    ).update(
                        location=instance.destination_location,
                        address_line=instance.destination_address,
                        appointment_time_window_start=instance.destination_appointment_window_start,
                        appointment_time_window_end=instance.destination_appointment_window_end,
                    )
