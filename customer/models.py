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
from typing import final

from django.conf import settings
from django.core.exceptions import ValidationError
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from localflavor.us.models import USStateField, USZipCodeField
from phonenumber_field.modelfields import PhoneNumberField

from billing.models import AccessorialCharge, DocumentClassification
from core.models import ChoiceField, GenericModel

User = settings.AUTH_USER_MODEL


@final
class FuelMethodChoices(models.TextChoices):
    """
    Fuel Method Choices
    """

    DISTANCE = "D", _("Distance")
    FLAT = "F", _("Flat")
    PERCENTAGE = "P", _("Percentage")


class Customer(GenericModel):
    """
    Stores customer information for billing and invoicing
    """

    is_active = models.BooleanField(
        _("Active"),
        default=True,
        help_text=_(
            "Designates whether this customer should be treated as active. "
            "Unselect this instead of deleting customers."
        ),
    )
    code = models.CharField(
        _("Code"),
        max_length=10,
        unique=True,
        editable=False,
        help_text=_("Customer code"),
    )
    name = models.CharField(
        _("Name"),
        max_length=150,
        help_text=_("Customer name"),
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

    class Meta:
        """
        Customer Metaclass
        """

        verbose_name = _("Customer")
        verbose_name_plural = _("Customers")
        ordering: list[str] = ["-code"]

    def __str__(self) -> str:
        """Customer string representation

        Returns:
            str: Customer string representation
        """
        return textwrap.wrap(f"{self.code} - {self.name}", 50)[0]

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer instance

        Returns:
            str: Customer url
        """
        return reverse("customer:customer-detail", kwargs={"pk": self.pk})


class CustomerBillingProfile(GenericModel):
    """
    Stores Billing Criteria related to the :model:`billing.Customer`. model.
    """

    customer = models.OneToOneField(
        Customer,
        on_delete=models.CASCADE,
        related_name="billing_profile",
        related_query_name="billing_profiles",
        help_text=_("Customer"),
        verbose_name=_("Customer"),
    )
    is_active = models.BooleanField(
        _("Active"),
        default=True,
        help_text=_(
            "Designates whether this customer billing profile should be treated as active. "
            "Unselect this instead of deleting customer billing profiles."
        ),
    )
    email_profile = models.ForeignKey(
        "CustomerEmailProfile",
        on_delete=models.CASCADE,
        related_name="billing_profiles",
        related_query_name="billing_profiles",
        help_text=_("Customer Email Profile"),
        verbose_name=_("Customer Email Profile"),
        blank=True,
        null=True,
    )
    rule_profile = models.ForeignKey(
        "CustomerRuleProfile",
        on_delete=models.CASCADE,
        related_name="billing_profiles",
        related_query_name="billing_profile",
        help_text=_("Rule Profile"),
        verbose_name=_("Rule Profile"),
        blank=True,
        null=True,
    )

    class Meta:
        """
        Metaclass for CustomerBillingProfile
        """

        verbose_name = _("Customer Billing Profile")
        verbose_name_plural = _("Customer Billing Profiles")
        ordering: list[str] = ["customer"]

    def __str__(self) -> str:
        """Customer Billing Profile string representation

        Returns:
            str: Customer Billing Profile string representation
        """
        return textwrap.wrap(f"{self.customer}", 50)[0]

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer billing profile instance

        Returns:
            str: Customer Billing Profile url
        """
        return reverse(
            "billing:customer-billing-profile-detail", kwargs={"pk": self.pk}
        )


class CustomerEmailProfile(GenericModel):
    """
    Stores Customer Email Profile related to the :model:`customer.Customer`. model.
    """

    name = models.CharField(
        _("Name"),
        max_length=50,
        help_text=_("Name"),
        unique=True,
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
        help_text=_("Blind Copy"),
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
        ordering: list[str] = ["-name"]

    def __str__(self) -> str:
        """CustomerEmailProfile string representation

        Returns:
            str: Customer Email Profile string representation
        """
        return textwrap.wrap(f"{self.name}", 50)[0]

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer email profile instance

        Returns:
            str: Customer email profile url
        """
        return reverse("billing:customer-email-profile", kwargs={"pk": self.pk})


class CustomerRuleProfile(GenericModel):
    """
    Stores Customer FTP Profile information related to :model:`customer.Customer`. model.
    """

    name = models.CharField(
        _("Name"),
        max_length=50,
        help_text=_("Name"),
        unique=True,
    )
    document_class = models.ManyToManyField(
        DocumentClassification,
        related_name="billing_profiles",
        related_query_name="billing_profile",
        verbose_name=_("Document Class"),
        help_text=_("Document class"),
    )

    class Meta:
        """
        Metaclass for CustomerRuleProfile
        """

        verbose_name = _("Customer Rule Profile")
        verbose_name_plural = _("Customer Rule Profiles")
        ordering: list[str] = ["-name"]

    def __str__(self) -> str:
        """CustomerRuleProfile string representation

        Returns:
            str: Customer Rule Profile string representation
        """
        return textwrap.wrap(f"{self.name}", 50)[0]

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer rule profile instance

        Returns:
            str: Customer rule profile url
        """
        return reverse("billing:customer-rule-profile", kwargs={"pk": self.pk})


class CustomerContact(GenericModel):
    """
    Stores contract information related to :model:`billing.Customer`.
    """

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
        unique=True,
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
        verbose_name = _("Customer Contact")
        verbose_name_plural = _("Customer Contacts")
        ordering: list[str] = ["customer", "name"]

    def __str__(self) -> str:
        """Customer Contact string representation

        Returns:
            str: Customer Contact string representation
        """
        return textwrap.wrap(f"{self.customer.code} - {self.name}", 50)[0]

    def clean(self) -> None:
        """Customer Contact clean method

        Returns:
            None

        Raises:
            ValidationError: If the customer contact is not valid.
        """
        if self.is_payable_contact and not self.email:
            raise ValidationError(
                {
                    "email": ValidationError(
                        _("Payable contact must have an email address"),
                        code="invalid",
                    )
                }
            )
        super().clean()

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer contact instance

        Returns:
            str: Customer contact url
        """
        return reverse("billing:customer-contact-detail", kwargs={"pk": self.pk})


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
        ordering: list[str] = ["customer"]

    def __str__(self) -> str:
        """Customer Fuel Profile string representation

        Returns:
            str: Customer Fuel Profile string representation
        """
        return textwrap.wrap(f"{self.customer}", 50)[0]

    def clean(self) -> None:
        """CustomerFuelProfile clean method

        Returns:
            None

        Raises:
            ValidationError: If the Customer Fuel Profile is not valid.

        """
        if self.fsc_method == CustomerFuelProfile.TableChoices.TABLE:
            raise ValidationError(
                ValidationError(
                    {
                        "customer_fuel_table": _(
                            "Customer Fuel Table is required if the FSC Method is Table."
                        )
                    },
                    code="required",
                )
            )
        super().clean()

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer fuel profile instance

        Returns:
            str: Customer fuel profile url
        """
        return reverse("billing:customer-fuel-profile", kwargs={"pk": self.pk})


class CustomerFuelTable(GenericModel):
    """
    Stores Customer Fuel Profile Information related to the :model:`billing.Customer` model.
    """

    id = models.CharField(
        _("ID"),
        max_length=10,
        unique=True,
        editable=False,
        primary_key=True,
        help_text=_("Customer Fuel Profile ID"),
    )
    name = models.CharField(
        _("Name"),
        max_length=10,
        help_text=_("Customer Fuel Profile Name"),
        unique=True,
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
        ordering: list[str] = ["id"]

    def __str__(self) -> str:
        """Customer Fuel Profile string representation

        Returns:
            str: Customer Fuel Profile string representation
        """
        return textwrap.wrap(f"{self.id} - {self.description}", 50)[0]

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
        ordering: list[str] = ["customer_fuel_table"]

    def __str__(self) -> str:
        """Customer Fuel Profile Detail string representation

        Returns:
            str: Customer Fuel Profile Detail string representation
        """
        return textwrap.wrap(f"{self.customer_fuel_table}", 50)[0]

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer fuel profile detail instance

        Returns:
            str: Customer fuel profile detail url
        """
        return reverse("billing:customer-fuel-profile-detail", kwargs={"pk": self.pk})
