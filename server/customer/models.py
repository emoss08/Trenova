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
#
#  This file is part of Monta.
#
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right
#  to copy, modify, and redistribute the software, but only for non-production use or with a total
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the
#  software will be made available under version 2 or later of the GNU General Public License.
#  If you use the software in violation of this license, your rights under the license will be
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all
#  warranties and conditions. If you use this license's text or the "Business Source License" name
#  and trademark, you must comply with the Licensor's covenants, which include specifying the
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
#  Grant, and not modifying the license in any other way.

import textwrap
import uuid
from typing import Any, final

from django.core.exceptions import ValidationError
from django.db import models
from django.db.transaction import atomic
from django.urls import reverse
from django.utils.functional import cached_property
from django.utils.translation import gettext_lazy as _
from localflavor.us.models import USStateField, USZipCodeField
from phonenumber_field.modelfields import PhoneNumberField

from billing.models import AccessorialCharge, DocumentClassification
from utils.models import ChoiceField, GenericModel, PrimaryStatusChoices, Weekdays


@final
class FuelMethodChoices(models.TextChoices):
    """
    Fuel Method Choices
    """

    DISTANCE = "D", _("Distance")
    FLAT = "F", _("Flat")
    PERCENTAGE = "P", _("Percentage")


class Customer(GenericModel):  # type: ignore
    """
    Stores customer information for billing and invoicing
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
        help_text=_("Status of the Customer."),
        default=PrimaryStatusChoices.ACTIVE,
    )
    code = models.CharField(
        _("Code"),
        max_length=10,
        editable=False,
        help_text=_("Customer code"),
    )
    name = models.CharField(
        _("Name"),
        max_length=150,
        help_text=_("Customer name"),
        db_index=True,
    )
    address_line_1 = models.CharField(
        _("Address Line 1"),
        max_length=150,
        help_text=_("Address line 1"),
        blank=True,
    )
    address_line_2 = models.CharField(
        _("Address Line 2"),
        max_length=150,
        blank=True,
        help_text=_("Address line 2"),
    )
    city = models.CharField(
        _("City"),
        max_length=150,
        help_text=_("City"),
        blank=True,
    )
    state = USStateField(
        _("State"),
        help_text=_("State"),
        blank=True,
    )
    zip_code = USZipCodeField(
        _("Zip Code"),
        help_text=_("Zip code"),
        blank=True,
    )
    has_customer_portal = models.CharField(
        _("Has Customer Portal?"),
        help_text=_(
            "Designates whether this customer has the customer portal. "
            "active or not."
        ),
        choices=[("Y", "Yes"), ("N", "No")],
        max_length=1,
        default="N",
    )
    auto_mark_ready_to_bill = models.CharField(
        _("Auto Mark Ready to Bill?"),
        help_text=_(
            "Designates whether to automatically mark customer orders ready to bill. "
            "if the order passes customer billing requirements."
        ),
        choices=[("Y", "Yes"), ("N", "No")],
        max_length=1,
        default="N",
    )
    advocate = models.ForeignKey(
        to="accounts.User",
        on_delete=models.CASCADE,
        related_name="customers",
        related_query_name="customer",
        help_text=_("Customer Advocate assigned to this customer."),
        verbose_name=_("Customer Advocate"),
        blank=True,
        null=True,
    )

    class Meta:
        """
        Customer Metaclass
        """

        verbose_name = _("Customer")
        verbose_name_plural = _("Customers")
        ordering = ["-code"]
        db_table = "customer"
        constraints = [
            models.UniqueConstraint(
                fields=["code", "organization"],
                name="unique_customer_code_organization",
            )
        ]

    def __str__(self) -> str:
        """Customer string representation

        Returns:
            str: Customer string representation
        """
        return textwrap.shorten(
            f"Customer {self.code}: {self.name}", width=40, placeholder="..."
        )

    def save(self, *args: Any, **kwargs: Any) -> None:
        """Customer save method

        Args:
            *args (Any): Variable length argument list.
            **kwargs (Any): Arbitrary keyword arguments

        Returns:
            None: This function does return anything.
        """

        if not self.code:
            self.code = self.generate_customer_code().upper()

        super().save(*args, **kwargs)

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer instance

        Returns:
            str: Customer url
        """
        return reverse("customer-detail", kwargs={"pk": self.pk})

    def generate_customer_code(self) -> str:
        """Generate a unique code for a customer instance.

        This function uses the first three characters of the customer's name to form a base for the code.
        It then appends a zero-padded sequence number derived from the current count of customer objects plus one.

        If the newly formulated code already exists in the database, the base code itself is returned making this
        function provide a fallback.

        IMPORTANT NOTE:
            It's highly recommended to run this inside a transaction where the new customer instance gets created
            to ensure the count correctly reflects the current total number of instances.

        Returns:
            str: A unique or quasi-unique customer code.
        """
        code = self.name[:3]
        new_code = f"{code}{self.__class__.objects.count() + 1:04d}"

        return new_code if self.__class__.objects.filter(code=code).exists() else code

    @cached_property
    def get_address_combination(self) -> str:
        """
        Returns:
            str: String representation of the customer address.
        """
        return f"{self.address_line_1} {self.address_line_2} {self.city} {self.state} {self.zip_code}"

    @cached_property
    def get_address(self) -> str:
        """
        Returns:
            str: String representation of the customer address.
        """
        return f"{self.address_line_1} {self.address_line_2}"

    @cached_property
    def get_city_state_zip(self) -> str:
        """
        Returns:
            str: String representation of the customer address.
        """
        return f"{self.city}, {self.state} {self.zip_code}"


class CustomerEmailProfile(GenericModel):
    """
    Stores Customer Email Profile related to the :model:`customer.Customer`. model.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    customer = models.OneToOneField(
        to=Customer,
        on_delete=models.CASCADE,
        related_name="email_profile",
        help_text=_("Customer assigned to Email Profile"),
        verbose_name=_("Customer"),
        blank=True,
        null=True,
    )
    subject = models.CharField(
        _("Subject"),
        max_length=100,
        help_text=_("Subject"),
        blank=True,
    )
    comment = models.CharField(
        _("Comment"),
        max_length=100,
        help_text=_("Comment"),
        blank=True,
    )
    from_address = models.CharField(
        _("From Address"),
        max_length=255,
        help_text=_("From Address"),
        blank=True,
    )
    blind_copy = models.CharField(
        _("Blind Copy"),
        max_length=255,
        help_text=_("Comma separated list of email addresses"),
        blank=True,
    )
    read_receipt = models.BooleanField(
        _("Read Receipt"),
        help_text=_("Read Receipt"),
        default=False,
    )
    read_receipt_to = models.CharField(
        _("Read Receipt To"),
        max_length=255,
        help_text=_("Read Receipt To"),
        blank=True,
    )
    attachment_name = models.CharField(
        _("Attachment Name"),
        max_length=255,
        help_text=_("Attachment Name"),
        blank=True,
    )

    class Meta:
        """
        Metaclass for CustomerEmailProfile
        """

        verbose_name = _("Customer Email Profile")
        verbose_name_plural = _("Customer Email Profiles")
        db_table = "customer_email_profile"

    def __str__(self) -> str:
        """CustomerEmailProfile string representation

        Returns:
            str: Customer Email Profile string representation
        """
        return textwrap.shorten(
            f"Customer Email Profile for Customer {self.customer.name if self.customer else 'None'}",
            width=60,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer email profile instance

        Returns:
            str: Customer email profile url
        """
        return reverse("customer-email-profile-detail", kwargs={"pk": self.pk})

    def update_customer_email_profile(self, **kwargs: Any) -> None:
        """Updates customer email profile information

        Args:
            **kwargs (Any): Customer email profile information to update
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()


class CustomerRuleProfile(GenericModel):
    """
    Stores Customer Rule Profile information related to :model:`customer.Customer`. model.
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
        help_text=_("Name"),
    )
    customer = models.OneToOneField(
        to=Customer,
        on_delete=models.CASCADE,
        related_name="rule_profile",
        help_text=_("Customer assigned to Rule Profile"),
        verbose_name=_("Customer"),
        blank=True,
        null=True,
    )
    document_class = models.ManyToManyField(
        DocumentClassification,
        related_name="customer_rule_profile",
        verbose_name=_("Document Class"),
        help_text=_("Document class"),
    )

    class Meta:
        """
        Metaclass for CustomerRuleProfile
        """

        verbose_name = _("Customer Rule Profile")
        verbose_name_plural = _("Customer Rule Profiles")
        ordering = ["-name"]
        db_table = "customer_rule_profile"
        constraints = [
            models.UniqueConstraint(
                fields=["name", "organization"],
                name="unique_customer_rule_profile_organization",
            )
        ]

    def __str__(self) -> str:
        """CustomerRuleProfile string representation

        Returns:
            str: Customer Rule Profile string representation
        """
        return textwrap.shorten(
            f"Customer Rule profile {self.name}",
            width=40,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer rule profile instance

        Returns:
            str: Customer rule profile url
        """
        return reverse("customer-rule-profile-detail", kwargs={"pk": self.pk})

    @atomic
    def update_customer_rule_profile(self, **kwargs: Any) -> None:
        """Updates customer rule profile information

        Args:
            **kwargs (Any): Customer rule profile information to update
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()


class CustomerContact(GenericModel):
    """
    Stores contract information related to :model:`billing.Customer`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    customer = models.ForeignKey(
        Customer,
        on_delete=models.CASCADE,
        related_name="contacts",
        related_query_name="contact",
        verbose_name=_("Customer"),
        help_text=_("Customer"),
    )
    is_active = models.BooleanField(
        _("Active"),
        default=True,
        help_text=_(
            "Designates whether this customer contact should be treated as active. "
            "Unselect this instead of deleting customer contacts."
        ),
    )
    name = models.CharField(
        _("Name"),
        max_length=150,
        help_text=_("Contact name"),
        db_index=True,
    )
    email = models.EmailField(
        _("Email"),
        max_length=150,
        help_text=_("Contact email"),
        blank=True,
    )
    title = models.CharField(
        _("Title"),
        max_length=100,
        help_text=_("Contact title"),
        blank=True,
    )
    phone = PhoneNumberField(
        _("Phone Number"),
        max_length=20,
        help_text=_("Contact phone"),
        null=True,
        blank=True,
    )
    is_payable_contact = models.BooleanField(
        _("Payable Contact"),
        default=False,
        help_text=_("Designates whether this contact is the payable contact"),
    )

    class Meta:
        """
        Metaclass for CustomerContact
        """

        verbose_name = _("Customer Contact")
        verbose_name_plural = _("Customer Contacts")
        ordering = ["customer", "name"]
        db_table = "customer_contact"

    def __str__(self) -> str:
        """Customer Contact string representation

        Returns:
            str: Customer Contact string representation
        """
        return textwrap.shorten(
            f"Contact {self.name} for Customer {self.customer.name}",
            width=50,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer contact instance

        Returns:
            str: Customer contact url
        """
        return reverse("billing:customer-contact-detail", kwargs={"pk": self.pk})

    def clean(self) -> None:
        """Customer Contact clean method

        Returns:
            None

        Raises:
            ValidationError: If the customer contact is not valid.
        """
        super().clean()

        if self.is_payable_contact and not self.email:
            raise ValidationError(
                {
                    "email": _(
                        "Payable contact must have an email address. Please Try Again."
                    ),
                }
            )

    def update_customer_contact(self, **kwargs: Any) -> None:
        """Updates customer contact information

        Args:
            **kwargs (Any): Customer contact information to update
        """
        for key, value in kwargs.items():
            setattr(self, key, value)
        self.save()


class CustomerFuelProfile(GenericModel):
    """
    Stores Customer Fuel Profile information related to the :model:`billing.Customer`. model.
    """

    @final
    class DaysToUseChoices(models.TextChoices):
        """
        Days to use choices
        """

        DELIVERY = "D", _("Delivery Date")
        SHIPMENT = "S", _("Actual Shipment Date")
        SCHED_SHIPMENT = "C", _("Scheduled Shipment Date")
        ENTERED = "E", _("Date Entered")

    @final
    class FuelRegionChoices(models.TextChoices):
        """
        Fuel Region choices
        """

        USA = "USA", _("US National Average")
        EAST = "EAST", _("East Coast")
        NEW_ENGLAND = "NE", _("New England")
        GENERAL_ATL = "GA", _("General Atlantic")
        LOWER_ATL = "LA", _("Lower Atlantic")
        MIDWEST = "MW", _("Midwest")
        GULF_COAST = "GC", _("Gulf Coast")
        ROCKY_MOUNTAIN = "RM", _("Rocky Mountain")
        WEST_COAST = "WC", _("West Coast")
        CALIFORNIA = "CA", _("California")
        WEST_COAST_NO_LA = "WCL", _("West Coast (No LA)")

    @final
    class TableChoices(models.TextChoices):
        """
        Table choices
        """

        PERCENTAGE = "P", _("Percentage")
        FLAT = "F", _("Flat")
        DISTANCE = "D", _("Distance")
        TABLE = "T", _("Table")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    customer = models.ForeignKey(
        Customer,
        on_delete=models.CASCADE,
        related_name="customer_fuel_profiles",
        related_query_name="customer_fuel_profile",
        help_text=_("Customer"),
        verbose_name=_("Customer"),
    )
    fsc_code = models.ForeignKey(
        AccessorialCharge,
        on_delete=models.CASCADE,
        related_name="customer_fuel_profiles",
        related_query_name="customer_fuel_profile",
        help_text=_("FSC Code"),
        verbose_name=_("FSC Code"),
    )
    start_date = models.DateField(
        _("Start Date"),
        help_text=_("Start Date"),
    )
    end_date = models.DateField(
        _("End Date"),
        help_text=_("End Date"),
        null=True,
        blank=True,
    )
    days_to_use = ChoiceField(
        _("Days to Use"),
        choices=DaysToUseChoices.choices,
        help_text=_("Days to Use"),
    )
    order_type = models.ForeignKey(
        "order.OrderType",
        on_delete=models.CASCADE,
        related_name="customer_fuel_profiles",
        related_query_name="customer_fuel_profile",
        help_text=_("Order Type"),
        verbose_name=_("Order Type"),
    )
    fuel_region = ChoiceField(
        _("Fuel Region"),
        choices=FuelRegionChoices.choices,
        help_text=_("Fuel Region"),
    )
    fsc_method = ChoiceField(
        _("FSC Method"),
        choices=TableChoices.choices,
        help_text=_("FSC Method"),
    )
    customer_fuel_table = models.ForeignKey(
        "CustomerFuelTable",
        on_delete=models.CASCADE,
        related_name="customer_fuel_profiles",
        related_query_name="customer_fuel_profile",
        help_text=_("Customer Fuel Table"),
        verbose_name=_("Customer Fuel Table"),
        blank=True,
        null=True,
    )
    base_price = models.DecimalField(
        _("Base Price"),
        max_digits=16,
        decimal_places=3,
        help_text=_("Base Price"),
        blank=True,
        null=True,
    )
    fuel_variance = models.DecimalField(
        _("Fuel Variance"),
        max_digits=16,
        decimal_places=3,
        help_text=_("Fuel Variance ex: 0.02"),
        blank=True,
        null=True,
    )
    amount = models.DecimalField(
        _("Amount"),
        max_digits=16,
        decimal_places=5,
        help_text=_("Amount"),
        blank=True,
        null=True,
    )
    percentage = models.DecimalField(
        _("Percentage"),
        max_digits=6,
        decimal_places=2,
        help_text=_("Percentage"),
        blank=True,
        null=True,
    )

    class Meta:
        verbose_name = _("Customer Fuel Profile")
        verbose_name_plural = _("Customer Fuel Profiles")
        ordering = ["customer"]
        db_table = "customer_fuel_profile"

    def __str__(self) -> str:
        """Customer Fuel Profile string representation

        Returns:
            str: Customer Fuel Profile string representation
        """
        return textwrap.shorten(
            f"Customer Fuel Profile for Customer {self.customer.name}",
            width=40,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer fuel profile instance

        Returns:
            str: Customer fuel profile url
        """
        return reverse("customer-fuel-profile-details", kwargs={"pk": self.pk})

    def clean(self) -> None:
        """CustomerFuelProfile clean method

        Returns:
            None

        Raises:
            ValidationError: If the Customer Fuel Profile is not valid.

        """
        super().clean()

        if self.fsc_method == CustomerFuelProfile.TableChoices.TABLE:
            raise ValidationError(
                {
                    "customer_fuel_table": _(
                        "Customer Fuel Table is required if the FSC Method is Table. Please try again."
                    )
                },
                code="required",
            )


class CustomerFuelTable(GenericModel):
    """
    Stores Customer Fuel Profile Information related to the :model:`billing.Customer` model.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    name = models.CharField(
        _("Name"),
        max_length=10,
        help_text=_("Customer Fuel Profile Name"),
    )
    description = models.CharField(
        _("Description"),
        max_length=150,
        help_text=_("Customer Fuel Profile Description"),
        blank=True,
    )

    class Meta:
        verbose_name = _("Customer Fuel Table")
        verbose_name_plural = _("Customer Fuel Table")
        ordering = ["id"]
        db_table = "customer_fuel_table"
        constraints = [
            models.UniqueConstraint(
                fields=["name", "organization"],
                name="unique_customer_fuel_table_name_organization",
            )
        ]

    def __str__(self) -> str:
        """Customer Fuel Table string representation

        Returns:
            str: Customer Fuel Table string representation
        """
        return textwrap.shorten(
            f"Customer Fuel Table {self.name}",
            width=30,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer fuel profile instance

        Returns:
            str: Customer fuel profile url
        """
        return reverse("billing:customer-fuel-profile-detail", kwargs={"pk": self.pk})


class CustomerFuelTableDetail(GenericModel):
    """
    Stores detailed information related to the `customer.CustomerFuelTable` model.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    customer_fuel_table = models.ForeignKey(
        CustomerFuelTable,
        on_delete=models.CASCADE,
        related_name="customer_fuel_table_details",
        related_query_name="customer_fuel_table_detail",
        help_text=_("Customer Fuel Profile"),
        verbose_name=_("Customer Fuel Profile"),
    )
    amount = models.DecimalField(
        _("Amount"),
        max_digits=16,
        decimal_places=5,
        help_text=_("Amount"),
        blank=True,
        null=True,
    )
    method = ChoiceField(
        _("Method"),
        choices=FuelMethodChoices.choices,
        help_text=_("Method"),
    )
    start_price = models.DecimalField(
        _("Start Price"),
        max_digits=5,
        decimal_places=3,
        help_text=_("Start Price"),
        blank=True,
        null=True,
    )
    percentage = models.DecimalField(
        _("Percentage"),
        max_digits=6,
        decimal_places=2,
        help_text=_("Percentage"),
        blank=True,
        null=True,
    )

    class Meta:
        verbose_name = _("Customer Fuel Profile Detail")
        verbose_name_plural = _("Customer Fuel Profile Details")
        ordering = ["customer_fuel_table"]
        db_table = "customer_fuel_profile_detail"

    def __str__(self) -> str:
        """Customer Fuel Profile Detail string representation

        Returns:
            str: Customer Fuel Profile Detail string representation
        """
        return textwrap.shorten(
            f"Details for Customer fuel table {self.customer_fuel_table.name}",
            width=50,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer fuel profile detail instance

        Returns:
            str: Customer fuel profile detail url
        """
        return reverse("billing:customer-fuel-profile-detail", kwargs={"pk": self.pk})


class DeliverySlot(GenericModel):
    """
    Stores delivery slot information related to the :model:`billing.Customer` model.
    """

    customer = models.ForeignKey(
        Customer,
        on_delete=models.CASCADE,
        help_text=_("Customer"),
        verbose_name=_("Customer"),
    )
    day_of_week = models.PositiveSmallIntegerField(
        _("Day of Week"), choices=Weekdays.choices, help_text=_("Day of Week")
    )
    start_time = models.TimeField(_("Start Time"), help_text=_("Start Time"))
    end_time = models.TimeField(_("End Time"), help_text=_("End Time"))
    location = models.ForeignKey(
        "location.Location",
        on_delete=models.CASCADE,
        help_text=_("Location"),
        verbose_name=_("Location"),
    )

    class Meta:
        """
        Metaclass for the Delivery Slot model.
        """

        verbose_name = _("Delivery Slot")
        verbose_name_plural = _("Delivery Slots")
        ordering = ["customer", "day_of_week", "start_time", "end_time"]
        db_table = "delivery_slot"
        unique_together = ["customer", "day_of_week", "start_time", "end_time"]
        constraints = [
            models.UniqueConstraint(
                fields=[
                    "customer",
                    "day_of_week",
                    "start_time",
                    "end_time",
                    "location",
                ],
                name="unique_delivery_slot",
            ),
            # TODO(wolfred): Write test for this check constraint.
            # Check if start_time is less than end_time
            models.CheckConstraint(
                check=models.Q(start_time__lt=models.F("end_time")),
                name="start_time_lt_end_time",
            ),
        ]

    def __str__(self) -> str:
        """String representation of Delivery Slot

        Returns:
            str: Delivery Slot string representation
        """
        return textwrap.shorten(
            f"Delivery Slot for {self.customer.name} on {self.get_day_of_week_display()} from {self.start_time} to"
            f" {self.end_time}",
            width=50,
            placeholder="...",
        )

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular delivery slot instance

        Returns:
            str: Delivery slot url
        """
        return reverse("billing:delivery-slot-detail", kwargs={"pk": self.pk})
