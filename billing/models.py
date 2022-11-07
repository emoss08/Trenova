# -*- coding: utf-8 -*-
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
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from localflavor.us.models import USStateField, USZipCodeField  # type: ignore

from control_file.models import CommentType
from core.models import GenericModel
from dispatch.models import DispatchControl
from organization.models import Depot

User = settings.AUTH_USER_MODEL


@final
class FuelMethodChoices(models.TextChoices):
    """
    Fuel Method Choices
    """

    DISTANCE = "D", _("Distance")
    FLAT = "F", _("Flat")
    PERCENTAGE = "P", _("Percentage")


class ChargeType(GenericModel):
    """
    Stores Other Charge Types
    """

    name = models.CharField(
        _("Name"),
        max_length=50,
        unique=True,
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        blank=True,
        null=True,
    )

    class Meta:
        verbose_name = _("Charge Type")
        verbose_name_plural = _("Charge Types")
        indexes: list[models.Index] = [
            models.Index(fields=["name"]),
        ]

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
    """
    Stores Other Charge information
    """

    code = models.CharField(
        _("Code"),
        max_length=50,
        unique=True,
        primary_key=True,
    )
    is_fuel_surcharge = models.BooleanField(
        _("Is Fuel Surcharge"),
        default=False,
    )
    is_detention = models.BooleanField(
        _("Is Detention"),
        default=False,
    )
    method = models.CharField(
        _("Method"),
        max_length=1,
        choices=FuelMethodChoices.choices,
        default=FuelMethodChoices.DISTANCE,
    )

    class Meta:
        verbose_name = _("Other Charge")
        verbose_name_plural = _("Other Charges")

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


class Customer(GenericModel):
    """
    Stores customer information
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
        primary_key=True,
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
    )
    state = USStateField(
        _("State"),
        help_text=_("State"),
    )
    zip_code = USZipCodeField(
        _("Zip Code"),
        help_text=_("Zip code"),
    )

    class Meta:
        verbose_name = _("Customer")
        verbose_name_plural = _("Customers")
        ordering: list[str] = ["code"]

    def __str__(self) -> str:
        """Customer string representation

        Returns:
            str: Customer string representation
        """
        return textwrap.wrap(f"{self.code} - {self.name}", 50)[0]

    def generate_customer_code(self) -> str:
        """Generate a unique code for the customer

        Returns:
            str: Customer code
        """
        code: str = self.name[:8].upper()
        new_code: str = f"{code}{Customer.objects.count()}"

        return code if not Customer.objects.filter(code=code).exists() else new_code

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer instance

        Returns:
            str: Customer url
        """
        return reverse("billing:customer-detail", kwargs={"pk": self.pk})


class CustomerBillingProfile(GenericModel):
    """
    Stores Billing Criteria related to the `billing.Customer` model.
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
    document_class = models.ManyToManyField(
        "DocumentClassification",
        related_name="billing_profiles",
        related_query_name="billing_profile",
        verbose_name=_("Document Class"),
        help_text=_("Document class"),
    )


class DocumentClassification(GenericModel):
    """
    Stores Document Classification information.
    """

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
        verbose_name = _("Document Classification")
        verbose_name_plural = _("Document Classifications")
        ordering: list[str] = ["name"]

    def __str__(self) -> str:
        """Document classification string representation

        Returns:
            str: Document classification string representation
        """
        return textwrap.wrap(f"{self.name}", 50)[0]

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular document classification instance

        Returns:
            str: Document classification url
        """
        return reverse("billing:document-classification-detail", kwargs={"pk": self.pk})


class CustomerFuelTable(GenericModel):
    """
    Stores Customer Fuel Profile Information
    """

    id = models.CharField(
        _("ID"),
        max_length=10,
        unique=True,
        editable=False,
        primary_key=True,
        help_text=_("Customer Fuel Profile ID"),
    )
    description = models.CharField(
        _("Description"),
        max_length=150,
        help_text=_("Customer Fuel Profile Description"),
    )

    class Meta:
        verbose_name = _("Customer Fuel Profile")
        verbose_name_plural = _("Customer Fuel Profiles")
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
    Stores detailed information related to the `CustomerFuelTable` model.
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
    method = models.CharField(
        _("Method"),
        max_length=1,
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


class CustomerFuelProfile(GenericModel):
    """
    Stores Customer Fuel Profile information
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
    )
    days_to_use = models.CharField(
        _("Days to Use"),
        max_length=1,
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
    fuel_region = models.CharField(
        _("Fuel Region"),
        max_length=4,
        choices=FuelRegionChoices.choices,
        help_text=_("Fuel Region"),
    )
    fsc_method = models.CharField(
        _("FSC Method"),
        max_length=1,
        choices=TableChoices.choices,
        help_text=_("FSC Method"),
    )
    customer_fuel_table = models.ForeignKey(
        CustomerFuelTable,
        on_delete=models.CASCADE,
        related_name="customer_fuel_profiles",
        related_query_name="customer_fuel_profile",
        help_text=_("Customer Fuel Profile"),
        verbose_name=_("Customer Fuel Profile"),
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
        _("Base Price"),
        max_digits=16,
        decimal_places=3,
        help_text=_("Base Price"),
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

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular customer fuel profile instance

        Returns:
            str: Customer fuel profile url
        """
        return reverse("billing:customer-fuel-profile", kwargs={"pk": self.pk})
