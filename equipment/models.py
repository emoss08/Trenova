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
from typing import final

from django.core.exceptions import ValidationError
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from localflavor.us.models import USStateField

from equipment.validators import us_vin_number_validator
from utils.models import ChoiceField, GenericModel
from worker.models import Worker


class EquipmentType(GenericModel):
    """
    Stores the equipment type information that can later be used to
    create :model:`equipment.Equipment` objects.
    """

    id = models.CharField(
        _("ID"),
        max_length=50,
        primary_key=True,
        help_text=_("ID of the equipment type"),
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
        ordering: list[str] = ["-id"]

    def __str__(self) -> str:
        """Equipment Type string representation

        Returns:
            str: String representation of the Equipment Type Model
        """
        return textwrap.wrap(self.id, 50)[0]

    def get_absolute_url(self) -> str:
        """Equipment Type absolute URL

        Returns:
            str: Absolute URL of the Equipment Type Model
        """
        return reverse("equipment:view-equipment-type", kwargs={"pk": self.pk})


class EquipmentTypeDetail(GenericModel):
    """
    Stores detailed information about a :model:`equipment.EquipmentType`.
    """

    @final
    class EquipmentClassChoices(models.TextChoices):
        """
        Equipment Class Choices
        """

        UNDEFINED = "UNDEFINED", _("UNDEFINED")
        CAR = "CAR", _("Car")
        VAN = "VAN", _("Van")
        PICKUP = "PICKUP", _("Pickup")
        WALK_IN = "WALK-IN", _("Walk-In")
        STRAIGHT = "STRAIGHT", _("Straight Truck")
        TRACTOR = "TRACTOR", _("Tractor")
        TRAILER = "TRAILER", _("Trailer")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    equipment_type = models.OneToOneField(
        EquipmentType,
        on_delete=models.CASCADE,
        related_name="equipment_type_details",
        related_query_name="equipment_type_detail",
        verbose_name=_("Equipment Type"),
    )
    equipment_class = ChoiceField(
        _("Equipment Class"),
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
        return textwrap.wrap(self.equipment_type.id, 50)[0]

    def update_details(self, **kwargs) -> None:
        """Updates the Equipment Type Detail Model

        Args:
            **kwargs: Keyword arguments to update the model
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()

    def get_absolute_url(self) -> str:
        """Equipment Type Detail absolute URL

        Returns:
            str: Absolute URL of the Equipment Type Detail Model
        """
        return reverse("equipment:view-equipment-type-detail", kwargs={"pk": self.pk})


class EquipmentManufacturer(GenericModel):
    """
    Stores the equipment manufacturer information that can later be used to
    create :model:`equipment.Equipment` objects.
    """

    id = models.CharField(
        _("ID"),
        max_length=50,
        primary_key=True,
        help_text=_("ID of the equipment manufacturer."),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Description of the equipment manufacturer."),
    )

    class Meta:
        """
        Equipment Manufacturer Model Metaclass
        """

        verbose_name = _("Equipment Manufacturer")
        verbose_name_plural = _("Equipment Manufacturers")
        ordering: list[str] = ["-id"]

    def __str__(self) -> str:
        """Equipment Manufacturer string representation

        Returns:
            str: String representation of the Equipment Manufacturer Model
        """
        return textwrap.wrap(self.id, 50)[0]

    def get_absolute_url(self) -> str:
        """Equipment Manufacturer absolute URL

        Returns:
            str: Absolute URL of the Equipment Manufacturer Model
        """
        return reverse("equipment:view-equipment-manufacturer", kwargs={"pk": self.pk})


class Equipment(GenericModel):
    """
    Stores information about a piece of equipment for a :model:`organization.Organization`.
    """

    @final
    class AuxiliaryPowerUnitTypeChoices(models.TextChoices):
        """
        Auxiliary Power Unit Type Choices
        """

        NONE = "none", _("None")
        APU = "apu", _("APU")
        BUNK = "bunk-heater", _("Bunk Heater")
        HYBRID = "hybrid", _("Hybrid")

    id = models.CharField(
        _("ID"),
        max_length=50,
        primary_key=True,
        help_text=_("ID of the equipment."),
    )
    equipment_type = models.ForeignKey(
        EquipmentType,
        on_delete=models.CASCADE,
        related_name="equipment",
        related_query_name="equipment",
        verbose_name=_("Equipment Type"),
    )
    is_active = models.BooleanField(
        _("Active"),
        default=True,
        help_text=_("Whether the Equipment is active or not."),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Description of the equipment."),
    )
    license_plate_number = models.CharField(
        _("License Plate Number"),
        max_length=50,
        blank=True,
        help_text=_("License plate number of the equipment."),
    )
    vin_number = models.CharField(
        _("VIN Number"),
        max_length=17,
        blank=True,
        help_text=_("VIN number of the equipment."),
        validators=[us_vin_number_validator],
    )
    odometer = models.PositiveIntegerField(
        _("Odometer"),
        default=0,
        help_text=_("Odometer of the equipment."),
    )
    engine_hours = models.PositiveIntegerField(
        _("Engine Hours"),
        default=0,
        help_text=_("Engine hours of the equipment."),
    )
    manufacturer = models.ForeignKey(
        EquipmentManufacturer,
        on_delete=models.CASCADE,
        related_name="equipments",
        related_query_name="equipment",
        verbose_name=_("Manufacturer"),
        blank=True,
        null=True,
    )
    manufactured_date = models.DateField(
        _("Manufactured Date"),
        blank=True,
        null=True,
        help_text=_("Manufactured date of the equipment."),
    )
    model = models.CharField(
        _("Model"),
        max_length=50,
        blank=True,
        help_text=_("Model of the equipment."),
    )
    model_year = models.PositiveIntegerField(
        _("Model Year"),
        null=True,
        blank=True,
        help_text=_("Model year of the equipment."),
    )
    state = USStateField(
        _("State"),
        blank=True,
        null=True,
        help_text=_("State of the equipment."),
    )
    leased = models.BooleanField(
        _("Leased"),
        default=False,
        help_text=_("Leased of the equipment."),
    )
    leased_date = models.DateField(
        _("Leased Date"),
        blank=True,
        null=True,
        help_text=_("Leased date of the equipment."),
    )
    primary_worker = models.OneToOneField(
        Worker,
        on_delete=models.SET_NULL,
        related_name="primary_equipment",
        related_query_name="primary_equipment",
        verbose_name=_("Primary Worker"),
        blank=True,
        null=True,
    )
    secondary_worker = models.OneToOneField(
        Worker,
        on_delete=models.SET_NULL,
        related_name="secondary_equipment",
        related_query_name="secondary_equipment",
        verbose_name=_("Secondary Worker"),
        blank=True,
        null=True,
    )

    # Advanced Options for the Equipment
    hos_exempt = models.BooleanField(
        _("HOS Exempt"),
        default=False,
        help_text=_("HOS exempt of the equipment."),
    )
    aux_power_unit_type = ChoiceField(
        _("Auxiliary Power Unit Type"),
        choices=AuxiliaryPowerUnitTypeChoices.choices,
        default=AuxiliaryPowerUnitTypeChoices.NONE,
        help_text=_("Auxiliary power unit type of the equipment."),
    )
    fuel_draw_capacity = models.PositiveIntegerField(
        _("Fuel Draw Capacity"),
        default=0,
        help_text=_("Fuel draw capacity of the equipment."),
    )
    num_of_axles = models.PositiveIntegerField(
        _("Number of Axles"),
        default=0,
        help_text=_("Number of axles of the equipment."),
    )
    transmission_manufacturer = models.CharField(
        _("Transmission Manufacturer"),
        max_length=50,
        blank=True,
        help_text=_("Transmission manufacturer of the equipment."),
    )
    transmission_type = models.CharField(
        _("Transmission Type"),
        max_length=50,
        blank=True,
        help_text=_("Transmission type of the equipment."),
    )
    has_berth = models.BooleanField(
        _("Has Berth"),
        default=False,
        help_text=_("Equipment has Sleeper Berth."),
    )
    has_electronic_engine = models.BooleanField(
        _("Has Electronic Engine"),
        default=False,
        help_text=_("Equipment has Electronic Engine."),
    )
    highway_use_tax = models.BooleanField(
        _("Highway Use Tax"),
        default=False,
        help_text=_("Equipment has Highway Use Tax."),
    )
    owner_operated = models.BooleanField(
        _("Owner Operated"),
        default=False,
        help_text=_("Equipment is Owner Operated."),
    )
    ifta_qualified = models.BooleanField(
        _("IFTA Qualified"),
        default=False,
        help_text=_("Equipment is IFTA Qualified."),
    )

    class Meta:
        """
        Equipment Model Metaclass
        """

        verbose_name = _("Equipment")
        verbose_name_plural = _("Equipment")
        ordering: list[str] = ["id"]

    def __str__(self) -> str:
        """Equipment string representation

        Returns:
            str: String representation of the Equipment Model
        """
        return textwrap.wrap(self.id, 50)[0]

    def clean(self) -> None:
        """Equipment Model clean method

        Raises:
            ValidationError: If the Equipment is leased and the leased date is not set
        """
        if self.leased and not self.leased_date:
            raise ValidationError(
                {
                    "leased_date": _(
                        "Leased date must be set if the equipment is leased."
                    )
                }
            )

        if self.primary_worker and self.secondary_worker:
            if self.primary_worker == self.secondary_worker:
                raise ValidationError(
                    {
                        "primary_worker": _(
                            "Primary worker and secondary worker cannot be the same."
                        )
                    }
                )

    def get_absolute_url(self) -> str:
        """Equipment absolute URL

        Returns:
            str: Absolute URL of the Equipment Model
        """
        return reverse("equipment:view-equipment", kwargs={"pk": self.pk})


class EquipmentMaintenancePlan(GenericModel):
    """
    Stores the maintenance plan information related to
    `equipment.EquipmentType` model.
    """

    id = models.CharField(
        _("ID"),
        max_length=50,
        primary_key=True,
        help_text=_("ID of the equipment maintenance plan."),
    )
    equipment_types = models.ManyToManyField(
        EquipmentType,
        related_name="maintenance_plan",
        related_query_name="maintenance_plans",
        verbose_name=_("Equipment Types"),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Description of the equipment maintenance plan."),
    )
    by_distance = models.BooleanField(
        _("By Distance"),
        default=False,
        help_text=_("Maintenance plan is by distance."),
    )
    by_time = models.BooleanField(
        _("By Time"),
        default=False,
        help_text=_("Maintenance plan is by time."),
    )
    by_engine_hours = models.BooleanField(
        _("By Engine Hours"),
        default=False,
        help_text=_("Maintenance plan is by engine hours."),
    )
    miles = models.PositiveIntegerField(
        _("Miles"),
        default=0,
        help_text=_("Miles of the equipment maintenance plan."),
    )
    months = models.PositiveIntegerField(
        _("Months"),
        default=0,
        help_text=_("Months of the equipment maintenance plan."),
    )
    engine_hours = models.PositiveIntegerField(
        _("Engine Hours"),
        default=0,
        help_text=_("Engine hours of the equipment maintenance plan."),
    )

    class Meta:
        """
        EquipmentMaintenancePlan Model Metaclass
        """

        verbose_name = _("Equipment Maintenance Plan")
        verbose_name_plural = _("Equipment Maintenance Plans")
        ordering: list[str] = ["id"]

    def __str__(self) -> str:
        """Equipment Maintenance Plan string representation

        Returns:
            str: String representation of the EquipmentMaintenancePlan Model
        """
        return textwrap.wrap(self.id, 50)[0]

    def clean(self) -> None:
        """Equipment Maintenance Plan clean method

        Raises:
            ValidationError: Validation Errors for the EquipmentMaintenancePlan Model
        """
        if not self.by_distance and not self.by_time and not self.by_engine_hours:
            raise ValidationError(
                {
                    "by_distance": _(
                        "At least one of the fields must be checked: "
                        "By Distance, By Time, By Engine Hours."
                    )
                }
            )

        if self.by_distance and not self.miles:
            raise ValidationError(
                {
                    "miles": _(
                        "Miles must be set if the maintenance plan is by distance."
                    )
                }
            )

        if self.by_time and not self.months:
            raise ValidationError(
                {"months": _("Months must be set if the maintenance plan is by time.")}
            )

        if self.by_engine_hours and not self.engine_hours:
            raise ValidationError(
                {
                    "engine_hours": _(
                        "Engine hours must be set if the maintenance plan is by engine hours."
                    )
                }
            )

        super().clean()

    def get_absolute_url(self) -> str:
        """Equipment Maintenance Plan absolute URL

        Returns:
            str: Absolute URL of the EquipmentMaintenancePlan Model
        """
        return reverse(
            "equipment:view-equipment-maintenance-plan", kwargs={"pk": self.pk}
        )
