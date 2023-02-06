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

import textwrap
import uuid

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from django_lifecycle import LifecycleModelMixin, hook, BEFORE_SAVE, AFTER_SAVE, AFTER_CREATE, BEFORE_CREATE

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
        return textwrap.shorten(
            f"{self.order} - {self.ref_num}", width=30, placeholder="..."
        )

    def clean(self) -> None:
        """Stop clean method

        Returns:
            None
        """
        MovementValidation(movement=self)

    @hook(AFTER_CREATE) # type: ignore
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


    @hook(BEFORE_CREATE) # type: ignore
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
