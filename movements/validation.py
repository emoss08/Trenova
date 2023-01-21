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

import datetime

from django.core.exceptions import ValidationError
from django.utils import timezone
from django.utils.translation import gettext_lazy as _

from utils.models import StatusChoices
from worker.models import WorkerProfile


class MovementValidation:
    """
    Class to Validate the movement model.
    """

    def __init__(self, *, movement):
        self.movement = movement
        self.validate_regulatory()
        self.validate_worker_commodity()
        self.validate_movement_stop_status()
        self.validate_worker_compare()
        self.validate_movement_worker()

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
            self.movement.primary_worker.profile.license_expiration_date
            and self.movement.primary_worker.profile.license_expiration_date
            < timezone.now().date()
        ):
            raise ValidationError(
                {
                    "primary_worker": _(
                        "Cannot assign a worker with an expired license. Please update the worker's profile and try again."
                    )
                },
                code="invalid",
            )

        if (
            self.movement.primary_worker.profile.physical_due_date
            and self.movement.primary_worker.profile.physical_due_date
            < timezone.now().date()
        ):
            raise ValidationError(
                {
                    "primary_worker": _(
                        "Cannot assign a worker with an expired physical. Please update the worker's profile and try again."
                    )
                },
                code="invalid",
            )

        if (
            self.movement.primary_worker.profile.medical_cert_date
            and self.movement.primary_worker.profile.medical_cert_date
            < timezone.now().date()
        ):
            raise ValidationError(
                {
                    "primary_worker": _(
                        "Cannot assign a worker with an expired medical certificate. Please update the worker's profile and try again."
                    )
                },
                code="invalid",
            )

        if (
            self.movement.primary_worker.profile.medical_cert_date
            and self.movement.primary_worker.profile.medical_cert_date
            < timezone.now().date()
        ):
            raise ValidationError(
                {
                    "primary_worker": _(
                        "Cannot assign a worker with an expired medical certificate. Please update the worker's profile and try again."
                    )
                },
                code="invalid",
            )

        if (
            self.movement.primary_worker.profile.mvr_due_date
            and self.movement.primary_worker.profile.mvr_due_date
            < timezone.now().date()
        ):
            raise ValidationError(
                {
                    "primary_worker": _(
                        "Cannot assign a worker with an expired MVR. Please update the worker's profile and try again."
                    ),
                },
                code="invalid",
            )

        if self.movement.primary_worker.profile.termination_date:
            raise ValidationError(
                {
                    "primary_worker": _(
                        "Cannot assign a terminated worker. Please update the worker's profile and try again."
                    )
                },
                code="invalid",
            )

    def validate_secondary_worker_regulatory(self):
        """Validate primary worker regulatory information.

        Returns:
            None

        Raises:
            ValidationError: if worker regulatory information is invalid.
        """

        if (
            self.movement.secondary_worker.profile.license_expiration_date
            and self.movement.secondary_worker.profile.license_expiration_date
            < timezone.now().date()
        ):
            raise ValidationError(
                {
                    "secondary_worker": _(
                        "Cannot assign a worker with an expired license. Please update the worker's profile and try again."
                    )
                },
                code="invalid",
            )

        if (
            self.movement.secondary_worker.profile.physical_due_date
            and self.movement.secondary_worker.profile.physical_due_date
            < timezone.now().date()
        ):
            raise ValidationError(
                {
                    "secondary_worker": _(
                        "Cannot assign a worker with an expired physical. Please update the worker's profile and try again."
                    )
                },
                code="invalid",
            )

        if (
            self.movement.secondary_worker.profile.medical_cert_date
            and self.movement.secondary_worker.profile.medical_cert_date
            < timezone.now().date()
        ):
            raise ValidationError(
                {
                    "secondary_worker": _(
                        "Cannot assign a worker with an expired medical certificate. Please update the worker's profile and try again."
                    )
                },
                code="invalid",
            )

        if (
            self.movement.secondary_worker.profile.mvr_due_date
            and self.movement.secondary_worker.profile.mvr_due_date
            < timezone.now().date()
        ):
            raise ValidationError(
                {
                    "secondary_worker": _(
                        "Cannot assign a worker with an expired MVR. Please update the worker's profile and try again."
                    ),
                },
                code="invalid",
            )

        if self.movement.secondary_worker.profile.termination_date:
            raise ValidationError(
                {
                    "secondary_worker": _(
                        "Cannot assign a terminated worker. Please update the worker's profile and try again."
                    )
                },
                code="invalid",
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
            raise ValidationError(
                {
                    "primary_worker": _(
                        "Primary worker cannot be the same as secondary worker. Please try again."
                    ),
                },
                code="invalid",
            )

    def validate_worker_commodity(self) -> None:
        """Validate Worker Commodity

        Validate that the assigned worker is allowed to move the commodity.

        Returns:
            None

        Raises:
            ValidationError: If the worker is not allowed to move the commodity.
        """

        if self.movement.order.hazmat:
            # Validation for the primary_worker
            if self.movement.primary_worker:
                if self.movement.primary_worker.profile.endorsements not in [
                    WorkerProfile.EndorsementChoices.HAZMAT,
                    WorkerProfile.EndorsementChoices.X,
                ]:
                    raise ValidationError(
                        {
                            "primary_worker": _(
                                "Worker must be hazmat certified to haul this order. Please try again."
                            ),
                        },
                        code="invalid",
                    )

                if (
                    self.movement.primary_worker.profile.hazmat_expiration_date
                    and self.movement.primary_worker.profile.hazmat_expiration_date
                    < datetime.date.today()
                ):
                    raise ValidationError(
                        {
                            "primary_worker": _(
                                "Worker hazmat certification has expired. Please try again."
                            ),
                        },
                        code="invalid",
                    )

            # Validation for the secondary_worker.
            if self.movement.secondary_worker:
                if self.movement.secondary_worker.profile.endorsements not in [
                    WorkerProfile.EndorsementChoices.HAZMAT,
                    WorkerProfile.EndorsementChoices.X,
                ]:
                    raise ValidationError(
                        {
                            "secondary_worker": _(
                                "Worker must be hazmat certified to haul this order. Please try again."
                            ),
                        },
                        code="invalid",
                    )

                if (
                    self.movement.secondary_worker.profile.hazmat_expiration_date
                    and self.movement.secondary_worker.profile.hazmat_expiration_date
                    < datetime.date.today()
                ):
                    raise ValidationError(
                        {
                            "secondary_worker": _(
                                "Worker hazmat certification has expired. Please try again."
                            ),
                        },
                        code="invalid",
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
            and self.movement.stops.filter(status=StatusChoices.NEW).exists()
        ):
            raise ValidationError(
                {
                    "status": _(
                        "Cannot change status to anything other than `NEW` if any of the stops are not in progress. Please try again."
                    )
                },
                code="invalid",
            )
        elif (
            self.movement.status == StatusChoices.NEW
            and self.movement.stops.filter(
                status__in=[StatusChoices.IN_PROGRESS, StatusChoices.COMPLETED]
            ).exists()
        ):
            raise ValidationError(
                {
                    "status": _(
                        "Cannot change status to `NEW` if any of the stops are in progress or completed. Please try again."
                    )
                },
                code="invalid",
            )

        if (
            self.movement.status == StatusChoices.COMPLETED
            and self.movement.stops.filter(
                status__in=[StatusChoices.NEW, StatusChoices.IN_PROGRESS]
            ).exists()
        ):
            raise ValidationError(
                {
                    "status": _(
                        "Cannot change status to `COMPLETED` if any of the stops are in progress or new. Please try again."
                    )
                },
                code="invalid",
            )

    def validate_movement_worker(self) -> None:
        """Validate Movement worker

        Require a primary worker and equipment to set the movement status
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
            and not self.movement.equipment
        ):
            raise ValidationError(
                {
                    "primary_worker": _(
                        "Primary worker is required before movement status can be changed to `In Progress` or `Completed`. Please try again."
                    ),
                    "equipment": _(
                        "Equipment is required before movement status can be changed to `In Progress` or `Completed`. Please try again."
                    ),
                },
                code="invalid",
            )
