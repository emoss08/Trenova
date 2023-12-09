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

import textwrap
import uuid
from typing import Any

from django.core.exceptions import ValidationError
from django.db import models
from django.db.models.functions import Lower
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from utils.models import ChoiceField, GenericModel, StatusChoices
from utils.types import ModelDelete


class Movement(GenericModel):
    """
    Stores movement information related to a :model:`shipment.Shipment`.
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
        editable=False,
        help_text=_("Movement Reference Number"),
    )
    status = ChoiceField(
        _("Status"),
        choices=StatusChoices.choices,
        default=StatusChoices.NEW,
        help_text=_("Status of the Movement"),
    )
    shipment = models.ForeignKey(
        "shipment.Shipment",
        on_delete=models.PROTECT,
        related_name="movements",
        related_query_name="movement",
        verbose_name=_("shipment"),
        help_text=_("Order of the Movement"),
    )
    tractor = models.ForeignKey(
        "equipment.Tractor",
        on_delete=models.PROTECT,
        related_name="movements",
        related_query_name="movement",
        verbose_name=_("Tractor"),
        null=True,
        blank=True,
        help_text=_("Tractor of the Movement"),
    )
    trailer = models.ForeignKey(
        "equipment.Trailer",
        verbose_name=_("Trailer"),
        on_delete=models.PROTECT,
        related_name="movements",
        related_query_name="movement",
        null=True,
        blank=True,
        help_text=_("Trailer associated to the movement"),
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
        db_table = "movement"
        constraints = [
            models.UniqueConstraint(
                Lower("ref_num"),
                "organization",
                name="unique_movement_ref_num_organization",
            )
        ]

    def __str__(self) -> str:
        """String representation of the Movement

        Returns:
            str: String representation of the Movement
        """
        return textwrap.shorten(
            f"Movement {self.ref_num}, Status: {self.status}",
            width=30,
            placeholder="...",
        )

    def save(self, *args: Any, **kwargs: Any) -> None:
        """Save method for the Movement

        Args:
            *args (Any): Arguments
            **kwargs (Any): Keyword Arguments

        Returns:
            None: This function does return anything.
        """
        self.set_tractor_and_workers()
        if not self.ref_num:
            self.ref_num = self.set_reference_number()

        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """Get the absolute url for the Movement

        Returns:
            str: Absolute url for the Movement
        """
        return reverse("movement-detail", kwargs={"pk": self.pk})

    def clean(self) -> None:
        """Movement clean method

        Returns:
            None
        """
        from movements.validation import MovementValidation

        MovementValidation(movement=self)
        super().clean()

    def delete(self, *args: Any, **kwargs: Any) -> ModelDelete:
        """Delete method for the Movement

        Args:
            *args(Any): Arguments
            **kwargs(Any): Keyword Arguments

        Returns:
            ModelDelete: tuple[int, dict[str, int]]
        """
        if self.organization.shipment_control.remove_shipment is False:
            raise ValidationError(
                {
                    "ref_num": _(
                        "Organization does not allow Movement removal. Please contact your administrator."
                    ),
                },
                code="invalid",
            )
        return super().delete(*args, **kwargs)

    def set_reference_number(self) -> str:
        """Generate a unique reference number for a Movement instance.

        This function constructs a reference number by appending to the string 'MOV' a zero-padded
        sequence number determined by the current count of Movement objects plus one.

        This ensures uniqueness in the reference number as it only assigns it if it doesn't exist,
        otherwise it assigns the default reference number "MOV000001".

        Note:
            It's highly recommended to run this inside a transaction where the new Movement instance
            gets created to ensure the count correctly reflects the current total number of instances.

        Returns:
            str: The unique reference number for the new Movement instance.
        """
        code = f"MOV{self.__class__.objects.count() + 1:06d}"
        return (
            "MOV000001"
            if self.__class__.objects.filter(ref_num=code).exists()
            else code
        )

    def set_tractor_and_workers(self) -> None:
        """
        Sets tractor and worker assignments based on certain conditions.
        This function checks the following:
        - If a tractor is assigned, it sets the primary and secondary workers of the tractor to the Movement instance,
        provided that these fields are not already set.
        - If a primary worker is assigned but not a tractor, it sets the primary tractor of the worker to the Movement
         instance.
        This ensures that each Movement instance gets assigned the right tractor and workers.

        Note:
            This function alters the current instance 'self' and might need to save the instance depending on how it's
            used.

        Returns:
            None: This function does not return anything.
        """
        if self.tractor:
            if self.tractor.primary_worker and not self.primary_worker:
                self.primary_worker = self.tractor.primary_worker
            if self.tractor.secondary_worker and not self.secondary_worker:
                self.secondary_worker = self.tractor.secondary_worker

        if self.primary_worker and not self.tractor:
            self.tractor = self.primary_worker.primary_tractor
