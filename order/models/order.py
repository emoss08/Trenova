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
from django.core.exceptions import ValidationError
from django.db import models
from django.db.models.aggregates import Sum
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from order.models import choices, hazardous_material, order_control, stop

# from order.models.stop import Stop
from utils.models import ChoiceField, GenericModel

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
        choices=choices.StatusChoices.choices,
        default=choices.StatusChoices.NEW,
    )
    revenue_code = models.ForeignKey(
        "accounting.RevenueCode",
        on_delete=models.PROTECT,
        related_name="orders",
        related_query_name="order",
        verbose_name=_("Revenue Code"),
        help_text=_("Revenue Code of the Order"),
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
        "order.Commodity",
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
        "order.HazardousMaterial",
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

    def set_hazardous_class(self) -> Optional[hazardous_material.HazardousMaterial]:
        """Set the hazardous class from commodity

        if a commodity is selected automatically set the hazardous
        class from the relationship between commodity and
        HazardousMaterial.

        Returns:
            HazardousMaterial: Instance of the HazardousMaterial
        """
        if self.commodity.hazmat:
            self.hazmat_id = self.commodity.hazmat
        return self.hazmat_id

    def total_piece(self) -> int:
        """Get the total piece count for the order

        Returns:
            int: Total piece count for the order
        """
        return stop.Stop.objects.filter(movement__order__exact=self).aggregate(
            Sum("pieces")
        )["pieces__sum"]

    def total_weight(self) -> int:
        """Get the total weight for the order.

        Returns:
            int: Total weight for the order
        """
        return stop.Stop.objects.filter(movement__order__exact=self).aggregate(
            Sum("weight")
        )["weight__sum"]

    def set_address(self) -> None:
        """Set the address for the order

        Returns:
            None
        """
        o_control: order_control.OrderControl = order_control.OrderControl.objects.get(
            organization=self.organization
        )

        if o_control.auto_pop_address:
            self.origin_address = self.origin_location.get_address_combination
            self.destination_address = self.destination_location.get_address_combination

    def validate_freight_rate_method(self) -> None:
        """Validate the freight charge amount

        If the rate method is flat, the freight charge
        amount must be set.

        Returns:
            None

        Raises:
            ValidationError: If the freight charge amount is not set

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

    def validate_per_mile_rate_method(self) -> None:
        """Validate the per mile rate method

        If the rate method is per mile, the mileage must be set.

        Returns:
            None

        Raises:
            ValidationError: If the mileage is not set
        """
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

    def validate_ready_to_bill(self) -> None:
        """Validate the order is ready to be billed

        Order must be marked completed before it can be marked
        ready to bill.

        Returns:
            None

        Raises:
            ValidationError: If the order is not completed
        """
        if self.ready_to_bill and self.status != choices.StatusChoices.COMPLETED:
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

    def validate(self) -> None:
        """Validate the order

        Returns:
            None

        Raises:
            ValidationError: If the order is not valid
        """
        self.validate_freight_rate_method()
        self.validate_per_mile_rate_method()
        self.validate_ready_to_bill()

    def clean(self) -> None:
        """Order save method

        Returns:
            None

        Raises:
            ValidationError: If the Order is not valid
        """
        self.validate()

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

        if self.status == choices.StatusChoices.COMPLETED:
            self.pieces = self.total_piece()
            self.weight = self.total_weight()

        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """Get the absolute url for the Order

        Returns:
            str: Absolute url for the Order
        """
        return reverse("order-detail", kwargs={"pk": self.pk})


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
        "control_file.CommentType",
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
