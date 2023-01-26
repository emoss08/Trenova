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
from typing import Any, final

from django.core.exceptions import ValidationError
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from utils.models import ChoiceField, GenericModel


@final
class FuelMethodChoices(models.TextChoices):
    """
    A class representing the possible fuel method choices.

    This class inherits from the `models.TextChoices` class and defines three constants:
    - DISTANCE: representing a fuel method based on distance
    - FLAT: representing a flat rate fuel method
    - PERCENTAGE: representing a fuel method based on a percentage
    """

    DISTANCE = "D", _("Distance")
    FLAT = "F", _("Flat")
    PERCENTAGE = "P", _("Percentage")


@final
class BillingExceptionChoices(models.TextChoices):
    """
    A class representing the possible billing exception choices.

    This class inherits from the `models.TextChoices` class and defines five constants:
    - PAPERWORK: representing a billing exception related to paperwork
    - CHARGE: representing a billing exception resulting in a charge
    - CREDIT: representing a billing exception resulting in a credit
    - DEBIT: representing a billing exception resulting in a debit
    - OTHER: representing any other type of billing exception
    """

    PAPERWORK = "PAPERWORK", _("Paperwork")
    CHARGE = "CHARGE", _("Charge")
    CREDIT = "CREDIT", _("Credit")
    DEBIT = "DEBIT", _("Debit")
    OTHER = "OTHER", _("OTHER")


class ChargeType(GenericModel):
    """Class for storing other charge types.

    Attributes:
        id (models.UUIDField): Primary key for the charge type. It has a default value
        of a new UUID, is not editable, and is unique.
        name (models.CharField): The name of the charge type. It has a max length of
        50 and must be unique.
        description (models.CharField): The description of the charge type. It has a
        max length of 100 and is optional.

    Methods:
        str(self) -> str: Returns the string representation of the charge type, which
        is the first 50 characters of the name.
        get_absolute_url(self) -> str: Returns the absolute URL for the charge type.

    Meta:
        verbose_name (str): The singular form of the name for the charge type model.
        verbose_name_plural (str): The plural form of the name for the charge type model.
        ordering (List[str]): The default ordering for instances of the charge type m
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
        unique=True,
        help_text=_("The name of the charge type."),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        blank=True,
        help_text=_("The description of the charge type."),
    )

    class Meta:
        """
        Metaclass for Charge Type
        """

        verbose_name = _("Charge Type")
        verbose_name_plural = _("Charge Types")
        ordering = ["name"]

    def __str__(self) -> str:
        """Charge Type string representation

        Returns:
            str: Charge Type name
        """
        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Charge Type absolute URL

        Returns:
            str: Charge Type absolute URL
        """
        return reverse("billing:charge_type_detail", kwargs={"pk": self.pk})


class AccessorialCharge(GenericModel):
    """Class for storing information about other charges.

    Attributes:
        code (models.CharField): The code for the other charge. It has a max length of 50
        and must be unique. It is also the primary key for the model.
        is_detention (models.BooleanField): A boolean field indicating whether the other charge is
        a detention charge. It has a default value of False.
        charge_amount (models.DecimalField): The amount of the other charge. It has a max of 10
        digits, with 2 decimal places, and a default value of 1.00.
        method (ChoiceField): The method for calculating the other charge. It has a set of
        choices defined in the FuelMethodChoices class and a default value of
        FuelMethodChoices.DISTANCE.

    Methods:
        str(self) -> str: Returns the string representation of the other charge, which
        is the first 50 characters of the code.
        get_absolute_url(self) -> str: Returns the absolute URL for the other charge.

    Meta:
        verbose_name (str): The singular form of the name for the other charge model.
        verbose_name_plural (str): The plural form of the name for the other charge model.
        ordering (List[str]): The default ordering for instances of the other charge model.
    """

    code = models.CharField(
        _("Code"),
        max_length=50,
        unique=True,
        primary_key=True,
    )
    is_detention = models.BooleanField(
        _("Is Detention"),
        default=False,
    )
    charge_amount = models.DecimalField(
        _("Charge Amount"),
        max_digits=10,
        decimal_places=2,
        default=1.00,
        help_text=_("Charge Amount"),
    )
    method = ChoiceField(
        _("Method"),
        choices=FuelMethodChoices.choices,
        default=FuelMethodChoices.DISTANCE,
    )

    class Meta:
        """
        Metaclass for the AccessorialCharge model.
        """

        verbose_name = _("Other Charge")
        verbose_name_plural = _("Other Charges")
        ordering: list[str] = ["code"]

    def __str__(self) -> str:
        """Other Charge string representation

        Returns:
            str: Other Charge string representation
        """
        return textwrap.wrap(self.code, 50)[0]

    def get_absolute_url(self) -> str:
        """Other Charge absolute URL

        Returns:
            str: Other Charge absolute URL
        """
        return reverse("billing:other_charge_detail", kwargs={"pk": self.pk})


class DocumentClassification(GenericModel):
    """
    Stores Document Classification information.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    name = models.CharField(
        _("Name"),
        max_length=150,
        help_text=_("Document classification name"),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("Document classification description"),
    )

    class Meta:
        """
        Metaclass for the DocumentClassification model.
        """

        verbose_name = _("Document Classification")
        verbose_name_plural = _("Document Classifications")
        ordering = ["name"]

    def __str__(self) -> str:
        """Document classification string representation

        Returns:
            str: Document classification string representation
        """
        return textwrap.wrap(self.name, 50)[0]

    def clean(self) -> None:
        """DocumentClassification Clean Method

        Returns:
            None

        Raises:
            ValidationError: If Document Classification is not valid.
        """

        super().clean()

        if self.__class__.objects.filter(name=self.name).exclude(pk=self.pk).exists():
            raise ValidationError(
                {
                    "name": _(
                        "Document classification with this name already exists. Please try again."
                    ),
                },
            )

    def delete(self, *args: Any, **kwargs: Any) -> None:
        """DocumentClassification Delete Method

        Args:
            *args (Any): Arguments
            **kwargs (Any): Keyword Arguments

        Returns:
            None
        """
        if self.name == "CON":
            raise ValidationError(
                {
                    "name": _(
                        "Document classification with this name cannot be deleted. Please try again."
                    ),
                },
                code="invalid",
            )
        super().delete(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular document classification instance

        Returns:
            str: Document classification url
        """
        return reverse("billing:document-classification-detail", kwargs={"pk": self.pk})

    def update_doc_class(self, **kwargs: Any) -> None:
        """
        Updates the document classification with the given kwargs
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()
<<<<<<< Updated upstream
=======


class BillingQueue(GenericModel):
    """
    Class for storing information about the billing queue.
    """

    @final
    class BillTypeChoices(models.TextChoices):
        """
        Status choices for Order model
        """

        INVOICE = "INVOICE", _("Invoice")
        CREDIT = "CREDIT", _("Credit")
        DEBIT = "DEBIT", _("Debit")
        PREPAID = "PREPAID", _("Prepaid")
        OTHER = "OTHER", _("Other")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    order_type = models.ForeignKey(
        "order.OrderType",
        on_delete=models.RESTRICT,
        verbose_name=_("Order Type"),
        related_name="billing_queue",
        help_text=_("Assigned order type to the billing queue"),
    )
    order = models.ForeignKey(
        "order.Order",
        on_delete=models.RESTRICT,
        related_name="billing_queue",
        help_text=_("Assigned order to the billing queue"),
        verbose_name=_("Order"),
    )
    revenue_code = models.ForeignKey(
        "accounting.RevenueCode",
        on_delete=models.RESTRICT,
        related_name="billing_queue",
        verbose_name=_("Revenue Code"),
        help_text=_("Assigned revenue code to the billing queue"),
        blank=True,
        null=True,
    )
    customer = models.ForeignKey(
        "customer.Customer",
        on_delete=models.RESTRICT,
        related_name="billing_queue",
        help_text=_("Assigned customer to the billing queue"),
        verbose_name=_("Customer"),
    )
    invoice_number = models.CharField(
        _("Invoice Number"),
        max_length=50,
        blank=True,
        null=True,
        help_text=_("Invoice number for the billing queue"),
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
    bill_type = ChoiceField(
        _("Bill Type"),
        choices=BillTypeChoices.choices,
        default=BillTypeChoices.INVOICE,
        help_text=_("Bill type for the billing queue"),
    )
    bill_date = models.DateField(
        _("Billed Date"),
        null=True,
        blank=True,
        help_text=_("Date invoiced was billed."),
    )
    mileage = models.DecimalField(
        _("Total Mileage"),
        max_digits=10,
        decimal_places=2,
        default=0,
        help_text=_("Total Mileage"),
        blank=True,
        null=True,
    )
    worker = models.ForeignKey(
        "worker.Worker",
        on_delete=models.RESTRICT,
        related_name="billing_queue",
        help_text=_("Assigned worker to the billing queue"),
        verbose_name=_("Worker"),
    )
    commodity = models.ForeignKey(
        "commodities.Commodity",
        on_delete=models.RESTRICT,
        related_name="billing_queue",
        help_text=_("Assigned commodity to the billing queue"),
        verbose_name=_("Commodity"),
        blank=True,
        null=True,
    )
    commodity_descr = models.CharField(
        _("Commodity Description"),
        max_length=255,
        blank=True,
        null=True,
    )
    other_charge_total = models.DecimalField(
        _("Other Charge Total"),
        max_digits=10,
        decimal_places=2,
        default=0.00,
        blank=True,
        null=True,
        help_text=_("Other charge total for Order"),
    )
    freight_charge_amount = models.DecimalField(
        _("Freight Charge Amount"),
        max_digits=10,
        decimal_places=2,
        default=0,
        help_text=_("Freight Charge Amount"),
        blank=True,
        null=True,
    )
    total_amount = models.DecimalField(
        _("Total Amount"),
        max_digits=10,
        decimal_places=2,
        default=0.00,
        blank=True,
        null=True,
        help_text=_("Total amount for Order"),
    )
    is_summary = models.BooleanField(
        _("Is Summary"),
        default=False,
        help_text=_("Is the invoice going to be a summary bill."),
    )
    is_cancelled = models.BooleanField(
        _("Is Cancelled"),
        default=False,
        help_text=_("Is the invoice cancelled."),
    )
    bol_number = models.CharField(
        _("BOL Number"),
        max_length=255,
        help_text=_("BOL Number"),
    )
    user = models.ForeignKey(
        "accounts.User",
        on_delete=models.RESTRICT,
        related_name="billing_queue",
        help_text=_("Assigned user to the billing queue"),
        verbose_name=_("User"),
        blank=True,
        null=True,
    )

    class Meta:
        """
        Metaclass for the BillingQueue model.
        """

        verbose_name = _("Billing Queue")
        verbose_name_plural = _("Billing Queues")
        ordering = ["order"]

    def __str__(self) -> str:
        """String Representation of the BillingQueue model

        Returns:
            str: BillingQueue string representation
        """
        return textwrap.wrap(self.order.pro_number, 50)[0]

    def clean(self) -> None:
        """
        Clean method for the BillingQueue model.

        Raises:
            ValidationError: If order does not meet the requirements for billing.
        """
        super().clean()

        errors = []

        # If order is already billed raise ValidationError
        if self.order.billed:
            errors.append(
                _(
                    "Order has already been billed. Please try again with a different order."
                )
            )

        # If order is already transferred to billing raise ValidationError
        if self.order.transferred_to_billing:
            errors.append(
                _(
                    "Order has already been transferred to billing. Please try again with a different order."
                )
            )

        # If order is voided raise ValidationError
        if self.order.status == StatusChoices.VOIDED:
            errors.append(
                _(
                    "Order has been voided. Please try again with a different order."
                )
            )

        # If order is not ready to be billed raise ValidationError
        if self.order.ready_to_bill is False:
            errors.append(
                _(
                    "Order is not ready to be billed. Please try again with a different order."
                )
            )

        if errors:
            raise ValidationError({"order": errors})

    def save(self, **kwargs: Any) -> None:
        """Save method for the BillingQueue model.

        Args:
            **kwargs (Any): Keyword Arguments

        Returns:
            None
        """
        self.full_clean()

        # If order has `pieces`, set `pieces` to order `pieces`
        if self.order.pieces:
            self.pieces = self.order.pieces

        # If order has `weight`, set `weight` to order `weight`
        if self.order.weight:
            self.weight = self.order.weight

        # If order has `mileage`, set `mileage` to order `mileage`
        if self.order.mileage:
            self.mileage = self.order.mileage

        # If order has `revenue_code`, set `revenue_code` to order `revenue_code`
        if self.order.revenue_code:
            self.revenue_code = self.order.revenue_code

        # If commodity `description` is set, set `commodity_descr` to the description of the commodity
        if self.commodity.description:
            self.commodity_descr = self.commodity.description

        # if order has `bol_number`, set `bol_number` to `bol_number`
        if self.order.bol_number:
            self.bol_number = self.order.bol_number

        # If `bill_type` is not set, set `bill_type` to `INVOICE`
        if not self.bill_type:
            self.bill_type = self.BillTypeChoices.INVOICE

        self.freight_charge_amount = self.order.freight_charge_amount
        self.total_amount = self.order.sub_total
        self.other_charge_total = self.order.other_charge_amount

        super().save(**kwargs)


class BillingException(GenericModel):
    """
    Class for storing information about the billing exception.
    """

    @final
    class BillingExceptionChoices(models.TextChoices):
        """
        A class representing the possible billing exception choices.

        This class inherits from the `models.TextChoices` class and defines five constants:
        - PAPERWORK: representing a billing exception related to paperwork
        - CHARGE: representing a billing exception resulting in a charge
        - CREDIT: representing a billing exception resulting in a credit
        - DEBIT: representing a billing exception resulting in a debit
        - OTHER: representing any other type of billing exception
        """

        PAPERWORK = "PAPERWORK", _("Paperwork")
        CHARGE = "CHARGE", _("Charge")
        CREDIT = "CREDIT", _("Credit")
        DEBIT = "DEBIT", _("Debit")
        OTHER = "OTHER", _("OTHER")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    exception_type = ChoiceField(
        _("Exception Type"),
        choices=BillingExceptionChoices.choices,
        default=BillingExceptionChoices.PAPERWORK,
        help_text=_("Type of billing exception"),
    )
    order = models.ForeignKey(
        "order.Order",
        on_delete=models.RESTRICT,
        related_name="billing_exception",
        help_text=_("Assigned order to the billing exception"),
    )
    exception_message = models.TextField(
        _("Exception Message"),
        help_text=_("Message for the billing exception"),
        blank=True,
        null=True,
    )

    class Meta:
        """
        Metaclass for the BillingException model.
        """

        verbose_name = _("Billing Exception")
        verbose_name_plural = _("Billing Exceptions")
        ordering = ["order"]

    def __str__(self) -> str:
        """String Representation of the BillingException model

        Returns:
            str: BillingException string representation
        """
        return textwrap.wrap(self.order.pro_number, 50)[0]


class BillingHistory(GenericModel):
    """
    Class for storing information about the billing history.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        help_text=_("Unique identifier for the billing history"),
    )
    batch_name = models.CharField(
        _("Batch Name"),
        max_length=100,
        unique=True,
        null=True,
        blank=True,
        help_text=_("Name of the batch"),
    )
    order = models.ForeignKey(
        "order.Order",
        on_delete=models.RESTRICT,
        related_name="billing_history",
        help_text=_("Assigned order to the billing history"),
    )
    bill_type = ChoiceField(
        _("Bill Type"),
        choices=BillingQueue.BillTypeChoices.choices,
        default=BillingQueue.BillTypeChoices.INVOICE,
        help_text=_("Type of bill"),
    )
    sub_total = models.DecimalField(
        _("Sub Total"),
        max_digits=10,
        decimal_places=2,
        blank=True,
        null=True,
        help_text=_("Sub total for Order"),
    )

    class Meta:
        """
        Metaclass for the BillingHistory model.
        """

        verbose_name = _("Billing History")
        verbose_name_plural = _("Billing Histories")
        ordering = ["order"]

    def __str__(self) -> str:
        """String Representation of the BillingHistory model

        Returns:
            str: BillingHistory string representation
        """
        return textwrap.wrap(self.order.pro_number, 50)[0]

    def clean(self) -> None:
        """Clean method for the BillingHistory model.

        Returns:
            None

        Raises:
            ValidationError
        """
        if not self.order.billed:
            raise ValidationError(
                {
                    "order": _(
                        "Order has not been billed. Please try again with a different order."
                    ),
                },
            )

    def save(self, **kwargs: Any) -> None:
        """Save method for the BillingHistory model.

        Args:
            **kwargs (Any): Keyword Arguments

        Returns:
            None
        """

        self.full_clean()
        self.sub_total = round(self.order.sub_total, 2)
>>>>>>> Stashed changes
