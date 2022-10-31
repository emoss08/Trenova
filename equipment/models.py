# -*- coding: utf-8 -*-
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

from typing import final

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from core.models import GenericModel


class EquipmentType(GenericModel):
    """
    Equipment Type Model Fields
    """
    name = models.CharField(
        _("Name"),
        max_length=50,
        unique=True,
        help_text=_("Name of the equipment type."),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Description of the equipment type."),
    )

    class Meta:
        """
        Equipment Type Model Metaclass
        """
        verbose_name = _("Equipment Type")
        verbose_name_plural = _("Equipment Types")
        ordering: list[str] = ["-name"]

    def __str__(self) -> str:
        """Equipment Type string representation

        Returns:
            str: String representation of the Equipment Type Model
        """
        return self.name

    def get_absolute_url(self) -> str:
        """Equipment Type absolute URL

        Returns:
            str: Absolute URL of the Equipment Type Model
        """
        return reverse("equipment:view-equipment-type", kwargs={"pk": self.pk})


class EquipmentTypeDetail(GenericModel):
    """
    Equipment Type Detail Model Fields
    """

    @final
    class EquipmentClassChoices(models.TextChoices):
        """
        Equipment Class Choices
        """

        UNDEFINED = "undefined", _("UNDEFINED")
        CAR = "car", _("Car")
        VAN = "van", _("Van")
        PICKUP = "pickup", _("Pickup")
        WALKIN = "walk-in", _("Walk-In")
        STRAIGHT = "straight", _("Straight Truck")
        TRACTOR = "tractor", _("Tractor")
        TRAILER = "trailer", _("Trailer")

    equipment_type = models.OneToOneField(
        EquipmentType,
        on_delete=models.CASCADE,
        related_name="equipment_type_details",
        related_query_name="equipment_type_detail",
        verbose_name=_("Equipment Type"),
    )
    equipment_class = models.CharField(
        _("Equipment Class"),
        max_length=50,
        choices=EquipmentClassChoices.choices,
        default=EquipmentClassChoices.UNDEFINED,
        help_text=_("Class of the equipment type."),
    )
    fixed_cost = models.DecimalField(
        _("Fixed Cost"),
        max_digits=10,
        decimal_places=4,
        default=0.0000,
        help_text=_("Fixed cost of the equipment type."),
    )
    variable_cost = models.DecimalField(
        _("Variable Cost"),
        max_digits=10,
        decimal_places=4,
        default=0.0000,
        help_text=_("Variable cost of the equipment type."),
    )
    height = models.DecimalField(
        _("Height (Inches)"),
        max_digits=10,
        decimal_places=4,
        default=0.0000,
        help_text=_("Height of the equipment type."),
    )
    length = models.DecimalField(
        _("Length (Inches)"),
        max_digits=10,
        decimal_places=4,
        default=0.0000,
        help_text=_("Length of the equipment type."),
    )
    width = models.DecimalField(
        _("Width (Inches)"),
        max_digits=10,
        decimal_places=4,
        default=0.0000,
        help_text=_("Width of the equipment type."),
    )
    weight = models.DecimalField(
        _("Weight (Pounds)"),
        max_digits=10,
        decimal_places=4,
        default=0.0000,
        help_text=_("Weight of the equipment type."),
    )
    idling_fuel_usage = models.DecimalField(
        _("Idling Fuel Usage (Gallons Per Hour)"),
        max_digits=10,
        decimal_places=4,
        default=0.0000,
        help_text=_("Idling fuel usage of the equipment type."),
    )
    exempt_from_tolls = models.BooleanField(
        _("Exempt From Tolls"),
        default=False,
        help_text=_("Exempt from tolls of the equipment type."),
    )

    class Meta:
        """
        Equipment Type Detail Model Metaclass
        """
        verbose_name = _("Equipment Type Detail")
        verbose_name_plural = _("Equipment Type Details")
        ordering: list[str] = ["-equipment_type"]

    def __str__(self) -> str:
        """Equipment Type Detail string representation

        Returns:
            str: String representation of the Equipment Type Detail Model
        """
        return self.equipment_type.name

    def get_absolute_url(self) -> str:
        """Equipment Type Detail absolute URL

        Returns:
            str: Absolute URL of the Equipment Type Detail Model
        """
        return reverse("equipment:view-equipment-type-detail", kwargs={"pk": self.pk})
