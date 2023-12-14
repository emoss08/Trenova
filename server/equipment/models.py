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
import typing
import uuid

from django.core.exceptions import ValidationError
from django.db import models
from django.db.models.functions import Lower
from django.urls import reverse
from django.utils import timezone
from django.utils.translation import gettext_lazy as _

from equipment.validators import us_vin_number_validator
from utils.models import ChoiceField, GenericModel, PrimaryStatusChoices
from worker.models import Worker


@typing.final
class AvailabilityChoices(models.TextChoices):
    """
    Availability Choices
    """

    AVAILABLE = "A", _("Available")
    OUT_OF_SERVICE = "OOS", _("Out of Service")
    AT_MAINTENANCE = "AM", _("At Maintenance")
    SOLD = "S", _("Sold")
    LOST = "L", _("Lost")


class EquipmentType(GenericModel):
    """
    Stores the equipment type information that can later be used to create
    :model:`equipment.Equipment` objects.
    """

    @typing.final
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
    status = ChoiceField(
        _("Status"),
        choices=PrimaryStatusChoices.choices,
        help_text=_("Status of the Customer."),
        default=PrimaryStatusChoices.ACTIVE,
    )
    name = models.CharField(
        _("Name"),
        max_length=50,
        help_text=_("Name of the equipment type"),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Description of the equipment type."),
    )
    cost_per_mile = models.DecimalField(
        verbose_name=_("Cost Per Mile"),
        max_digits=10,
        decimal_places=2,
        default=0.00,
        help_text=_("Cost per mile of the equipment type."),
        blank=True,
        null=True,
    )
    # Equipment Type Details
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
        blank=True,
        null=True,
    )
    variable_cost = models.DecimalField(
        _("Variable Cost"),
        max_digits=10,
        decimal_places=4,
        default=0.0000,
        help_text=_("Variable cost of the equipment type."),
        blank=True,
        null=True,
    )
    height = models.DecimalField(
        _("Height (Inches)"),
        max_digits=10,
        decimal_places=4,
        default=0.0000,
        help_text=_("Height of the equipment type."),
        blank=True,
        null=True,
    )
    length = models.DecimalField(
        _("Length (Inches)"),
        max_digits=10,
        decimal_places=4,
        default=0.0000,
        help_text=_("Length of the equipment type."),
        blank=True,
        null=True,
    )
    width = models.DecimalField(
        _("Width (Inches)"),
        max_digits=10,
        decimal_places=4,
        default=0.0000,
        help_text=_("Width of the equipment type."),
        blank=True,
        null=True,
    )
    weight = models.DecimalField(
        _("Weight (Pounds)"),
        max_digits=10,
        decimal_places=4,
        default=0.0000,
        help_text=_("Weight of the equipment type."),
        blank=True,
        null=True,
    )
    idling_fuel_usage = models.DecimalField(
        _("Idling Fuel Usage (Gallons Per Hour)"),
        max_digits=10,
        decimal_places=4,
        default=0.0000,
        help_text=_("Idling fuel usage of the equipment type."),
        blank=True,
        null=True,
    )
    exempt_from_tolls = models.BooleanField(
        _("Exempt From Tolls"),
        default=False,
        help_text=_("Indicates if the equipment type is exempt from tolls."),
    )

    class Meta:
        """
        Equipment Type Model Metaclass
        """

        verbose_name = _("Equipment Type")
        verbose_name_plural = _("Equipment Types")
        db_table = "equipment_type"
        constraints = [
            models.UniqueConstraint(
                Lower("name"),
                "organization",
                name="unique_equipment_type_name_organization",
            )
        ]

    def __str__(self) -> str:
        """Equipment Type string representation

        Returns:
            str: String representation of the Equipment Type Model
        """
        return textwrap.shorten(self.name, width=40, placeholder="...")

    def get_absolute_url(self) -> str:
        """Equipment Type absolute URL

        Returns:
            str: Absolute URL of the Equipment Type Model
        """
        return reverse("equipment-types-detail", kwargs={"pk": self.pk})


class EquipmentManufacturer(GenericModel):
    """
    Stores the equipment manufacturer information that can later be used to create
    :model:`equipment.Equipment` objects.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    status = ChoiceField(
        _("Status"),
        choices=PrimaryStatusChoices.choices,
        help_text=_("Status of the equipment manufacturer."),
        default=PrimaryStatusChoices.ACTIVE,
    )
    name = models.CharField(
        _("Name"),
        max_length=50,
        help_text=_("Name of the equipment manufacturer."),
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
        db_table = "equipment_manufacturer"
        db_table_comment = "Stores the equipment manufacturer information that can later be used to create equipment objects."
        constraints = [
            models.UniqueConstraint(
                Lower("name"),
                "organization",
                name="unique_equipment_manufacturer_organization",
            )
        ]

    def __str__(self) -> str:
        """Equipment Manufacturer string representation

        Returns:
            str: String representation of the Equipment Manufacturer Model
        """
        return textwrap.shorten(self.name, width=40, placeholder="...")

    def get_absolute_url(self) -> str:
        """Equipment Manufacturer absolute URL

        Returns:
            str: Absolute URL of the Equipment Manufacturer Model
        """
        return reverse("equipment-manufacturers-detail", kwargs={"pk": self.pk})


class Tractor(GenericModel):
    """
    Stores information about a piece of Tractor for a :model:`organization.Organization`.
    """

    @typing.final
    class AuxiliaryPowerUnitTypeChoices(models.TextChoices):
        """
        Auxiliary Power Unit Type Choices
        """

        NONE = "N", _("None")
        APU = "APU", _("APU")
        BUNK = "BH", _("Bunk Heater")
        HYBRID = "H", _("Hybrid")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    code = models.CharField(
        _("Code"),
        max_length=50,
        help_text=_("Code of the tractor."),
    )
    equipment_type = models.ForeignKey(
        EquipmentType,
        on_delete=models.CASCADE,
        related_name="tractor",
        related_query_name="tractor",
        verbose_name=_("Equipment Type"),
    )
    status = ChoiceField(
        _("Status"),
        choices=AvailabilityChoices.choices,
        help_text=_("Status of the trailer."),
        default=AvailabilityChoices.AVAILABLE,
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Description of the tractor."),
    )
    license_plate_number = models.CharField(
        _("License Plate Number"),
        max_length=50,
        blank=True,
        help_text=_("License plate number of the tractor."),
    )
    vin_number = models.CharField(
        _("VIN Number"),
        max_length=17,
        blank=True,
        help_text=_("VIN number of the tractor."),
        validators=[us_vin_number_validator],
    )
    manufacturer = models.ForeignKey(
        EquipmentManufacturer,
        on_delete=models.CASCADE,
        related_name="tractor",
        related_query_name="tractor",
        verbose_name=_("Manufacturer"),
        blank=True,
        null=True,
    )
    manufactured_date = models.DateField(
        _("Manufactured Date"),
        blank=True,
        null=True,
        help_text=_("Manufactured date of the tractor."),
    )
    model = models.CharField(
        _("Model"),
        max_length=50,
        blank=True,
        help_text=_("Model of the tractor."),
    )
    year = models.PositiveIntegerField(
        _("Model Year"),
        null=True,
        blank=True,
        help_text=_("Model year of the tractor."),
    )
    state = models.CharField(
        _("State"),
        max_length=5,
        blank=True,
        help_text=_("State"),
    )
    leased = models.BooleanField(
        _("Leased"),
        default=False,
        help_text=_("Leased of the tractor."),
    )
    leased_date = models.DateField(
        _("Leased Date"),
        blank=True,
        null=True,
        help_text=_("Leased date of the tractor."),
    )
    primary_worker = models.OneToOneField(
        Worker,
        on_delete=models.SET_NULL,
        related_name="primary_tractor",
        related_query_name="primary_tractor",
        verbose_name=_("Primary Worker"),
        blank=True,
        null=True,
    )
    secondary_worker = models.OneToOneField(
        Worker,
        on_delete=models.SET_NULL,
        related_name="secondary_tractor",
        related_query_name="secondary_tractor",
        verbose_name=_("Secondary Worker"),
        blank=True,
        null=True,
    )

    # Advanced Options for the Equipment
    hos_exempt = models.BooleanField(
        _("HOS Exempt"),
        default=False,
        help_text=_("HOS exempt of the tractor."),
    )
    aux_power_unit_type = ChoiceField(
        _("Auxiliary Power Unit Type"),
        choices=AuxiliaryPowerUnitTypeChoices.choices,
        default=AuxiliaryPowerUnitTypeChoices.NONE,
        help_text=_("Auxiliary power unit type of the tractor."),
    )
    fuel_draw_capacity = models.PositiveIntegerField(
        _("Fuel Draw Capacity"),
        default=0,
        help_text=_("Fuel draw capacity of the tractor."),
    )
    num_of_axles = models.PositiveIntegerField(
        _("Number of Axles"),
        default=0,
        help_text=_("Number of axles of the tractor."),
    )
    transmission_manufacturer = models.CharField(
        _("Transmission Manufacturer"),
        max_length=50,
        blank=True,
        help_text=_("Transmission manufacturer of the tractor."),
    )
    transmission_type = models.CharField(
        _("Transmission Type"),
        max_length=50,
        blank=True,
        help_text=_("Transmission type of the tractor."),
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
    fleet_code = models.ForeignKey(
        "dispatch.FleetCode",
        on_delete=models.CASCADE,
        related_name="tractor",
        related_query_name="tractor",
        verbose_name=_("Fleet"),
        blank=True,
        null=True,
        help_text=_("Fleet of the tractor."),
    )

    class Meta:
        """
        Tractor Model Metaclass
        """

        verbose_name = _("Tractor")
        verbose_name_plural = _("Tractor")
        ordering = ["code"]
        db_table = "tractor"
        constraints = [
            models.UniqueConstraint(
                Lower("code"),
                "organization",
                name="unique_tractor_code_organization",
            )
        ]

    def __str__(self) -> str:
        """Tractor string representation

        Returns:
            str: String representation of the Tractor Model
        """
        return textwrap.wrap(self.code, 50)[0]

    def get_absolute_url(self) -> str:
        """Tractor absolute URL

        Returns:
            str: Absolute URL of the Tractor Model
        """
        return reverse("tractor-detail", kwargs={"pk": self.pk})

    def clean(self) -> None:
        """Tractor Model clean method

        Raises:
            ValidationError: If the Tractor is leased and the leased date is not set
        """

        errors = {}
        if self.leased and not self.leased_date:
            errors["leased_date"] = _(
                "Leased date must be set if the tractor is leased. Please try again."
            )

        if (
            self.primary_worker
            and self.secondary_worker
            and self.primary_worker == self.secondary_worker
        ):
            errors["primary_worker"] = _(
                "Primary worker and secondary worker cannot be the same. Please try again."
            )

        if self.primary_worker and self.fleet_code != self.primary_worker.fleet_code:
            errors["primary_worker"] = _(
                "Primary worker must be in the same fleet as the tractor. Please try again."
            )

        if errors:
            raise ValidationError(errors)


class Trailer(GenericModel):
    """
    Stores information about a piece of Trailer for a :model:`organization.Organization`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    code = models.CharField(
        _("Code"),
        max_length=50,
        help_text=_("Code of the trailer."),
    )
    status = ChoiceField(
        _("Status"),
        choices=AvailabilityChoices.choices,
        help_text=_("Status of the trailer."),
        default=AvailabilityChoices.AVAILABLE,
    )
    equipment_type = models.ForeignKey(
        EquipmentType,
        on_delete=models.RESTRICT,
        related_name="trailer",
        verbose_name=_("Equipment Type"),
        help_text=_("Equipment type of the trailer."),
    )
    make = models.CharField(
        _("Make"),
        max_length=50,
        blank=True,
        help_text=_("Make of the trailer."),
    )
    model = models.CharField(
        _("Model"),
        max_length=50,
        blank=True,
        help_text=_("Model of the trailer."),
    )
    year = models.PositiveIntegerField(
        _("Year"),
        default=timezone.now().year,
        help_text=_("Year of the trailer."),
        blank=True,
        null=True,
    )
    vin_number = models.CharField(
        _("VIN Number"),
        max_length=17,
        blank=True,
        help_text=_("VIN number of the trailer."),
        validators=[us_vin_number_validator],
    )
    fleet_code = models.ForeignKey(
        "dispatch.FleetCode",
        on_delete=models.CASCADE,
        related_name="trailer",
        verbose_name=_("Fleet"),
        blank=True,
        null=True,
        help_text=_("Fleet of the trailer."),
    )
    state = models.CharField(
        _("State"),
        max_length=5,
        blank=True,
        help_text=_("State"),
    )
    owner = models.CharField(
        _("Owner"),
        max_length=50,
        blank=True,
        help_text=_("Owner of the trailer."),
    )
    manufacturer = models.ForeignKey(
        EquipmentManufacturer,
        on_delete=models.CASCADE,
        related_name="trailer",
        related_query_name="trailer",
        verbose_name=_("Manufacturer"),
        blank=True,
        null=True,
    )
    license_plate_number = models.CharField(
        _("License Plate Number"),
        max_length=50,
        blank=True,
        help_text=_("License plate number of the trailer."),
    )
    license_plate_state = models.CharField(
        _("State"),
        max_length=5,
        blank=True,
        help_text=_("State"),
    )
    # ------------
    # TODO(WOLFRED): Send a notification letting the equipment team know that the license plate is about to expire
    # ------------
    last_inspection = models.DateField(
        _("Last Inspection"),
        blank=True,
        null=True,
        help_text=_("Last inspection date of the trailer."),
    )
    is_leased = models.BooleanField(
        _("Is Leased"),
        default=False,
        help_text=_("Is the trailer leased."),
    )
    leased_date = models.DateField(
        _("Leased Date"),
        blank=True,
        null=True,
        help_text=_("Leased date of the trailer."),
    )
    registration_number = models.CharField(
        _("Registration Number"),
        max_length=50,
        blank=True,
        help_text=_("Registration number of the trailer."),
    )
    registration_state = models.CharField(
        _("State"),
        max_length=5,
        blank=True,
        help_text=_("State"),
    )
    registration_expiration = models.DateField(
        _("Registration Expiration"),
        blank=True,
        null=True,
        help_text=_("Registration expiration date of the trailer."),
    )

    class Meta:
        """
        Metaclass for the Trailer Model
        """

        verbose_name = _("Trailer")
        verbose_name_plural = _("Trailer")
        db_table = "trailer"
        constraints = [
            models.UniqueConstraint(
                Lower("code"),
                "organization",
                name="unique_trailer_code_organization",
            )
        ]

    def __str__(self) -> str:
        """String representation of the Trailer model

        Returns:
            str: String representation of the Trailer model
        """
        return textwrap.shorten(self.code, width=50, placeholder="...")

    def save(self, *args: typing.Any, **kwargs: typing.Any) -> None:
        """Save method for the Trailer model

        Returns:
            None: This method does not return anything
        """

        # If the trailer is leased and the leased date is not set, set it to today
        if self.is_leased and not self.leased_date:
            self.leased_date = timezone.now().date()

        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """Trailer absolute URL

        Returns:
            str: Absolute URL of the trailer Model
        """
        return reverse("trailer-detail", kwargs={"pk": self.pk})

    def clean(self) -> None:
        """Clean method for the Trailer model

        Returns:
            None: This method does not return anything

        Raises:
            ValidationError: Raised if the equipment type is not a trailer
        """

        super().clean()

        if (
            self.equipment_type
            and self.equipment_type.equipment_class
            != EquipmentType.EquipmentClassChoices.TRAILER
        ):
            raise ValidationError(
                {
                    "equipment_type": _(
                        "Cannot assign a non-trailer equipment type to a trailer. Check the equipment class"
                        " and try again."
                    )
                },
                code="invalid",
            )


class EquipmentMaintenancePlan(GenericModel):
    """
    Stores the maintenance plan information related to
    `equipment.EquipmentType` model.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    name = models.CharField(
        _("Name"),
        max_length=50,
        help_text=_("Name of the equipment maintenance plan."),
    )
    equipment_types = models.ManyToManyField(
        EquipmentType,
        related_name="maintenance_plan",
        related_query_name="maintenance_plans",
        verbose_name=_("Equipment Types"),
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
        ordering = ["name"]
        db_table = "equipment_maintenance_plan"
        constraints = [
            models.UniqueConstraint(
                fields=["name", "organization"],
                name="unique_equipment_maintenance_plan_name_organization",
            )
        ]
        permissions = [
            ("admin.equipment_maintenance.view", "Can view equipment maintenance")
        ]

    def __str__(self) -> str:
        """Equipment Maintenance Plan string representation

        Returns:
            str: String representation of the EquipmentMaintenancePlan Model
        """
        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Equipment Maintenance Plan absolute URL

        Returns:
            str: Absolute URL of the EquipmentMaintenancePlan Model
        """
        return reverse("equipment-maintenance-plans-detail", kwargs={"pk": self.pk})

    def clean(self) -> None:
        """Equipment Maintenance Plan clean method

        Raises:
            ValidationError: Validation Errors for the EquipmentMaintenancePlan Model
        """
        super().clean()

        errors = {}

        if not self.by_distance and not self.by_time and not self.by_engine_hours:
            errors["by_distance"] = _(
                "At least one of the fields must be checked: "
                "By Distance, By Time, By Engine Hours. Please try again."
            )

        if self.by_distance and not self.miles:
            errors["miles"] = _(
                "Miles must be set if the maintenance plan is by distance. Please try again."
            )

        if self.by_time and not self.months:
            errors["months"] = _(
                "Months must be set if the maintenance plan is by time. Please try again."
            )

        if self.by_engine_hours and not self.engine_hours:
            errors["engine_hours"] = _(
                "Engine hours must be set if the maintenance plan is by engine hours. Please try again."
            )

        if errors:
            raise ValidationError(errors)
