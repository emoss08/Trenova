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

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from django_lifecycle import AFTER_CREATE, BEFORE_CREATE, LifecycleModelMixin, hook

from movements.validation import MovementValidation
from utils.models import ChoiceField, GenericModel, StatusChoices


class Movement(LifecycleModelMixin, GenericModel):
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
                fields=["ref_num", "organization"],
                name="unique_movement_ref_num_organization",
            )
        ]

    def __str__(self) -> str:
        """String representation of the Movement

        Returns:
            str: String representation of the Movement
        """
        return textwrap.shorten(
            f"{self.status} - {self.ref_num}", width=30, placeholder="..."
        )

    def clean(self) -> None:
        """Stop clean method

        Returns:
            None
        """
        MovementValidation(movement=self)

    @hook(AFTER_CREATE)  # type: ignore
    def generate_initial_stops_after_create(self) -> None:
        """Generate initial movements stops.

        This hook should only be fired if the first movement is being added to the order.
        Its purpose is to create the initial stops for the movement, by taking the origin
        and destination from the order. This is done by calling the StopService. This
        service will then create the stops and sequence them.

        Returns:
            None
        """
        from stops.services.generation import StopService

        if self.order.status == StatusChoices.NEW and self.order.movements.count() == 1:
            StopService.create_initial_stops(movement=self, order=self.order)

    @hook(BEFORE_CREATE)  # type: ignore
    def generate_ref_num_before_create(self) -> None:
        """Generate the ref_num before saving the Movement

        Returns:
            None
        """
        from movements.services.generation import MovementService

        if not self.ref_num:
            self.ref_num = MovementService.set_ref_number()

    def get_absolute_url(self) -> str:
        """Get the absolute url for the Movement

        Returns:
            str: Absolute url for the Movement
        """
        return reverse("movement-detail", kwargs={"pk": self.pk})
