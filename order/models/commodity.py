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

# THIS FILE IS A FUCKING NIGHTMARE BUT PYTHON & FUCKING DJANGO!

from __future__ import annotations

import textwrap
from typing import Any, final

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from utils.models import ChoiceField, GenericModel


class Commodity(GenericModel):
    """
    Commodity Model Fields
    """

    @final
    class UnitOfMeasureChoices(models.TextChoices):
        """
        Unit of Measure choices for Commodity model
        """

        PALLET = "PALLET", _("Pallet")
        TOTE = "TOTE", _("Tote")
        DRUM = "DRUM", _("Drum")
        CYLINDER = "CYLINDER", _("Cylinder")
        CASE = "CASE", _("Case")
        AMPULE = "AMPULE", _("Ampule")
        BAG = "BAG", _("Bag")
        BOTTLE = "BOTTLE", _("Bottle")
        PAIL = "PAIL", _("Pail")
        PIECES = "PIECES", _("Pieces")
        ISO_TANK = "ISO_TANK", _("ISO Tank")

    name = models.CharField(
        _("Name"),
        max_length=255,
        unique=True,
        help_text=_("Name of the Commodity"),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Description of the Commodity"),
    )
    min_temp = models.DecimalField(
        _("Minimum Temperature"),
        max_digits=10,
        decimal_places=2,
        help_text=_("Minimum Temperature of the Commodity"),
        null=True,
        blank=True,
    )
    max_temp = models.DecimalField(
        _("Maximum Temperature"),
        max_digits=10,
        decimal_places=2,
        help_text=_("Maximum Temperature of the Commodity"),
        null=True,
        blank=True,
    )
    set_point_temp = models.DecimalField(
        _("Set Point Temperature"),
        max_digits=10,
        decimal_places=2,
        help_text=_("Set Point Temperature of the Commodity"),
        null=True,
        blank=True,
    )
    unit_of_measure = ChoiceField(
        _("Unit of Measure"),
        choices=UnitOfMeasureChoices.choices,
        help_text=_("Unit of Measure of the Commodity"),
        blank=True,
    )
    hazmat = models.ForeignKey(
        "order.HazardousMaterial",
        on_delete=models.PROTECT,
        verbose_name=_("Hazardous Material"),
        help_text=_("Hazardous Material of the Commodity"),
        null=True,
        blank=True,
    )
    is_hazmat = models.BooleanField(
        _("Is Hazardous Material"),
        default=False,
        help_text=_("Is the Commodity a Hazardous Material"),
    )

    class Meta:
        """
        Commodity Metaclass
        """

        verbose_name = _("Commodity")
        verbose_name_plural = _("Commodities")
        ordering: list[str] = ["name"]

    def __str__(self) -> str:
        """Commodity String Representation

        Returns:
            str: Commodity Name
        """
        return textwrap.wrap(self.name, 50)[0]

    def save(self, *args: Any, **kwargs: Any) -> None:
        """Save Commodity

        Args:
            *args (Any): Variable length argument list.
            **kwargs (Any): Keyword Arguments

        Returns:
            None
        """
        if self.hazmat:
            self.is_hazmat = True
        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """Commodity Absolute URL

        Returns:
            str: Commodity Absolute URL
        """
        return reverse("order:commodity_detail", kwargs={"pk": self.pk})
