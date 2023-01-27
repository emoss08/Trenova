"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""

from django.core.exceptions import ValidationError
from django.utils.translation import gettext_lazy as _

from utils.models import StatusChoices


class StopValidation:
    """
    Validation Class for validating Stop Model
    """

    def __init__(self, *, stop) -> None:
        """Initialize the StopValidation class

        Args:
            stop (Stop): The stop to validate

        Returns:
            None
        """
        self.stop = stop
        self.validate_arrival_departure_movement()
        self.validate_movement_driver_equipment()
        self.validate_reserve_status_change()
        self.validate_compare_app_time()
        self.ensure_location()

    def validate_arrival_departure_movement(self) -> None:
        """Validate arrival and departure times for movement

        If the movement does not have a primary worker or equipment assigned, and
        arrival time is set in the stop. Raise a validation error.

        Returns:
            None

        Raises:
            ValidationError: If the movement does not have a primary worker or
                equipment assigned, and arrival time is set in the stop.
        """

        if (
            not self.stop.movement.primary_worker
            and not self.stop.movement.equipment
            and self.stop.arrival_time
        ):
            raise ValidationError(
                {
                    "arrival_time": _(
                        "Must assign worker or equipment to movement before setting arrival time. Please try again."
                    ),
                },
            )

    def validate_movement_driver_equipment(self) -> None:
        """Validate that the movement driver and equipment are valid

        If the stop status is changed to in progress, validate that the movement
        has a primary driver and equipment assigned. If not raise a validation
        error.

        Returns:
            None

        Raises:
            ValidationError: If the movement does not have a primary driver or
                equipment assigned.
        """

        if (
            not self.stop.movement.primary_worker
            and not self.stop.movement.equipment
            and self.stop.status
            in [
                StatusChoices.IN_PROGRESS,
                StatusChoices.COMPLETED,
            ]
        ):
            raise ValidationError(
                {
                    "status": _(
                        "Cannot change status to in progress or completed if there is no equipment"
                        " or primary worker. Please try again."
                    )
                },
                code="invalid",
            )

    def validate_reserve_status_change(self) -> None:
        """Validate the status change for previous stop

        If the stop status is changed to NEW, validate that the stop previously
        was not in progress or completed. If it was, raise a validation error.

        Returns:
            None

        Raises:
            ValidationError: If the stop status is changed to NEW and the
                previous stop was in progress or completed.
        """
        if self.stop.sequence > 1:
            previous_stop = self.stop.movement.stops.filter(
                sequence=self.stop.sequence - 1
            ).first()

            if (
                previous_stop
                and previous_stop.status != StatusChoices.COMPLETED
                and self.stop.status
                in [
                    StatusChoices.IN_PROGRESS,
                    StatusChoices.COMPLETED,
                ]
            ):
                raise ValidationError(
                    {
                        "status": _(
                            "Cannot change status to in progress or completed if previous stop is "
                            "not completed. Please try again."
                        )
                    },
                    code="invalid",
                )

    def validate_compare_app_time(self) -> None:
        """Validating the appointment time.

        If departure time is set and not the arrival time, raise a validation error.
        If appointment time is changed on a previous stop validate the appointment time
        of the next stop, if after the stop appointment time being changed.

        Returns:
            None

        Raises:
            ValidationError: If the appointment time is not valid.
        """
        if self.stop.departure_time and not self.stop.arrival_time:
            raise ValidationError(
                {
                    "arrival_time": _(
                        "Must set arrival time before setting departure time. Please try again."
                    ),
                },
            )

        if (
            self.stop.departure_time
            and self.stop.departure_time < self.stop.arrival_time
        ):
            raise ValidationError(
                {
                    "departure_time": _(
                        "Departure time must be after arrival time. Please try again."
                    ),
                },
            )

        if self.stop.sequence < self.stop.movement.stops.count():
            next_stop = self.stop.movement.stops.filter(
                sequence__exact=self.stop.sequence + 1
            ).first()

            if next_stop and self.stop.appointment_time > next_stop.appointment_time:
                raise ValidationError(
                    {
                        "appointment_time": _(
                            "Appointment time must be before next stop. Please try again."
                        )
                    }
                )

    def ensure_location(self):
        """Ensure location is entered

        Ensure that either location or address_line is entered.
        If neither is entered, raise a validation error.

        Returns:

        """
        if not self.stop.location and not self.stop.address_line:
            raise ValidationError(
                {
                    "location": ValidationError(
                        _("Must enter a location or address line. Please try again."),
                        code="invalid",
                    )
                }
            )
