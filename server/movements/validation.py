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

import datetime

from django.core.exceptions import ValidationError
from django.utils import timezone
from django.utils.translation import gettext_lazy as _

from equipment.models import AvailabilityChoices
from movements import models
from utils.models import StatusChoices
from worker.models import WorkerProfile


class MovementValidation:
    """Class to validate the movement model.

    This class validates a movement model and raises a `ValidationError` if any
    validation fails. The validation includes checking regulatory compliance, worker
    commodity compatibility, movement stop status, worker comparison, and movement worker.

    Attributes:
        movement: The movement model to be validated.
        errors: A dictionary that stores the validation error messages.

    Raises:
        ValidationError: If any validation fails.
    """

    def __init__(self, *, movement: models.Movement):
        """Initialize the `MovementValidation` class.

        Args:
            movement: The movement model to be validated.
        """
        self.movement = movement
        self.errors = {}
        self.validate()

    def validate(self) -> None:
        """Validate the movement model.

        The `validate` method calls several other validation methods to perform
        all the necessary validations.

        Returns:
            None: This function does not return anything.

        Raises:
            ValidationError: If any validation fails.
        """
        self.validate_regulatory()
        self.validate_worker_commodity()
        self.validate_movement_stop_status()
        self.validate_worker_compare()
        self.validate_movement_worker()
        self.validate_worker_tractor_fleet()
        self.validate_movement_shipment()
        self.validate_trailer_status_is_available()

        if self.errors:
            raise ValidationError(self.errors)

    def validate_regulatory(self) -> None:
        """Validate Worker regulatory.

        Call all regulatory validation methods. If any of the methods
        raise a ValidationError, the method will return the first
        ValidationError raised.

        Returns:
            None
        """

        if self.movement.organization.dispatch_control.regulatory_check:
            if self.movement.primary_worker:
                self.validate_primary_worker_regulatory()
            if self.movement.secondary_worker:
                self.validate_secondary_worker_regulatory()

    def validate_primary_worker_regulatory(self):
        """Validate primary worker regulatory information.

        Returns:
            None

        Raises:
            ValidationError: if worker regulatory information is invalid.
        """

        if (
            self.movement.primary_worker
            and self.movement.primary_worker.profile.license_expiration_date
            and self.movement.primary_worker.profile.license_expiration_date
            < timezone.now().date()
        ):
            self.errors["primary_worker"] = _(
                "Cannot assign a worker with an expired license. Please update the worker's"
                " profile and try again."
            )

        if (
            self.movement.primary_worker
            and self.movement.primary_worker.profile.physical_due_date
            and self.movement.primary_worker.profile.physical_due_date
            < timezone.now().date()
        ):
            self.errors["primary_worker"] = _(
                "Cannot assign a worker with an expired physical. Please update the worker's"
                " profile and try again."
            )

        if (
            self.movement.primary_worker
            and self.movement.primary_worker.profile.medical_cert_date
            and self.movement.primary_worker.profile.medical_cert_date
            < timezone.now().date()
        ):
            self.errors["primary_worker"] = _(
                "Cannot assign a worker with an expired medical certificate. Please update the worker's"
                " profile and try again."
            )

        if (
            self.movement.primary_worker
            and self.movement.primary_worker.profile.medical_cert_date
            and self.movement.primary_worker.profile.medical_cert_date
            < timezone.now().date()
        ):
            self.errors["primary_worker"] = _(
                "Cannot assign a worker with an expired medical certificate. Please update the worker's"
                " profile and try again."
            )

        if (
            self.movement.primary_worker
            and self.movement.primary_worker.profile.mvr_due_date
            and self.movement.primary_worker.profile.mvr_due_date
            < timezone.now().date()
        ):
            self.errors["primary_worker"] = _(
                "Cannot assign a worker with an expired MVR. Please update the worker's"
                " profile and try again."
            )

        if (
            self.movement.primary_worker
            and self.movement.primary_worker.profile.termination_date
        ):
            self.errors["primary_worker"] = _(
                "Cannot assign a terminated worker. Please update the worker's profile and try again."
            )

    def validate_secondary_worker_regulatory(self):
        """Validate primary worker regulatory information.

        Returns:
            None

        Raises:
            ValidationError: if worker regulatory information is invalid.
        """

        if (
            self.movement.secondary_worker
            and self.movement.secondary_worker.profile.license_expiration_date
            and self.movement.secondary_worker.profile.license_expiration_date
            < timezone.now().date()
        ):
            self.errors["secondary_worker"] = _(
                "Cannot assign a worker with an expired license. Please update the worker's"
                " profile and try again."
            )

        if (
            self.movement.secondary_worker
            and self.movement.secondary_worker.profile.physical_due_date
            and self.movement.secondary_worker.profile.physical_due_date
            < timezone.now().date()
        ):
            self.errors["secondary_worker"] = _(
                "Cannot assign a worker with an expired physical. Please update the worker's"
                " profile and try again."
            )

        if (
            self.movement.secondary_worker
            and self.movement.secondary_worker.profile.medical_cert_date
            and self.movement.secondary_worker.profile.medical_cert_date
            < timezone.now().date()
        ):
            self.errors["secondary_worker"] = _(
                "Cannot assign a worker with an expired medical certificate. Please update the worker's"
                " profile and try again."
            )

        if (
            self.movement.secondary_worker
            and self.movement.secondary_worker.profile.mvr_due_date
            and self.movement.secondary_worker.profile.mvr_due_date
            < timezone.now().date()
        ):
            self.errors["secondary_worker"] = _(
                "Cannot assign a worker with an expired MVR. Please update the worker's"
                " profile and try again."
            )

        if (
            self.movement.secondary_worker
            and self.movement.secondary_worker.profile.termination_date
        ):
            self.errors["secondary_worker"] = _(
                "Cannot assign a terminated worker. Please update the worker's profile and try again."
            )

    def validate_worker_compare(self) -> None:
        """Validate that the workers do not match when creating movement.

        Returns:
            None

        Raises:
            ValidationError: If the workers are the same.
        """

        if (
            self.movement.primary_worker
            and self.movement.secondary_worker
            and self.movement.primary_worker == self.movement.secondary_worker
        ):
            self.errors["primary_worker"] = _(
                "Primary worker cannot be the same as secondary worker. Please try again."
            )

    def validate_worker_commodity(self) -> None:
        """Validate Worker Commodity

        Validate that the assigned worker is allowed to move the commodity.

        Returns:
            None: This function does not return anything.

        Raises:
            ValidationError: If the worker is not allowed to move the commodity.
        """

        if not self.movement.shipment.hazardous_material:
            return

        # Validation for the primary_worker
        if self.movement.primary_worker:
            if self.movement.primary_worker.profile.endorsements not in [
                WorkerProfile.EndorsementChoices.HAZMAT,
                WorkerProfile.EndorsementChoices.X,
            ]:
                self.errors["primary_worker"] = _(
                    "Worker must be hazmat certified to haul this shipment. Please try again."
                )

            if (
                self.movement.primary_worker.profile.hazmat_expiration_date
                and self.movement.primary_worker.profile.hazmat_expiration_date
                < datetime.date.today()
            ):
                self.errors["primary_worker"] = _(
                    "Worker hazmat certification has expired. Please try again."
                )

        # Validation for the secondary_worker.
        if self.movement.secondary_worker:
            if self.movement.secondary_worker.profile.endorsements not in [
                WorkerProfile.EndorsementChoices.HAZMAT,
                WorkerProfile.EndorsementChoices.X,
            ]:
                self.errors["secondary_worker"] = _(
                    "Worker must be hazmat certified to haul this shipment. Please try again."
                )

            if (
                self.movement.secondary_worker.profile.hazmat_expiration_date
                and self.movement.secondary_worker.profile.hazmat_expiration_date
                < datetime.date.today()
            ):
                self.errors["secondary_worker"] = _(
                    "Worker hazmat certification has expired. Please try again."
                )

    def validate_movement_stop_status(self) -> None:
        """Validate Movement Stop Status

        Validate that the movement status is in progress before setting the
        status to stop.

        Returns:
            None

        Raises:
            ValidationError: Movement is not valid.
        """
        if (
            self.movement.status in [StatusChoices.IN_PROGRESS, StatusChoices.COMPLETED]
            and self.movement.stops.filter(
                status=StatusChoices.NEW, sequence=1
            ).exists()
        ):
            self.errors["status"] = _(
                "Cannot change status to anything other than `NEW` if any of the stops are"
                " not in progress. Please try again."
            )
        if (
            self.movement.status == StatusChoices.NEW
            and self.movement.stops.filter(
                status__in=[StatusChoices.IN_PROGRESS, StatusChoices.COMPLETED]
            ).exists()
        ):
            self.errors["status"] = _(
                "Cannot change status to `NEW` if any of the stops are in progress or"
                " completed. Please try again."
            )

        if (
            self.movement.status == StatusChoices.COMPLETED
            and self.movement.stops.filter(
                status__in=[StatusChoices.NEW, StatusChoices.IN_PROGRESS]
            ).exists()
        ):
            self.errors["status"] = _(
                "Cannot change status to `COMPLETED` if any of the stops are in"
                " progress or new. Please try again."
            )

    def validate_movement_worker(self) -> None:
        """Validate Movement worker

        Require a primary worker and Tractor to set the movement status
        to in progress.

        Returns:
            None

        Raises:
            ValidationError: If the old movement worker is not
            None and the user tries to change the worker.
        """

        if (
            self.movement.status in [StatusChoices.IN_PROGRESS, StatusChoices.COMPLETED]
            and not self.movement.primary_worker
            and not self.movement.tractor
        ):
            self.errors["primary_worker"] = _(
                "Primary worker is required before movement status can be changed to"
                " `In Progress` or `Completed`. Please try again."
            )
            self.errors["tractor"] = _(
                "Tractor is required before movement status can be changed to"
                " `In Progress` or `Completed`. Please try again."
            )

    def validate_worker_tractor_fleet(self) -> None:
        """Validate Worker and tractor are in the same fleet.

        Returns:
            None: This function has no return.
        """

        if (
            self.movement.organization.dispatch_control.tractor_worker_fleet_constraint
            and self.movement.primary_worker
            and self.movement.tractor
            and self.movement.primary_worker.fleet_code_id
            != self.movement.tractor.fleet_code_id
        ):
            self.errors["primary_worker"] = _(
                "The primary worker and tractor must belong to the same fleet to add or update a record. "
                "Please ensure they are part of the same fleet and try again."
            )
            self.errors["tractor"] = _(
                "The primary worker and tractor must belong to the same fleet to add or update a record. "
                "Please ensure they are part of the same fleet and try again."
            )

    def validate_movement_shipment(self) -> None:
        """Validate previous movement is completed before setting current movement in progress

        This method validates the given instance of the Movement model before saving it. Specifically,
        it checks if the instance's status is set to 'IN_PROGRESS' and if there are any previous movements
        with the same shipment that are not completed. If there are any previous movements that are not
        completed, it raises a validation error with a message indicating that the previous movement(s) must
        be completed before the current movement can be set to 'IN_PROGRESS'.

        Raises:
            ValidationError: If the instance status is set to IN_PROGRESS and the previous movement(s) with the same shipment
            are not completed.

        Returns:
            None: This function does not return anything.
        """

        if self.movement.status in [StatusChoices.IN_PROGRESS, StatusChoices.COMPLETED]:
            previous_movements = models.Movement.objects.filter(
                shipment=self.movement.shipment, id__lt=self.movement.id
            )

            for movement in previous_movements:
                if movement.status != StatusChoices.COMPLETED:
                    self.errors["status"] = _(
                        f"The previous movement (ID: {movement.ref_num}) must be completed before"
                        f" this movement can be set to `{self.movement.get_status_display()}`"
                        " Please try again."
                    )

    def validate_voided_movement(self) -> None:
        """Validate voided movement status cannot be changed. If the movement is voided, then
        throw a ValidationError.

        Returns:
            None: This function does not return anything.
        """
        movement = self.movement

        if not movement.exists():
            return None

        if movement.status == StatusChoices.VOIDED:
            self.errors["status"] = _(
                "Cannot update a voided movement. Please contact your administrator."
            )

    def validate_trailer_status_is_available(self) -> None:
        """Validate the status of the trailer is "Available" before assigning it to a movement.
        If not then throw a ValidationError.

        Returns:
            None: This function does not return anything.
        """

        if (
            self.movement.trailer
            and self.movement.trailer.status != AvailabilityChoices.AVAILABLE
        ):
            self.errors["trailer"] = _(
                "Cannot assign a trailer that is not `available`. Please try again."
            )
