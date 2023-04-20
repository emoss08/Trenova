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

from __future__ import annotations

import decimal
import textwrap
import uuid
from typing import Any, final

from django.conf import settings
from django.core.validators import FileExtensionValidator
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from djmoney.models.fields import MoneyField

from utils.models import (
    ChoiceField,
    GenericModel,
    RatingMethodChoices,
    StatusChoices,
)

User = settings.AUTH_USER_MODEL


def order_documentation_upload_to(instance: OrderDocumentation, filename: str) -> str:
    """Returns the path to upload the order documentation to.

    Upload the order documentation to the order documentation directory
    and name the file with the order id and the filename.

    Args:
        instance (Order): The instance of the Order.
        filename (str): The name of the file.

    Returns:
        Upload path for the order documentation to be stored. For example.

        `Order_documentation/M000123/invoice-12341.pdf`

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
    check_for_duplicate_bol = models.BooleanField(
        _("Check for Duplicate BOL"),
        default=False,
        help_text=_("Check for duplicate BOL numbers when entering an order."),
    )
    remove_orders = models.BooleanField(
        _("Ability to Remove Orders"),
        default=False,
        help_text=_(
            "Ability to remove orders from system. This will disallow the removal of Orders, Movements and Stops"
        ),
    )

    class Meta:
        """
        Metaclass for OrderControl
        """

        verbose_name = _("Order Control")
        verbose_name_plural = _("Order Controls")
        ordering = ["organization"]
        db_table = "order_control"

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
            `/order_control/edd1e612-cdd4-43d9-b3f3-bc099872088b/`
        """
        return reverse("order_control:detail", kwargs={"pk": self.pk})


class OrderType(GenericModel):  # type:ignore
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
        help_text=_("Name of the Order Type"),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Description of the Order Type"),
    )

    class Meta:
        """
        Metaclass for OrderType model
        """

        verbose_name = _("Order Type")
        verbose_name_plural = _("Order Types")
        ordering = ["name"]
        db_table = "order_type"
        constraints = [
            models.UniqueConstraint(
                fields=["name", "organization"],
                name="unique_order_type_name",
            )
        ]

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
        return reverse("order-types-detail", kwargs={"pk": self.pk})


class Order(GenericModel):  # type:ignore
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
    rate = models.ForeignKey(
        "dispatch.Rate",
        on_delete=models.RESTRICT,
        related_name="orders",
        related_query_name="order",
        verbose_name=_("Rate"),
        help_text=_("Associated Rate to the Order."),
        blank=True,
        null=True,
    )
    mileage = models.FloatField(
        _("Total Mileage"),
        default=0,
        help_text=_("Total Mileage"),
        blank=True,
        null=True,
    )
    other_charge_amount = MoneyField(
        _("Additional Charge Amount"),
        max_digits=19,
        decimal_places=4,
        default=0,
        help_text=_("Additional Charge Amount"),
        default_currency="USD",
    )
    freight_charge_amount = MoneyField(
        _("Freight Charge Amount"),
        max_digits=19,
        decimal_places=4,
        default=0,
        help_text=_("Freight Charge Amount"),
        default_currency="USD",
        blank=True,
        null=True,
    )
    rate_method = ChoiceField(
        _("Rating Method"),
        blank=True,
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
    sub_total = MoneyField(
        _("Sub Total Amount"),
        max_digits=19,
        decimal_places=4,
        default=0,
        help_text=_("Sub Total Amount"),
        default_currency="USD",
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
    voided_comm = models.CharField(
        _("Voided Comment"),
        max_length=100,
        blank=True,
        help_text=_("Voided Comment"),
    )
    auto_rate = models.BooleanField(
        _("Auto Rate"),
        default=True,
        help_text=_("Determines whether order will be auto-rated by entered rate."),
    )

    class Meta:
        """
        Metaclass for the Order model
        """

        verbose_name = _("Order")
        verbose_name_plural = _("Orders")
        ordering = ["pro_number"]
        db_table = "order"
        constraints = [
            models.UniqueConstraint(
                fields=["pro_number", "organization"],
                name="unique_order_number_per_organization",
            )
        ]

    def __str__(self) -> str:
        """String representation of the Order

        Returns:
            str: String representation of the Order
        """
        return textwrap.wrap(self.pro_number, 10)[0]

    def clean(self) -> None:
        """Order clean method

        Returns:
            None

        Raises:
            ValidationError: If the Order is not valid
        """
        from order.validation import OrderValidation

        super().clean()
        OrderValidation(order=self)

    def save(self, *args, **kwargs) -> None:
        from dispatch.services import transfer_rate_details
        from route.services import get_mileage

        if self.auto_rate:
            transfer_rate_details.transfer_rate_details(order=self)

        self.sub_total = self.calculate_total()

        self.mileage = get_mileage(order=self)
        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """Get the absolute url for the Order

        Returns:
            str: Absolute url for the Order
        """
        return reverse("order-detail", kwargs={"pk": self.pk})

    def calculate_total(self) -> Any | decimal.Decimal:
        """Calculate the sub_total for an order

        Calculate the sub_total for the order if the organization 'OrderControl'
        has auto_total_order as True. If not, this method will be skipped in the
        save method.

        Returns:
            decimal.Decimal: The total for the order
        """

        # Handle the flat fee rate calculation

        if self.freight_charge_amount and self.rate_method == RatingMethodChoices.FLAT:
            return self.freight_charge_amount + self.other_charge_amount

        # Handle the mileage rate calculation
        if (
            self.freight_charge_amount
            and self.mileage
            and self.rate_method
            and self.rate_method == RatingMethodChoices.PER_MILE
        ):
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
        db_table = "order_documentation"

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
        db_table = "order_comment"

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


class AdditionalCharge(GenericModel):  # type: ignore
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
    accessorial_charge = models.ForeignKey(
        "billing.AccessorialCharge",
        on_delete=models.CASCADE,
        related_name="additional_charges",
        related_query_name="additional_charge",
        verbose_name=_("Charge"),
        help_text=_("Charge"),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("Description of the charge"),
        blank=True,
    )
    charge_amount = MoneyField(
        _("Charge Amount"),
        max_digits=19,
        decimal_places=4,
        default_currency="USD",
        help_text=_("Charge Amount"),
        null=True,
        blank=True,
    )
    unit = models.PositiveIntegerField(
        _("Unit"),
        default=1,
        help_text=_("Number of units to be charged"),
    )
    sub_total = MoneyField(
        _("Sub Total"),
        max_digits=19,
        decimal_places=4,
        default_currency="USD",
        help_text=_("Sub Total"),
        null=True,
        blank=True,
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
        db_table = "additional_charge"

    def __str__(self) -> str:
        """String representation of the AdditionalCharges

        Returns:
            str: String representation of the AdditionalCharges
        """
        return textwrap.shorten(
            f"{self.order} - {self.accessorial_charge}", 50, placeholder="..."
        )

    def save(self, *args: Any, **kwargs: Any) -> None:
        """
        Save the AdditionalCharge
        """
        self.charge_amount = self.accessorial_charge.charge_amount
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
        db_table = "reason_code"
        constraints = [
            models.UniqueConstraint(
                fields=["code", "organization"],
                name="unique_reason_code_organization",
            )
        ]

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
        return reverse("reason-codes-detail", kwargs={"pk": self.pk})
