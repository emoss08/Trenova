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

from django.core.exceptions import ValidationError
from django.utils.functional import Promise
from django.utils.translation import gettext_lazy as _

from stops import models
from utils.models import StatusChoices


class StopValidation:
    """
    Validation Class for validating Stop Model
    """

    def __init__(self, *, instance: models.Stop) -> None:
        """
        Validation Class for validating Stop Model

        This class is used to validate the Stop model before saving it to the database.
        It performs multiple checks on the stop, such as verifying the presence of a primary worker or tractor,
        the arrival and departure time, and the status of the previous stop. If any errors are found, a
        ValidationError is raised.

        Attributes:
            instance (Stop): The stop to validate
        """
        self.instance = instance
        self.errors: dict[str, Promise] = {}
        self.validate()

        if self.errors:
            raise ValidationError(self.errors)

    def validate(self) -> None:
        """Validate the stop model.

        This method calls all the validation methods in the class, which perform specific checks on the stop.
        If any errors are found, they are stored in the `errors` dictionary.

        Returns:
            None
        """
        self.validate_arrival_departure_movement()
        self.validate_movement_driver_tractor()
        self.validate_reserve_status_change()
        self.validate_compare_app_time()
        self.ensure_location()

    def validate_arrival_departure_movement(self) -> None:
        """Validate arrival and departure times for movement

        If the movement does not have a primary worker or tractor assigned, and
        arrival time is set in the stop, a validation error is raised.

        Returns:
            None

        Raises:
            ValidationError: If the movement does not have a primary worker or
                tractor assigned, and arrival time is set in the stop.
        """

        if (
            not self.instance.movement.primary_worker
            and not self.instance.movement.tractor
            and self.instance.arrival_time
        ):
            self.errors["arrival_time"] = _(
                "Must assign worker or tractor to movement before setting arrival time. Please try again."
            )

    def validate_movement_driver_tractor(self) -> None:
        """Validate that the movement driver and tractor are valid

        If the stop status is changed to in progress, validate that the movement
        has a primary driver and tractor assigned. If not, a validation error is raised.

        Returns:
            None

        Raises:
            ValidationError: If the movement does not have a primary driver or
                tractor assigned.
        """

        if (
            not self.instance.movement.primary_worker
            and not self.instance.movement.tractor
            and self.instance.status
            in [
                StatusChoices.IN_PROGRESS,
                StatusChoices.COMPLETED,
            ]
        ):
            self.errors["status"] = _(
                "Cannot change status to in progress or completed if there is no tractor or primary worker. Please try again."
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
        if self.instance.sequence > 1:
            previous_stop = self.instance.movement.stops.filter(
                sequence=self.instance.sequence - 1
            ).first()

            if (
                previous_stop
                and previous_stop.status != StatusChoices.COMPLETED
                and self.instance.status
                in [
                    StatusChoices.IN_PROGRESS,
                    StatusChoices.COMPLETED,
                ]
            ):
                self.errors["status"] = _(
                    "Cannot change status to in progress or completed if previous stop is not completed. Please try again."
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
        if self.instance.departure_time and not self.instance.arrival_time:
            self.errors["arrival_time"] = _(
                "Must set arrival time before setting departure time. Please try again."
            )

        if (
            self.instance.departure_time
            and self.instance.arrival_time
            and self.instance.departure_time < self.instance.arrival_time
        ):
            self.errors["departure_time"] = _(
                "Departure time must be after arrival time. Please try again."
            )

        if self.instance.sequence < self.instance.movement.stops.count():
            next_stop = self.instance.movement.stops.filter(
                sequence__exact=self.instance.sequence + 1
            ).first()

            if (
                next_stop
                and self.instance.appointment_time > next_stop.appointment_time
            ):
                self.errors["appointment_time"] = _(
                    "Appointment time must be before next stop. Please try again."
                )

    def ensure_location(self):
        """Ensure location is entered

        Ensure that either location or address_line is entered.
        If neither is entered, raise a validation error.

        Returns:

        """
        if not self.instance.location and not self.instance.address_line:
            self.errors["location"] = _(
                "Must enter a location or address line. Please try again."
            )
