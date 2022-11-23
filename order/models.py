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

from __future__ import annotations

import decimal
import textwrap
from typing import Any, final

from django.conf import settings
from django.core.exceptions import ValidationError
from django.db import models
from django.db.models import Sum
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from accounting.models import RevenueCode
from billing.models import DocumentClassification
from control_file.models import CommentType
from customer.models import Customer
from dispatch.models import DelayCode
from equipment.models import Equipment, EquipmentType
from location.models import Location
from organization.models import Organization
from utils.models import ChoiceField, GenericModel
from worker.models import Worker

User = settings.AUTH_USER_MODEL


def order_documentation_upload_to(instance: OrderDocumentation, filename: str) -> str:
    """
    order_documentation_upload_to _summary_

    Args:
        instance (Order): The instance of the Order.
        filename (str): file name.

    Returns:
        str: upload path for the order documentation to be stored.
    """
    return f"order_documentation/{instance.order.pro_number}/{filename}"


@final
class StatusChoices(models.TextChoices):
    """
    Status Choices for Order, Stop & Movement Statuses.
    """

    NEW = "N", _("New")
    IN_PROGRESS = "P", _("In Progress")
    COMPLETED = "C", _("Completed")
    BILLED = "B", _("Billed")
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
    enforce_customer = models.BooleanField(
        _("Enforce Customer"),
        default=False,
        help_text=_("Enforce Customer to being enter when entering an order."),
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
        help_text=_("Automatically generate routes for order entry."),
    )
    auto_pop_address = models.BooleanField(
        _("Auto Populate Address"),
        default=True,
        help_text=_(
            "Auto populate address from location ID " "when entering an order."
        ),
    )
    auto_sequence_stops = models.BooleanField(
        _("Auto Sequence Stops"),
        default=True,
        help_text=_("Auto Sequence stops for the order and movements."),
    )
    auto_order_total = models.BooleanField(
        _("Auto Order Total"),
        default=True,
        help_text=_("Automate the order total amount calculation."),
    )

    class Meta:
        """
        Metaclass for OrderControl
        """

        verbose_name = _("Order Control")
        verbose_name_plural = _("Order Controls")
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
    Hazardous Class Model Fields that can be used in the
    :model:`order.Order` & :model:`order.Commodity` model.
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
    hazard_class = ChoiceField(
        _("Hazard Class"),
        choices=HazardousClassChoices.choices,
        help_text=_("Hazard Class of the Hazardous Material"),
    )
    packing_group = ChoiceField(
        _("Packing Group"),
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
    unit_of_measure = ChoiceField(
        _("Unit of Measure"),
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


class QualifierCode(GenericModel):
    """
    Stores Qualifier Code information that can be used in stop notes.
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
        """
        Qualifier Code Metaclass
        """

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
    code_type = ChoiceField(
        _("Code Type"),
        choices=CodeTypeChoices.choices,
        help_text=_("Code Type of the Reason Code"),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("Description of the Reason Code"),
    )

    class Meta:
        """
        Reason Code Metaclass
        """

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


class Order(GenericModel):
    """
    Stores order information related to a :model:`organization.Organization`.
    """

    @final
    class RatingMethodChoices(models.TextChoices):
        """
        Rating Method choices for Order Model
        """

        FLAT = "F", _("Flat Fee")
        PER_MILE = "PM", _("Per Mile")
        PER_STOP = "PS", _("Per Stop")
        POUNDS = "PP", _("Per Pound")

    # General Information
    pro_number = models.CharField(
        _("Pro Number"),
        max_length=10,
        unique=True,
        editable=False,
        help_text=_("Pro Number of the Order"),
    )
    status = ChoiceField(
        _("Status"),
        choices=StatusChoices.choices,
        default=StatusChoices.NEW,
    )
    revenue_code = models.ForeignKey(
        RevenueCode,
        on_delete=models.PROTECT,
        related_name="orders",
        related_query_name="order",
        verbose_name=_("Revenue Code"),
        help_text=_("Revenue Code of the Order"),
    )
    origin_location = models.ForeignKey(
        Location,
        on_delete=models.PROTECT,
        related_name="origin_order",
        related_query_name="origin_orders",
        verbose_name=_("Origin Location"),
        help_text=_("Origin Location of the Order"),
    )
    origin_address = models.CharField(
        _("Origin Address"),
        max_length=255,
        blank=True,
        help_text=_("Origin Address of the Order"),
    )
    origin_appointment = models.DateTimeField(
        _("Origin Appointment"),
        help_text=_(
            "The time the equipment is expected to arrive at the origin/pickup."
        ),
    )
    destination_location = models.ForeignKey(
        Location,
        on_delete=models.PROTECT,
        related_name="destination_order",
        related_query_name="destination_orders",
        verbose_name=_("Destination Location"),
    )
    destination_address = models.CharField(
        _("Destination Address"),
        max_length=255,
        blank=True,
    )
    destination_appointment = models.DateTimeField(
        _("Destination Appointment Time"),
        help_text=_(
            "The time the equipment is expected to arrive at the destination/drop-off."
        ),
    )

    # Billing Information for the order
    mileage = models.DecimalField(
        _("Total Mileage"),
        max_digits=10,
        decimal_places=2,
        default=0,
        help_text=_("Total Mileage"),
    )
    other_charge_amount = models.DecimalField(
        _("Additional Charge Amount"),
        max_digits=10,
        decimal_places=2,
        default=0,
        help_text=_("Additional Charge Amount"),
    )
    freight_charge_amount = models.DecimalField(
        _("Freight Charge Amount"),
        max_digits=10,
        decimal_places=2,
        default=0,
        help_text=_("Freight Charge Amount"),
    )
    rate_method = ChoiceField(
        _("Rating Method"),
        choices=RatingMethodChoices.choices,
        default=RatingMethodChoices.FLAT,
        help_text=_("Rating Method"),
    )
    customer = models.ForeignKey(
        Customer,
        on_delete=models.PROTECT,
        related_name="orders",
        related_query_name="order",
        verbose_name=_("Customer"),
        help_text=_("Customer of the Order"),
    )
    pieces = models.PositiveIntegerField(
        _("Pieces"),
        help_text=_("Total Piece Count of the Order"),
        default=0,
    )
    weight = models.DecimalField(
        _("Weight"),
        max_digits=10,
        decimal_places=2,
        help_text=_("Total Weight of the Order"),
        default=0,
    )
    ready_to_bill = models.BooleanField(
        _("Ready to Bill"),
        default=False,
        help_text=_("Ready to Bill"),
    )
    bill_date = models.DateField(
        _("Billed Date"),
        null=True,
        blank=True,
        help_text=_("Billed Date"),
    )
    billed = models.BooleanField(
        _("Billed"),
        default=False,
        help_text=_("Billed"),
    )
    transferred_to_billing = models.BooleanField(
        _("Transferred to Billing"),
        default=False,
        help_text=_("Transferred to Billing"),
    )
    billing_transfer_date = models.DateTimeField(
        _("Billing Transfer Date"),
        null=True,
        blank=True,
        help_text=_("Billing Transfer Date"),
    )
    sub_total = models.DecimalField(
        _("Sub Total Amount"),
        max_digits=10,
        decimal_places=2,
        default=0,
        help_text=_("Sub Total Amount"),
    )

    # Dispatch Information
    equipment_type = models.ForeignKey(
        EquipmentType,
        on_delete=models.PROTECT,
        related_name="orders",
        related_query_name="order",
        verbose_name=_("Equipment Type"),
        help_text=_("Equipment Type"),
    )
    commodity = models.ForeignKey(
        Commodity,
        on_delete=models.PROTECT,
        related_name="orders",
        related_query_name="order",
        verbose_name=_("Commodity"),
        help_text=_("Commodity"),
    )
    entered_by = models.ForeignKey(
        User,
        on_delete=models.PROTECT,
        related_name="orders",
        related_query_name="order",
        verbose_name=_("User"),
        help_text=_("Order entered by User"),
    )
    hazmat_id = models.ForeignKey(
        HazardousMaterial,
        on_delete=models.PROTECT,
        related_name="orders",
        related_query_name="order",
        verbose_name=_("Hazardous Class"),
        null=True,
        blank=True,
        help_text=_("Hazardous Class"),
    )
    temperature_min = models.DecimalField(
        _("Minimum Temperature"),
        max_digits=10,
        decimal_places=1,
        null=True,
        blank=True,
        help_text=_("Minimum Temperature"),
    )
    temperature_max = models.DecimalField(
        _("Maximum Temperature"),
        max_digits=10,
        decimal_places=1,
        null=True,
        blank=True,
        help_text=_("Maximum Temperature"),
    )
    bol_number = models.CharField(
        _("BOL Number"),
        max_length=255,
        help_text=_("BOL Number"),
    )
    consignee_ref_number = models.CharField(
        _("Consignee Reference Number"),
        max_length=255,
        blank=True,
        help_text=_("Consignee Reference Number"),
    )
    comment = models.CharField(
        _("Comment"),
        max_length=100,
        blank=True,
        help_text=_("Planning Comment"),
    )

    class Meta:
        """
        Order Metaclass
        """

        verbose_name = _("Order")
        verbose_name_plural = _("Orders")
        ordering: list[str] = ["-pro_number"]

    def __str__(self) -> str:
        """String representation of the Order

        Returns:
            str: String representation of the Order
        """
        return textwrap.wrap(self.pro_number, 10)[0]

    @property
    def total_pieces(self) -> int:
        """Get the total piece count for the order

        Returns:
            int: Total piece count for the order
        """
        return Stop.objects.filter(movement__order__exact=self).aggregate(
            Sum("pieces")
        )["pieces__sum"]

    @property
    def total_weight(self) -> int:
        """Get the total weight for the order.

        Returns:
            int: Total weight for the order
        """
        return Stop.objects.filter(movement__order__exact=self).aggregate(
            Sum("weight")
        )["weight__sum"]

    def calculate_total(self) -> decimal.Decimal:
        """Calculate the sub_total for an order

        # TODO(emoss): Move this into a service class

        Returns:
            decimal.Decimal: The total for the order
        """

        # Handle the flat fee rate calculation
        if self.rate_method == Order.RatingMethodChoices.FLAT:
            return self.freight_charge_amount + self.other_charge_amount

        # Handle the mileage rate calculation
        if self.rate_method == Order.RatingMethodChoices.PER_MILE:
            return self.freight_charge_amount * self.mileage + self.other_charge_amount

        return self.freight_charge_amount

    def set_address(self) -> None:
        """Set the address for the order

        Returns:
            None
        """
        order_control = OrderControl.objects.get(organization=self.organization)

        if order_control.auto_pop_address:
            self.origin_address = self.origin_location.get_address_combination
            self.destination_address = self.destination_location.get_address_combination

    def clean(self) -> None:
        """Order save method

        Returns:
            None

        Raises:
            ValidationError: If the Order is not valid
        """
        if (
            self.rate_method == Order.RatingMethodChoices.FLAT
            and self.freight_charge_amount is None
        ):
            raise ValidationError(
                {
                    "rate_method": ValidationError(
                        _("Freight Charge Amount is required for flat rating method."),
                        code="invalid",
                    )
                }
            )

        if (
            self.rate_method == Order.RatingMethodChoices.PER_MILE
            and self.mileage is None
        ):
            raise ValidationError(
                {
                    "rate_method": ValidationError(
                        _("Mileage is required for per mile rating method."),
                        code="invalid",
                    )
                }
            )
        if self.ready_to_bill and self.status != StatusChoices.COMPLETED:
            raise ValidationError(
                {
                    "ready_to_bill": ValidationError(
                        _(
                            "Cannot mark an order ready to bill if the order status"
                            " is not complete."
                        ),
                        code="invalid",
                    )
                }
            )

    def save(self, *args: Any, **kwargs: Any) -> None:
        """Order save method

        Returns:
            None
        """
        self.full_clean()

        if self.status == StatusChoices.COMPLETED:
            self.pieces = self.total_pieces
            self.weight = self.total_weight

        if self.ready_to_bill:
            self.sub_total = self.calculate_total()

        self.set_address()

        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """Get the absolute url for the Order

        Returns:
            str: Absolute url for the Order
        """
        return reverse("order-detail", kwargs={"pk": self.pk})


class Movement(GenericModel):
    """
    Stores movement information related to a :model:`order.Order`.
    """

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
        Order,
        on_delete=models.PROTECT,
        related_name="movements",
        related_query_name="movement",
        verbose_name=_("Order"),
        help_text=_("Order of the Movement"),
    )
    equipment = models.ForeignKey(
        Equipment,
        on_delete=models.PROTECT,
        related_name="movements",
        related_query_name="movement",
        verbose_name=_("Equipment"),
        null=True,
        blank=True,
        help_text=_("Equipment of the Movement"),
    )
    primary_worker = models.ForeignKey(
        Worker,
        on_delete=models.PROTECT,
        related_name="movements",
        related_query_name="movement",
        verbose_name=_("Primary Worker"),
        null=True,
        blank=True,
        help_text=_("Primary Worker of the Movement"),
    )
    secondary_worker = models.ForeignKey(
        Worker,
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

    def _validate_movement_statuses(self) -> None:
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

    def _validate_movement_worker(self) -> None:
        """Validate Movement worker

        Require a primary worker and equipment to set the
        movement status to in progress.

        Returns:
            None

        Raises:
            ValidationError: If the old movement worker is not None and the user tries to change the worker.
        """
        if self.status == (
            StatusChoices.IN_PROGRESS and not self.primary_worker and not self.equipment
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

    def _validate_worker_compare(self) -> None:
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

    def _validate_movement_stop_status(self) -> None:
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

    def validate(self) -> None:
        """Validate the Movement

        Returns:
            None

        Raises:
            ValidationError: If the Movement is not valid
        """
        self._validate_movement_statuses()
        self._validate_movement_worker()
        self._validate_worker_compare()
        self._validate_movement_stop_status()

    def clean(self) -> None:
        """Stop clean method

        Returns:
            None
        """
        self.validate()

    def get_absolute_url(self) -> str:
        """Get the absolute url for the Movement

        Returns:
            str: Absolute url for the Movement
        """
        return reverse("movement-detail", kwargs={"pk": self.pk})


class Stop(GenericModel):
    """
    Stores movement information related to a :model:`order.Movement`.
    """

    status = ChoiceField(
        choices=StatusChoices.choices,
        default=StatusChoices.NEW,
        help_text=_("The status of the stop."),
    )
    sequence = models.PositiveIntegerField(
        _("Sequence"),
        default=1,
        help_text=_("The sequence of the stop in the movement."),
    )
    movement = models.ForeignKey(
        Movement,
        on_delete=models.CASCADE,
        related_name="stops",
        related_query_name="stop",
        verbose_name=_("Movement"),
        help_text=_("The movement that the stop belongs to."),
    )
    location = models.ForeignKey(
        Location,
        on_delete=models.PROTECT,
        related_name="stops",
        related_query_name="stop",
        verbose_name=_("Location"),
        help_text=_("The location of the stop."),
    )
    pieces = models.PositiveIntegerField(
        _("Pieces"),
        default=0,
        null=True,
        blank=True,
        help_text=_("Pieces"),
    )
    weight = models.PositiveIntegerField(
        _("Weight"),
        default=0,
        null=True,
        blank=True,
        help_text=_("Weight"),
    )
    address_line = models.CharField(
        _("Stop Address"),
        max_length=255,
        help_text=_("Stop Address"),
    )
    appointment_time = models.DateTimeField(
        _("Stop Appointment Time"),
        help_text=_("The time the equipment is expected to arrive at the stop."),
    )
    arrival_time = models.DateTimeField(
        _("Stop Arrival Time"),
        null=True,
        blank=True,
        help_text=_("The time the equipment actually arrived at the stop."),
    )
    departure_time = models.DateTimeField(
        _("Stop Departure Time"),
        null=True,
        blank=True,
        help_text=_("The time the equipment actually departed from the stop."),
    )
    stop_type = ChoiceField(
        choices=StopChoices.choices,
        help_text=_("The type of stop."),
    )

    class Meta:
        """
        Metaclass for the Stop model
        """

        verbose_name = _("Stop")
        verbose_name_plural = _("Stops")
        ordering: list[str] = ["movement", "sequence"]

    def __str__(self) -> str:
        """String representation of the Stop

        Returns:
            str: String representation of the Stop
        """
        return f"{self.movement} - {self.sequence} - {self.location}"

    def clean(self) -> None:
        """Stop clean Method

        Returns:
            None

        Raises:
            ValidationError: If the stop is not valid.

        """
        if self.pk:
            if self.status == StatusChoices.NEW:
                old_status = Stop.objects.get(pk=self.pk).status

                if old_status in [StatusChoices.IN_PROGRESS, StatusChoices.COMPLETED]:
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

            if self.sequence > 1:
                previous_stop = self.movement.stops.filter(
                    sequence=self.sequence - 1
                ).first()

                if (
                    previous_stop
                    and self.appointment_time < previous_stop.appointment_time
                ):
                    raise ValidationError(
                        {
                            "appointment_time": ValidationError(
                                _("Appointment time must be after previous stop."),
                                code="invalid",
                            )
                        }
                    )

                if previous_stop and previous_stop.status != StatusChoices.COMPLETED:
                    if self.status in [
                        StatusChoices.IN_PROGRESS,
                        StatusChoices.COMPLETED,
                    ]:
                        raise ValidationError(
                            {
                                "status": ValidationError(
                                    _(
                                        "Cannot change status to in progress or completed if"
                                        " previous stop is not completed."
                                    ),
                                    code="invalid",
                                )
                            }
                        )

                if self.sequence < self.movement.stops.count():
                    next_stop = self.movement.stops.filter(
                        sequence__exact=self.sequence + 1
                    ).first()

                    if next_stop and self.appointment_time > next_stop.appointment_time:
                        raise ValidationError(
                            {
                                "appointment_time": ValidationError(
                                    _("Appointment time must be before next stop."),
                                    code="invalid",
                                )
                            }
                        )

                    # If the next stop is in progress or completed, the current stop cannot be available
                    if (
                        next_stop
                        and self.status != StatusChoices.COMPLETED
                        and next_stop.status
                        in [
                            StatusChoices.COMPLETED,
                            StatusChoices.IN_PROGRESS,
                        ]
                    ):
                        raise ValidationError(
                            {
                                "status": ValidationError(
                                    _(
                                        "Previous stop must be completed before this stop can"
                                        " be in progress or completed."
                                    ),
                                    code="invalid",
                                )
                            }
                        )

                    if not self.movement.primary_worker and not self.movement.equipment:
                        if self.status in [
                            StatusChoices.IN_PROGRESS,
                            StatusChoices.COMPLETED,
                        ]:
                            raise ValidationError(
                                {
                                    "status": ValidationError(
                                        _(
                                            "Cannot change status to in progress or completed if"
                                            " there is no equipment or primary worker."
                                        ),
                                        code="invalid",
                                    )
                                }
                            )

                        if self.arrival_time or self.departure_time:
                            raise ValidationError(
                                {
                                    "arrival_time": ValidationError(
                                        _(
                                            "Must assign worker or equipment to movement before"
                                            " setting arrival or departure time."
                                        ),
                                        code="invalid",
                                    )
                                }
                            )

                        if self.departure_time and not self.arrival_time:
                            raise ValidationError(
                                {
                                    "departure_time": ValidationError(
                                        _(
                                            "Must set arrival time before setting departure time."
                                        ),
                                        code="invalid",
                                    )
                                }
                            )

                        if self.departure_time < self.arrival_time:
                            raise ValidationError(
                                {
                                    "departure_time": ValidationError(
                                        _("Departure time must be after arrival time."),
                                        code="invalid",
                                    )
                                }
                            )

    def save(self, *args: Any, **kwargs: Any) -> None:
        """Stop save method

        Args:
            *args (Any): Arguments
            **kwargs (Any): Keyword Arguments

        Returns:
            None
        """

        self.full_clean()

        # If the status changes to in progress, change the movement status associated to this stop to in progress.
        if self.status == StatusChoices.IN_PROGRESS:
            self.movement.status = StatusChoices.IN_PROGRESS
            self.movement.save()

        # if the last stop is completed, change the movement status to complete.
        if self.status == StatusChoices.COMPLETED:
            if (
                self.movement.stops.filter(status=StatusChoices.COMPLETED).count()
                == self.movement.stops.count()
            ):
                self.movement.status = StatusChoices.COMPLETED
                self.movement.save()

        # If the arrival time is set, change the status to in progress.
        if self.arrival_time:
            self.status = StatusChoices.IN_PROGRESS

            # If the arrival time of the stop is after the appointment time, create a service incident.
            if self.arrival_time > self.appointment_time:
                ServiceIncident.objects.create(
                    organization=self.movement.order.organization,
                    movement=self.movement,
                    stop=self,
                    delay_time=self.arrival_time - self.appointment_time,
                )

        # If the stop arrival and departure time are set, change the status to complete.
        if self.arrival_time and self.departure_time:
            self.status = StatusChoices.COMPLETED

        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """Get the absolute url for the Stop

        Returns:
            str: Absolute url for the Stop
        """
        return reverse("stop-detail", kwargs={"pk": self.pk})


class ServiceIncident(GenericModel):
    """
    Stores Service Incident information related to a
    :model:`order.Order` and :model:`order.Stop`.
    """

    movement = models.ForeignKey(
        Movement,
        on_delete=models.CASCADE,
        related_name="service_incidents",
        related_query_name="service_incident",
        verbose_name=_("Movement"),
    )
    stop = models.ForeignKey(
        "Stop",
        on_delete=models.CASCADE,
        related_name="service_incidents",
        related_query_name="service_incident",
        verbose_name=_("Stop"),
    )
    delay_code = models.ForeignKey(
        DelayCode,
        on_delete=models.PROTECT,
        related_name="service_incidents",
        related_query_name="service_incident",
        verbose_name=_("Delay Code"),
        blank=True,
        null=True,
    )
    delay_reason = models.CharField(
        _("Delay Reason"),
        max_length=100,
        blank=True,
    )
    delay_time = models.DurationField(
        _("Delay Time"),
        null=True,
        blank=True,
    )

    class Meta:
        """
        ServiceIncident Metaclass
        """

        verbose_name = _("Service Incident")
        verbose_name_plural = _("Service Incidents")

    def __str__(self) -> str:
        """String representation of the ServiceIncident

        Returns:
            str: String representation of the ServiceIncident
        """
        return f"{self.movement} - {self.stop} - {self.delay_code}"

    def get_absolute_url(self) -> str:
        """Get the absolute url for the ServiceIncident

        Returns:
            str: Absolute url for the ServiceIncident
        """
        return reverse("service-incident-detail", kwargs={"pk": self.pk})


class OrderDocumentation(GenericModel):
    """
    Stores documentation related to a :model:`order.Order`.
    """

    order = models.ForeignKey(
        Order,
        on_delete=models.CASCADE,
        related_name="order_documentation",
        verbose_name=_("Order"),
    )
    document = models.FileField(
        _("Document"),
        upload_to=order_documentation_upload_to,
        null=True,
        blank=True,
    )
    document_class = models.ForeignKey(
        DocumentClassification,
        on_delete=models.CASCADE,
        related_name="order_documentation",
        verbose_name=_("Document Class"),
        help_text=_("Document Class"),
    )

    class Meta:
        """
        OrderDocumentation Metaclass
        """

        verbose_name = _("Order Documentation")
        verbose_name_plural = _("Order Documentation")

    def __str__(self) -> str:
        """String representation of the OrderDocumentation

        Returns:
            str: String representation of the OrderDocumentation
        """
        return f"{self.order} - {self.document_class}"

    def get_absolute_url(self) -> str:
        """Get the absolute url for the OrderDocumentation

        Returns:
            str: Absolute url for the OrderDocumentation
        """
        return reverse("order-documentation-detail", kwargs={"pk": self.pk})


class OrderComment(GenericModel):
    """
    Stores comments related to a :model:`order.Order`.
    """

    order = models.ForeignKey(
        Order,
        on_delete=models.CASCADE,
        related_name="order_comments",
        related_query_name="order_comment",
        verbose_name=_("Order"),
    )
    comment_type = models.ForeignKey(
        CommentType,
        on_delete=models.CASCADE,
        related_name="order_comments",
        related_query_name="order_comment",
        verbose_name=_("Comment Type"),
        help_text=_("Comment Type"),
    )
    comment = models.TextField(
        _("Comment"),
        help_text=_("Comment"),
    )
    entered_by = models.ForeignKey(
        User,
        on_delete=models.CASCADE,
        related_name="order_comments",
        related_query_name="order_comment",
        verbose_name=_("Entered By"),
        help_text=_("Entered By"),
    )

    class Meta:
        """
        OrderComment Metaclass
        """

        verbose_name = _("Order Comment")
        verbose_name_plural = _("Order Comments")

    def __str__(self) -> str:
        """String representation of the OrderComment

        Returns:
            str: String representation of the OrderComment
        """
        return f"{self.order} - {self.comment}"

    def get_absolute_url(self) -> str:
        """Get the absolute url for the OrderComment

        Returns:
            str: Absolute url for the OrderComment
        """
        return reverse("order-comment-detail", kwargs={"pk": self.pk})
