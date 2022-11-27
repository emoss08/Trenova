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

    def __init__(self, stop, stop_object) -> None:
        """Initialize the StopValidation class

        Args:
            stop (Stop): The stop to validate
            stop_object (Stop): Instance of the stop Model

        Returns:
            None
        """
        self.stop = stop
        self.stop_object = stop_object

    def validate(self) -> None:
        """Validate the stop

        Validate the stop by calling the validation methods.

        Returns:
            None

        Raises:
            ValidationError: If the stop is not valid
        """

        self.validate_compare_app_time()
        self.validate_previous_app_time()
        self.validate_next_app_time()
        self.validate_reserve_status_change()
        self.validate_movement_driver_equipment()
        self.validate_arrival_departure_movement()

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
                        "Must assign worker or equipment to movement before"
                        " setting arrival time."
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

        if not self.stop.movement.primary_worker and not self.stop.movement.equipment:
            if self.stop.status in [
                StatusChoices.IN_PROGRESS,
                StatusChoices.COMPLETED,
            ]:
                raise ValidationError(
                    {
                        "status": ValidationError(
                            _(
                                "Cannot change status to in progress or completed if"
                                " there is no equipment or primary worker."
                            ),
                            code="invalid",
                        )
                    }
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
        if self.stop.status == StatusChoices.NEW:
            old_status = self.stop_object.objects.get(pk=self.stop.pk).status

            if old_status in [StatusChoices.IN_PROGRESS, StatusChoices.COMPLETED]:
                raise ValidationError(
                    {
                        "status": ValidationError(
                            _(
                                "Cannot change status to new if the status was"
                                " previously in progress or completed."
                            ),
                            code="invalid",
                        )
                    }
                )
            previous_stop = self.stop.movement.stops.filter(
                sequence=self.stop.sequence - 1
            ).first()

            if previous_stop and previous_stop.status != StatusChoices.COMPLETED:
                if self.stop.status in [
                    StatusChoices.IN_PROGRESS,
                    StatusChoices.COMPLETED,
                ]:
                    raise ValidationError(
                        {
                            "status": ValidationError(
                                _(
                                    "Cannot change status to in progress or completed if"
                                    " previous stop is not completed."
                                ),
                                code="invalid",
                            )
                        }
                    )

    def validate_next_app_time(self) -> None:
        """Validate appointment time for next stop.

        If the appointment time on the stop is not after the previous stop time,
        raise a validation error.

        Returns:
            None

        Raises:
            ValidationError: If the appointment time is not after the previous
                stop time.
        """

        if self.stop.sequence > 1:
            previous_stop = self.stop.movement.stops.filter(
                sequence=self.stop.sequence - 1
            ).first()

            if (
                previous_stop
                and self.stop.appointment_time < previous_stop.appointment_time
            ):
                raise ValidationError(
                    {
                        "appointment_time": _(
                            "Appointment time must be after previous stop."
                        ),
                    },
                )

    def validate_previous_app_time(self) -> None:
        """Validate the stop appointment time is after the previous stop

        If the stop appointment time is after the previous stop appointment time,
        raise a validation error.

        Returns:
            None

        Raises:
            ValidationError: If the stop appointment time is not after the
                previous stop appointment time.
        """
        if self.stop.sequence < self.stop.movement.stops.count():
            next_stop = self.stop.movement.stops.filter(
                sequence__exact=self.stop.sequence + 1
            ).first()

            if next_stop and self.stop.appointment_time > next_stop.appointment_time:
                raise ValidationError(
                    {
                        "appointment_time": _(
                            "Appointment time must be before next stop."
                        ),
                    },
                )

            if (
                next_stop
                and self.stop.status != StatusChoices.COMPLETED
                and next_stop.status
                in [
                    StatusChoices.COMPLETED,
                    StatusChoices.IN_PROGRESS,
                ]
            ):
                raise ValidationError(
                    {
                        "status": _(
                            "Previous stop must be completed before this stop can"
                            " be in progress or completed."
                        ),
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
                    "departure_time": _(
                        "Must set arrival time before setting departure time."
                    ),
                },
            )

        if (
            self.stop.departure_time
            and self.stop.arrival_time
            and self.stop.departure_time < self.stop.arrival_time
        ):
            raise ValidationError(
                {
                    "departure_time": _("Departure time must be after arrival time."),
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
                            "Appointment time must be before next stop."
                        )
                    }
                )
