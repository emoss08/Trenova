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
from typing import Any, Optional, final

from django.conf import settings
from django.db import models
from django.db.models.aggregates import Sum
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from commodities.models import HazardousMaterial
from order.validation import OrderValidation
from stops.models import Stop
from utils.models import ChoiceField, GenericModel, RatingMethodChoices, StatusChoices

User = settings.AUTH_USER_MODEL


def order_documentation_upload_to(instance: OrderDocumentation, filename: str) -> str:
    """
    order_documentation_upload_to

    Args:
        instance (Order): The instance of the Order.
        filename (str): file name.

    Returns:
        str: upload path for the order documentation to be stored.
    """
    return f"order_documentation/{instance.order.pro_number}/{filename}"


class OrderControl(GenericModel):
    """
    Stores the order control information for a related :model:`organization.Organization`.
    """

    organization = models.OneToOneField(
        "organization.Organization",
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
    enforce_origin_destination = models.BooleanField(
        _("Compare Origin Destination"),
        default=False,
        help_text=_(
            "Compare and validate that origin and destination are not the same."
        ),
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


class Order(GenericModel):
    """
    Stores order information related to a :model:`organization.Organization`.
    """

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
        "accounting.RevenueCode",
        on_delete=models.PROTECT,
        related_name="orders",
        related_query_name="order",
        verbose_name=_("Revenue Code"),
        help_text=_("Revenue Code of the Order"),
        blank=True,
        null=True,
    )
    origin_location = models.ForeignKey(
        "location.Location",
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
        "location.Location",
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
        "customer.Customer",
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
        "equipment.EquipmentType",
        on_delete=models.PROTECT,
        related_name="orders",
        related_query_name="order",
        verbose_name=_("Equipment Type"),
        help_text=_("Equipment Type"),
    )
    commodity = models.ForeignKey(
        "commodities.Commodity",
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
    hazmat = models.ForeignKey(
        "commodities.HazardousMaterial",
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

    def clean(self) -> None:
        """Order save method

        Returns:
            None

        Raises:
            ValidationError: If the Order is not valid
        """

        # Call the OrderValidation class
        OrderValidation(  # type: ignore
            order=self, organization=self.organization, order_control=OrderControl
        ).validate()

    def save(self, *args: Any, **kwargs: Any) -> None:
        """Order save method

        Returns:
            None
        """
        self.full_clean()

        if self.ready_to_bill:
            self.sub_total = self.calculate_total()

        self.set_address()
        self.set_hazardous_class()

        if self.status == StatusChoices.COMPLETED:
            self.pieces = self.total_piece()
            self.weight = self.total_weight()

        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """Get the absolute url for the Order

        Returns:
            str: Absolute url for the Order
        """
        return reverse("order-detail", kwargs={"pk": self.pk})

    def calculate_total(self) -> decimal.Decimal:
        """Calculate the sub_total for an order

        Returns:
            decimal.Decimal: The total for the order
        """

        # Handle the flat fee rate calculation
        if self.rate_method == RatingMethodChoices.FLAT:
            return self.freight_charge_amount + self.other_charge_amount

        # Handle the mileage rate calculation
        if self.rate_method == RatingMethodChoices.PER_MILE:
            return self.freight_charge_amount * self.mileage + self.other_charge_amount

        return self.freight_charge_amount

    def set_hazardous_class(self) -> Optional[HazardousMaterial]:
        """Set the hazardous class from commodity

        if a commodity is selected automatically set the hazardous
        class from the relationship between commodity and
        HazardousMaterial.

        Returns:
            HazardousMaterial: Instance of the HazardousMaterial
        """
        if self.commodity.hazmat:
            self.hazmat = self.commodity.hazmat
        return self.hazmat

    def total_piece(self) -> int:
        """Get the total piece count for the order

        Returns:
            int: Total piece count for the order
        """
        return Stop.objects.filter(movement__order__exact=self).aggregate(
            Sum("pieces")
        )["pieces__sum"]

    def total_weight(self) -> int:
        """Get the total weight for the order.

        Returns:
            int: Total weight for the order
        """
        return Stop.objects.filter(movement__order__exact=self).aggregate(
            Sum("weight")
        )["weight__sum"]

    def set_address(self) -> None:
        """Set the address for the order

        Returns:
            None
        """
        o_control: OrderControl = OrderControl.objects.get(
            organization=self.entered_by.organization
        )

        if o_control.auto_pop_address:
            self.origin_address = self.origin_location.get_address_combination
            self.destination_address = self.destination_location.get_address_combination


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
        "billing.DocumentClassification",
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
        "dispatch.CommentType",
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


class AdditionalCharge(GenericModel):
    """
    Stores Additional Charge related to a :model:`order.Order`.
    """

    order = models.ForeignKey(
        Order,
        on_delete=models.CASCADE,
        related_name="additional_charges",
        related_query_name="additional_charge",
        verbose_name=_("Order"),
    )
    charge = models.ForeignKey(
        "billing.AccessorialCharge",
        on_delete=models.CASCADE,
        related_name="additional_charges",
        related_query_name="additional_charge",
        verbose_name=_("Charge"),
        help_text=_("Charge"),
    )
    charge_amount = models.DecimalField(
        _("Charge Amount"),
        max_digits=10,
        decimal_places=2,
        null=True,
        blank=True,
        help_text=_("Charge Amount"),
    )
    unit = models.PositiveIntegerField(
        _("Unit"),
        default=1,
        help_text=_("Number of units to be charged"),
    )
    sub_total = models.DecimalField(
        _("Sub Total Amount"),
        max_digits=10,
        decimal_places=2,
        default=0,
        help_text=_("Sub Total Amount"),
    )
    entered_by = models.ForeignKey(
        User,
        on_delete=models.CASCADE,
        related_name="additional_charges",
        related_query_name="additional_charge",
        verbose_name=_("Entered By"),
        help_text=_("Entered By"),
    )

    class Meta:
        """
        AdditionalCharges Metaclass
        """

        verbose_name = _("Additional Charge")
        verbose_name_plural = _("Additional Charges")

    def __str__(self) -> str:
        """String representation of the AdditionalCharges

        Returns:
            str: String representation of the AdditionalCharges
        """
        return f"{self.order} - {self.charge}"

    def save(self, *args: Any, **kwargs: Any):
        """
        Save the AdditionalCharge
        """
        self.charge_amount = self.charge.charge_amount
        self.sub_total = self.charge_amount * self.unit
        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """Get the absolute url for the AdditionalCharges

        Returns:
            str: Absolute url for the AdditionalCharges
        """
        return reverse("additional-charges-detail", kwargs={"pk": self.pk})


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
