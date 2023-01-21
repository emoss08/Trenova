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
import uuid
from typing import Any, final

from django.conf import settings
from django.core.exceptions import ValidationError
from django.core.validators import FileExtensionValidator
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from commodities.models import HazardousMaterial
from stops.models import Stop
from utils.models import ChoiceField, GenericModel, RatingMethodChoices, StatusChoices

User = settings.AUTH_USER_MODEL


def order_documentation_upload_to(instance: OrderDocumentation, filename: str) -> str:
    """Returns the path to upload the order documentation to.

    Upload the order documentation to the order documentation directory
    and name the file with the order id and the filename.

    Args:
        instance (Order): The instance of the Order.
        filename (str): file name.

    Returns:
        Upload path for the order documentation to be stored. For example.

        `order_documentation/M000123/invoice-12341.pdf`

        Upload path is always a string. If the file is not uploaded, the
        upload path will be an empty string.

    See Also:
        `OrderDocumentation`: The model that this function is used for.
    """
    return f"order_documentation/{instance.order.pro_number}/{filename}"


class OrderControl(GenericModel):
    """Stores the order control information for a related :model:`organization.Organization`.

    The OrderControl model stores the order control information for a related
    organization. It is used to store information such as whether to automatically
    rate orders, calculate distance, enforce customer information, generate routes,
    and more.

    Attributes:
        id (UUIDField): Primary key and default value is a randomly generated UUID.
            Editable and unique.
        organization (OneToOneField): ForeignKey to the related organization model
            with a CASCADE on delete. Has a verbose name of "Organization" and
            related names of "order_control" and "order_controls".
        auto_rate_orders (BooleanField): Default value is True.
            Help text is "Auto rate orders".
        calculate_distance (BooleanField): Default value is True.
            Help text is "Calculate distance for the order".
        enforce_rev_code (BooleanField): Default value is False.
            Help text is "Enforce rev code being entered when entering an order.".
        generate_routes (BooleanField): Default value is False.
            Help text is "Automatically generate routes for order entry.".
        auto_sequence_stops (BooleanField): Default value is True.
            Help text is "Auto Sequence stops for the order and movements.".
        auto_order_total (BooleanField): Default value is True.
            Help text is "Automate the order total amount calculation.".
        enforce_origin_destination (BooleanField): Default value is False.
            Help text is "Compare and validate that origin and destination are not the same.".

    Methods:
        get_absolute_url(self) -> str:
            Returns the URL for this object's detail view.

        save(self, *args, **kwargs) -> None:
            Saves the current object to the database.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
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
        help_text=_("Automatically Calculate distance for the order"),
    )
    enforce_rev_code = models.BooleanField(
        _("Enforce Rev Code"),
        default=False,
        help_text=_("Enforce rev code code being entered when entering an order."),
    )
    enforce_voided_comm = models.BooleanField(
        _("Enforce Voided Comm"),
        default=False,
        help_text=_("Enforce comment when voiding an order."),
    )
    generate_routes = models.BooleanField(
        _("Generate Routes"),
        default=False,
        help_text=_("Automatically generate routing information for the order."),
    )
    enforce_commodity = models.BooleanField(
        _("Enforce Commodity Code"),
        default=False,
        help_text=_("Enforce the commodity input on the entry of an order."),
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
        ordering = ["organization"]

    def __str__(self) -> str:
        """Order control string representation

        Returns:
            str: Order control string representation
        """
        return textwrap.wrap(self.organization.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Order control absolute url

        Returns:
            Absolute url for the order control object. For example,
            `/order_control/1/`
        """
        return reverse("order_control:detail", kwargs={"pk": self.pk})


class OrderType(GenericModel):
    """Stores the order type information for a related :model:`organization.Organization`.

    The OrderType model stores information about an order type, such as its name,
    description, and whether it is active. It also has metadata for ordering and
    verbose names.

    Attributes:
        id (UUIDField): Primary key and default value is a randomly generated UUID.
            Editable and unique.
        is_active (BooleanField): Default value is True. Verbose name is "Is Active".
        name (CharField): Verbose name is "Name". Max length is 255 and must be unique.
            Help text is "Name of the Order Type".
        description (TextField): Verbose name is "Description". Can be blank.
            Help text is "Description of the Order Type".

    Methods:
        __str__(self) -> str:
            Returns the name of the OrderType.
        get_absolute_url(self) -> str:
            Returns the absolute URL for the OrderType's detail view.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
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
        ordering = ["name"]
        verbose_name_plural = _("Order Types")

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

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    # General Information
    pro_number = models.CharField(
        _("Pro Number"),
        max_length=10,
        unique=True,
        editable=False,
        help_text=_("Pro Number of the Order"),
    )
    order_type = models.ForeignKey(
        OrderType,
        on_delete=models.PROTECT,
        verbose_name=_("Order Type"),
        related_name="orders",
        related_query_name="order",
        help_text=_("Order Type of the Order"),
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
        blank=True,
        null=True,
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
        blank=True,
        null=True,
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
        blank=True,
        null=True,
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
        ordering = ["-pro_number"]

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

        # Validate 'freight_charge_amount' is entered if 'rate_method' is 'FLAT'
        if (
            self.rate_method == RatingMethodChoices.FLAT
            and not self.freight_charge_amount
        ):
            raise ValidationError(
                {
                    "rate_method": _(
                        "Freight Rate Method is Flat but Freight Charge Amount is not set. Please try again."
                    )
                },
                code="invalid",
            )

        # Validate order not marked 'ready_to_bill' if 'status' is not COMPLETED
        if self.ready_to_bill and self.status != StatusChoices.COMPLETED:
            raise ValidationError(
                {
                    "ready_to_bill": _(
                        "Cannot mark an order ready to bill if status is not 'COMPLETED'. Please try again."
                    )
                },
                code="invalid",
            )

        # Validate 'mileage' is entered if 'rate_method' is 'PER_MILE'
        if self.rate_method == RatingMethodChoices.PER_MILE and not self.mileage:
            raise ValidationError(
                {
                    "rate_method": _(
                        "Rating Method 'PER-MILE' requires Mileage to be set. Please try again."
                    )
                },
                code="invalid",
            )

        # Validate compare origin and destination are not the same.
        if (
            self.organization.order_control.enforce_origin_destination
            and self.origin_location
            and self.destination_location
            and self.origin_location == self.destination_location
        ):
            raise ValidationError(
                {
                    "origin_location": _(
                        "Origin and Destination locations cannot be the same. Please try again."
                    )
                },
                code="invalid",
            )

        # Validate that origin_location or origin_address is provided.
        if not self.origin_location and not self.origin_address:
            raise ValidationError(
                {
                    "origin_address": _(
                        "Origin Location or Address is required. Please try again."
                    ),
                },
                code="invalid",
            )

        # Validate that destination_location or destination_address is provided.
        if not self.destination_location and not self.destination_address:
            raise ValidationError(
                {
                    "destination_address": _(
                        "Destination Location or Address is required. Please try again."
                    ),
                },
                code="invalid",
            )

        # Validate revenue code is entered if Order Control requires it for the organization.
        if self.organization.order_control.enforce_rev_code and not self.revenue_code:
            raise ValidationError(
                {"revenue_code": _("Revenue code is required. Please try again.")},
                code="invalid",
            )

        # Validate commodity is entered if Order Control requires it for the organization.
        if self.organization.order_control.enforce_commodity and not self.commodity:
            raise ValidationError(
                {"commodity": _("Commodity is required. Please try again.")},
                code="invalid",
            )

        super().clean()

    def save(self, **kwargs: Any) -> None:
        """Order save method

        Args:
            kwargs (Any): Keyword Arguments

        Returns:
            None
        """
        self.full_clean()

        # If order marked 'ready_to_bill' and organization order control 'auto_order_total' is set.
        # Calculate the total for the order and save it as the 'sub_total'.
        if self.ready_to_bill and self.organization.order_control.auto_order_total:
            self.sub_total = self.calculate_total()

        # If origin location is provided, set origin address to location address.
        if self.origin_location and not self.origin_address:
            self.origin_address = self.origin_location.get_address_combination

        # If destination location is provided, set destination address to location address.
        if self.destination_location and not self.destination_address:
            self.destination_address = self.destination_location.get_address_combination

        # If the commodity has a hazmat class, set the hazmat class on the order.
        if self.commodity and self.commodity.hazmat:
            self.hazmat = self.commodity.hazmat

        super().save(**kwargs)

    def get_absolute_url(self) -> str:
        """Get the absolute url for the Order

        Returns:
            str: Absolute url for the Order
        """
        return reverse("order-detail", kwargs={"pk": self.pk})

    def calculate_total(self) -> decimal.Decimal:
        """Calculate the sub_total for an order

        Calculate the sub_total for the order if the organization 'OrderControl'
        has auto_total_order as True. If not, this method will be skipped in the
        save method.

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


class OrderDocumentation(GenericModel):
    """
    Stores documentation related to a :model:`order.Order`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    order = models.ForeignKey(
        Order,
        on_delete=models.CASCADE,
        related_name="order_documentation",
        verbose_name=_("Order"),
    )
    document = models.FileField(
        _("Document"),
        upload_to=order_documentation_upload_to,
        validators=[FileExtensionValidator(allowed_extensions=["pdf"])],
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
        return textwrap.shorten(
            f"{self.order} - {self.document_class}", 50, placeholder="..."
        )

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

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
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
        ordering = ["-created"]

    def __str__(self) -> str:
        """String representation of the OrderComment

        Returns:
            str: String representation of the OrderComment
        """
        return textwrap.shorten(
            f"{self.order} - {self.comment_type}", 50, placeholder="..."
        )

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

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
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
        return textwrap.shorten(f"{self.order} - {self.charge}", 50, placeholder="...")

    def save(self, **kwargs: Any):
        """
        Save the AdditionalCharge
        """
        self.charge_amount = self.charge.charge_amount
        self.sub_total = self.charge_amount * self.unit

        super().save(**kwargs)

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

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    is_active = models.BooleanField(
        _("Is Active"),
        default=True,
        help_text=_("Is Active"),
    )
    code = models.CharField(
        _("Code"),
        max_length=5,
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
        ordering = ["code"]

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
