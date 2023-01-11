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
import uuid

from django.core.exceptions import ValidationError
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from dispatch.models import DispatchControl
from utils.models import ChoiceField, GenericModel, StatusChoices
from worker.models import WorkerProfile


class Movement(GenericModel):
    """
    Stores movement information related to a :model:`order.Order`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    ref_num = models.CharField(
        _("Movement Reference Number"),
        max_length=10,
        unique=True,
        editable=False,
        help_text=_("Movement Reference Number"),
    )
    status = ChoiceField(
        _("Status"),
        choices=StatusChoices.choices,
        default=StatusChoices.NEW,
        help_text=_("Status of the Movement"),
    )
    order = models.ForeignKey(
        "order.Order",
        on_delete=models.PROTECT,
        related_name="movements",
        related_query_name="movement",
        verbose_name=_("Order"),
        help_text=_("Order of the Movement"),
    )
    equipment = models.ForeignKey(
        "equipment.Equipment",
        on_delete=models.PROTECT,
        related_name="movements",
        related_query_name="movement",
        verbose_name=_("Equipment"),
        null=True,
        blank=True,
        help_text=_("Equipment of the Movement"),
    )
    primary_worker = models.ForeignKey(
        "worker.Worker",
        on_delete=models.PROTECT,
        related_name="movements",
        related_query_name="movement",
        verbose_name=_("Primary Worker"),
        null=True,
        blank=True,
        help_text=_("Primary Worker of the Movement"),
    )
    secondary_worker = models.ForeignKey(
        "worker.Worker",
        on_delete=models.PROTECT,
        related_name="secondary_movements",
        related_query_name="secondary_movement",
        verbose_name=_("Secondary Worker"),
        null=True,
        blank=True,
        help_text=_("Secondary Worker of the Movement"),
    )

    class Meta:
        """
        Movement Metaclass
        """

        verbose_name = _("Movement")
        verbose_name_plural = _("Movements")

    def __str__(self) -> str:
        """String representation of the Movement

        Returns:
            str: String representation of the Movement
        """
        return f"{self.order} - {self.ref_num}"

    @staticmethod
    def set_ref_number() -> str:
        """Set the Movement Reference Number

        Returns:
            str: Movement Reference Number
        """

        code = f"MOV{Movement.objects.count() + 1:06d}"
        return "MOV000001" if Movement.objects.filter(ref_num=code).exists() else code

    def validate_movement_statuses(self) -> None:
        """Validate Movement status

        If the old movement status is in progress, or completed.
        and the user tries to set it back to NEW, raise an error.

        Returns:
            None

        Raises:
            ValidationError: If the old movement status is in progress, or completed.
        """

        old_status = Movement.objects.get(pk=self.pk).status

        if self.status == StatusChoices.NEW and old_status in [
            StatusChoices.IN_PROGRESS,
            StatusChoices.COMPLETED,
        ]:
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

    def validate_movement_worker(self) -> None:
        """Validate Movement worker

        Require a primary worker and equipment to set the
        movement status to in progress.

        Returns:
            None

        Raises:
            ValidationError: If the old movement worker is not
            None and the user tries to change the worker.
        """

        if (
                self.status == StatusChoices.IN_PROGRESS
                and not self.primary_worker
                and not self.equipment
        ):
            raise ValidationError(
                {
                    "primary_worker": ValidationError(
                        _("Primary worker is required for in progress status."),
                        code="invalid",
                    ),
                    "equipment": ValidationError(
                        _("Equipment is required for in progress status."),
                        code="invalid",
                    ),
                }
            )

    def validate_worker_compare(self) -> None:
        """Validate Worker Comparison

        Validate that the workers do not match when creating
        movement.

        Returns:
            None

        Raises:
            ValidationError: If the workers are the same.
        """

        if (
                self.primary_worker
                and self.secondary_worker
                and self.primary_worker == self.secondary_worker
        ):
            raise ValidationError(
                {
                    "primary_worker": ValidationError(
                        _("Primary worker cannot be the same as secondary worker."),
                        code="invalid",
                    ),
                    "secondary_worker": ValidationError(
                        _("Primary and secondary workers cannot be the same."),
                        code="invalid",
                    ),
                }
            )

    def validate_worker_commodity(self) -> None:
        """Validate Worker Commodity

        Validate that the assigned worker is allowed to move the commodity.

        Returns:
            None

        Raises:
            ValidationError: If the worker is not allowed to move the commodity.
        """

        if self.order.hazmat and self.primary_worker:

            if self.primary_worker.profile.endorsements not in [
                WorkerProfile.EndorsementChoices.HAZMAT,
                WorkerProfile.EndorsementChoices.X,
            ]:
                raise ValidationError(
                    {
                        "primary_worker": ValidationError(
                            _(
                                "Primary worker must be hazmat certified to haul this order."
                            ),
                            code="invalid",
                        ),
                    }
                )

            if (
                    self.primary_worker.profile.hazmat_expiration_date
                    and self.primary_worker.profile.hazmat_expiration_date
                    < datetime.date.today()
            ):
                raise ValidationError(
                    {
                        "primary_worker": ValidationError(
                            _("Primary worker hazmat certification has expired."),
                            code="invalid",
                        ),
                    }
                )

    def validate_primary_worker_regulatory(self) -> None:
        """Validate Worker Regulatory

        Validate that the workers are regulatory when creating
        movement.

        Returns:
            None

        Raises:
            ValidationError: If the workers are not regulatory.
        """

        if self.primary_worker:
            dispatch_control = DispatchControl.objects.get(
                organization=self.primary_worker.organization
            )
            if dispatch_control.regulatory_check:
                if (
                        self.primary_worker.profile.license_expiration_date
                        and self.primary_worker.profile.license_expiration_date
                        < datetime.date.today()
                ):
                    raise ValidationError(
                        {
                            "primary_worker": ValidationError(
                                _("Primary worker license is expired."),
                                code="invalid",
                            )
                        }
                    )

                if (
                        self.primary_worker.profile.physical_due_date
                        and self.primary_worker.profile.physical_due_date
                        < datetime.date.today()
                ):
                    raise ValidationError(
                        {
                            "primary_worker": ValidationError(
                                _("Primary worker physical is expired."),
                                code="invalid",
                            )
                        }
                    )

                if (
                        self.primary_worker.profile.medical_cert_date
                        and self.primary_worker.profile.medical_cert_date
                        < datetime.date.today()
                ):
                    raise ValidationError(
                        {
                            "primary_worker": ValidationError(
                                _("Primary worker medical certificate is expired."),
                                code="invalid",
                            )
                        }
                    )

                if (
                        self.primary_worker.profile.mvr_due_date
                        and self.primary_worker.profile.mvr_due_date < datetime.date.today()
                ):
                    raise ValidationError(
                        {
                            "primary_worker": ValidationError(
                                _("Primary worker MVR is expired."),
                                code="invalid",
                            )
                        }
                    )
                if self.primary_worker.profile.termination_date:
                    raise ValidationError(
                        {
                            "primary_worker": ValidationError(
                                _("Primary worker is terminated."),
                                code="invalid",
                            )
                        }
                    )

    def validate_movement_stop_status(self) -> None:
        """Validate Movement Stop Status

        Validate that the movement status is in progress
        before setting the status to stop.

        Returns:
            None

        Raises:
            ValidationError: Movement is not valid.
        """
        if (
                self.status == StatusChoices.IN_PROGRESS
                and self.stops.filter(status=StatusChoices.NEW).exists()
        ):
            raise ValidationError(
                {
                    "status": ValidationError(
                        _(
                            "Cannot change status to in progress if any of the"
                            " stops are not in progress."
                        )
                    )
                }
            )
        elif (
                self.status == StatusChoices.NEW
                and self.stops.filter(status=StatusChoices.IN_PROGRESS).exists()
        ):
            raise ValidationError(
                {
                    "status": ValidationError(
                        _(
                            "Cannot change status to available if"
                            " the movement stops are in progress"
                        )
                    )
                }
            )

        if (
                self.status == StatusChoices.COMPLETED
                and self.stops.filter(
            status__in=[StatusChoices.NEW, StatusChoices.IN_PROGRESS]
        ).exists()
        ):
            raise ValidationError(
                {
                    "status": ValidationError(
                        _(
                            "Cannot change status to completed if any of the stops are"
                            " in progress or new."
                        ),
                        code="invalid",
                    )
                }
            )

    def clean(self) -> None:
        """Stop clean method

        Returns:
            None
        """
        self.validate_primary_worker_regulatory()
        self.validate_movement_statuses()
        self.validate_movement_worker()
        self.validate_worker_compare()
        self.validate_movement_stop_status()
        self.validate_worker_commodity()

    def get_absolute_url(self) -> str:
        """Get the absolute url for the Movement

        Returns:
            str: Absolute url for the Movement
        """
        return reverse("movement-detail", kwargs={"pk": self.pk})
