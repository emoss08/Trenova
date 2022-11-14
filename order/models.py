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
from typing import Any, final

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from core.models import GenericModel
from organization.models import Organization


def order_documentation_upload_to(instance, filename: str) -> str:
    """
    order_documentation_upload_to _summary_

    Args:
        instance (Order): The instance of the Order.
        filename (str): file name.

    Returns:
        str: upload path for the order documentation to be stored.
    """
    ...


@final
class StatusChoices(models.TextChoices):
    """
    Status Choices for Order, Stop & Movement Statuses.
    """

    AVAILABLE = "A", _("Available")
    IN_PROGRESS = "P", _("In Progress")
    COMPLETED = "C", _("Completed")
    VOIDED = "V", _("Voided")


@final
class StopChoices(models.TextChoices):
    """
    Status Choices for the Stop Model
    """

    PICKUP = "P", _("Pickup")
    SPLIT_PICKUP = "SP", _("Split Pickup")
    SPLIT_DROP = "SD", _("Split Drop Off")
    DELIVERY = "D", _("Delivery")
    DROP_OFF = "DO", _("Drop Off")


class RatingMethodChoices(models.TextChoices):
    """
    Rating Method choices for Order Model
    """

    FLAT = "F", _("Flat Fee")
    PER_MILE = "PM", _("Per Mile")
    PER_STOP = "PS", _("Per Stop")
    POUNDS = "PP", _("Per Pound")


# Configuration Files
class OrderControl(GenericModel):
    """
    Stores the order control information for a related :model:`organization.Organization`.
    """

    organization = models.OneToOneField(
        Organization,
        on_delete=models.CASCADE,
        verbose_name=_("Organization"),
        related_name="order_control",
        related_query_name="order_controls",
    )
    auto_rate_orders = models.BooleanField(
        _("Auto Rate"),
        default=True,
        help_text=_("Auto rate orders"),
    )
    calculate_distance = models.BooleanField(
        _("Calculate Distance"),
        default=True,
        help_text=_("Calculate distance for the order"),
    )
    enforce_bill_to = models.BooleanField(
        _("Enforce Bill To"),
        default=False,
        help_text=_("Enforce bill to to being enter when entering an order."),
    )
    enforce_rev_code = models.BooleanField(
        _("Enforce Rev Code"),
        default=False,
        help_text=_("Enforce rev code code being entered when entering an order."),
    )
    enforce_shipper = models.BooleanField(
        _("Enforce Shipper"),
        default=False,
        help_text=_("Enforce shipper when putting in an order."),
    )
    enforce_cancel_comm = models.BooleanField(
        _("Enforce Voided Comm"),
        default=False,
        help_text=_("Enforce comment when cancelling an order."),
    )

    generate_routes = models.BooleanField(
        _("Generate Routes"),
        default=False,
        help_text=_("Generate routes for the organization"),
    )

    class Meta:
        """
        Metaclass for OrderControl
        """

        verbose_name: str = _("Order Control")
        verbose_name_plural: str = _("Order Controls")
        ordering: list[str] = ["organization"]

    def __str__(self) -> str:
        """Order control string representation

        Returns:
            str: Order control string representation
        """
        return textwrap.wrap(self.organization.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Order control absolute url

        Returns:
            str: Order control absolute url
        """
        return reverse("order_control:detail", kwargs={"pk": self.pk})


class HazardousMaterial(GenericModel):
    """
    Hazardous Class Model Fields
    """

    @final
    class HazardousClassChoices(models.TextChoices):
        """
        Status choices for Order model
        """

        CLASS_1_1 = "1.1", _("Division 1.1: Mass Explosive Hazard")
        CLASS_1_2 = "1.2", _("Division 1.2: Projection Hazard")
        CLASS_1_3 = "1.3", _(
            "Division 1.3: Fire and/or Minor Blast/Minor Projection Hazard"
        )
        CLASS_1_4 = "1.4", _("Division 1.4: Minor Explosion Hazard")
        CLASS_1_5 = "1.5", _(
            "Division 1.5: Very Insensitive With Mass Explosion Hazard"
        )
        CLASS_1_6 = "1.6", _(
            "Division 1.6: Extremely Insensitive; No Mass Explosion Hazard"
        )
        CLASS_2_1 = "2.1", _("Division 2.1: Flammable Gases")
        CLASS_2_2 = "2.2", _("Division 2.2: Non-Flammable Gases")
        CLASS_2_3 = "2.3", _("Division 2.3: Poisonous Gases")
        CLASS_3 = "3", _("Division 3: Flammable Liquids")
        CLASS_4_1 = "4.1", _("Division 4.1: Flammable Solids")
        CLASS_4_2 = "4.2", _("Division 4.2: Spontaneously Combustible Solids")
        CLASS_4_3 = "4.3", _("Division 4.3: Dangerous When Wet")
        CLASS_5_1 = "5.1", _("Division 5.1: Oxidizing Substances")
        CLASS_5_2 = "5.2", _("Division 5.2: Organic Peroxides")
        CLASS_6_1 = "6.1", _("Division 6.1: Toxic Substances")
        CLASS_6_2 = "6.2", _("Division 6.2: Infectious Substances")
        CLASS_7 = "7", _("Division 7: Radioactive Material")
        CLASS_8 = "8", _("Division 8: Corrosive Substances")
        CLASS_9 = "9", _("Division 9: Miscellaneous Hazardous Substances and Articles")

    @final
    class PackingGroupChoices(models.TextChoices):
        """
        Status choices for Order model
        """

        ONE = "I", _("I")
        TWO = "II", _("II")
        THREE = "III", _("III")

    is_active = models.BooleanField(
        default=True,
        verbose_name=_("Is Active"),
    )
    name = models.CharField(
        _("Name"),
        max_length=255,
        unique=True,
        help_text=_("Name of the Hazardous Class"),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Description of the Hazardous Class"),
    )
    hazard_class = models.CharField(
        _("Hazard Class"),
        max_length=255,
        choices=HazardousClassChoices.choices,
        help_text=_("Hazard Class of the Hazardous Material"),
    )
    packing_group = models.CharField(
        _("Packing Group"),
        max_length=255,
        choices=PackingGroupChoices.choices,
        help_text=_("Packing Group of the Hazardous Material"),
        blank=True,
    )
    erg_number = models.CharField(
        _("ERG Number"),
        max_length=255,
        blank=True,
    )
    proper_shipping_name = models.TextField(
        _("Proper Shipping Name"),
        help_text=_("Proper Shipping Name of the Hazardous Material"),
        blank=True,
    )

    class Meta:
        verbose_name = _("Hazardous Material")
        verbose_name_plural = _("Hazardous Materials")
        ordering: list[str] = ["name"]

    def __str__(self) -> str:
        """Hazardous Material String Representation

        Returns:
            str: Hazardous Material Name
        """
        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Hazardous Material Absolute URL

        Returns:
            str: Hazardous Material Absolute URL
        """
        return reverse("order:hazardousmaterial_detail", kwargs={"pk": self.pk})


class OrderType(GenericModel):
    """
    Order Type Model Fields
    """

    is_active = models.BooleanField(
        default=True,
        verbose_name=_("Is Active"),
    )
    name = models.CharField(
        _("Name"),
        max_length=255,
        unique=True,
        help_text=_("Name of the Order Type"),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Description of the Order Type"),
    )

    class Meta:
        verbose_name = _("Order Type")
        verbose_name_plural = _("Order Types")
        ordering: list[str] = ["name"]

    def __str__(self) -> str:
        """Order Type String Representation

        Returns:
            str: Order Type Name
        """
        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Order Type Absolute URL

        Returns:
            str: Order Type Absolute URL
        """
        return reverse("order:ordertype_detail", kwargs={"pk": self.pk})


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
    unit_of_measure = models.CharField(
        _("Unit of Measure"),
        max_length=255,
        choices=UnitOfMeasureChoices.choices,
        help_text=_("Unit of Measure of the Commodity"),
        blank=True,
    )
    hazmat = models.ForeignKey(
        HazardousMaterial,
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
        verbose_name = _("Commodity")
        verbose_name_plural = _("Commodities")
        ordering: list[str] = ["name"]

    def __str__(self) -> str:
        """Commodity String Representation

        Returns:
            str: Commodity Name
        """
        return textwrap.wrap(self.name, 50)[0]

    def save(self, **kwargs: Any) -> None:
        """Save Commodity

        Args:
            **kwargs (Any): Keyword Arguments
        """
        if self.hazmat:
            self.is_hazmat = True
        super().save(**kwargs)

    def get_absolute_url(self) -> str:
        """Commodity Absolute URL

        Returns:
            str: Commodity Absolute URL
        """
        return reverse("order:commodity_detail", kwargs={"pk": self.pk})


class QualifierCode(GenericModel):
    """
    Stores Qualifier Code information that can be used in stop notes
    """

    code = models.CharField(
        _("Code"),
        max_length=255,
        unique=True,
        help_text=_("Code of the Qualifier Code"),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("Description of the Qualifier Code"),
    )

    class Meta:
        verbose_name = _("Qualifier Code")
        verbose_name_plural = _("Qualifier Codes")
        ordering: list[str] = ["code"]

    def __str__(self) -> str:
        """Qualifier Code String Representation

        Returns:
            str: Code of the Qualifier
        """
        return textwrap.wrap(self.code, 50)[0]

    def get_absolute_url(self) -> str:
        """Qualifier Code Absolute URL

        Returns:
            str: Qualifier Code Absolute URL
        """
        return reverse("order:qualifiercode-detail", kwargs={"pk": self.pk})


class ReasonCode(GenericModel):
    """
    Stores Reason code information for when a load is voided or cancelled.
    """

    @final
    class CodeTypeChoices(models.TextChoices):
        """
        Code Type choices for Reason Code model
        """

        VOIDED = "VOIDED", _("Voided")
        CANCELLED = "CANCELLED", _("Cancelled")

    code = models.CharField(
        _("Code"),
        max_length=255,
        unique=True,
        help_text=_("Code of the Reason Code"),
    )
    code_type = models.CharField(
        _("Code Type"),
        max_length=9,
        choices=CodeTypeChoices.choices,
        help_text=_("Code Type of the Reason Code"),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("Description of the Reason Code"),
    )

    class Meta:
        verbose_name = _("Reason Code")
        verbose_name_plural = _("Reason Codes")
        ordering: list[str] = ["code"]

    def __str__(self) -> str:
        """Reason Code String Representation

        Returns:
            str: Code of the Reason
        """
        return textwrap.wrap(self.code, 50)[0]

    def get_absolute_url(self) -> str:
        """Reason Code Absolute URL

        Returns:
            str: Reason Code Absolute URL
        """
        return reverse("order:reasoncode-detail", kwargs={"pk": self.pk})


# class Order(GenericModel):
#     """
#     Stores order information.
#     """
#
#     pro_number = models.CharField(
#         _("Pro Number"),
#         max_length=10,
#         unique=True,
#         editable=False,
#         help_text=_("Pro Number of the Order"),
#     )
