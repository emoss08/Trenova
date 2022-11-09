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

from core.models import GenericModel

User = settings.AUTH_USER_MODEL


@final
class FuelMethodChoices(models.TextChoices):
    """
    Fuel Method Choices
    """

    DISTANCE = "D", _("Distance")
    FLAT = "F", _("Flat")
    PERCENTAGE = "P", _("Percentage")


@final
class BillingExceptionChoices(models.TextChoices):
    """
    Billing Exception Choices
    """

    PAPERWORK = "PAPERWORK", _("Paperwork")
    CHARGE = "CHARGE", _("Charge")
    CREDIT = "CREDIT", _("Credit")
    DEBIT = "DEBIT", _("Debit")
    OTHER = "OTHER", _("OTHER")


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


# class BillingQueue(GenericModel):
#     """
#     Stores billing information temporarily before moving to
#     Model: `billing.Billing`
#     """
#     ...


class InvoiceType(GenericModel):
    """
    Stores Invoice Type information.
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
    )

    class Meta:
        verbose_name = _("Invoice Type")
        verbose_name_plural = _("Invoice Types")
        indexes: list[models.Index] = [
            models.Index(fields=["name"]),
        ]

    def __str__(self) -> str:
        """Invoice Type string representation

        Returns:
            str: Invoice Type name
        """
        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Invoice Type absolute URL

        Returns:
            str: Invoice Type absolute URL
        """
        return reverse("billing:invoice_type_detail", kwargs={"pk": self.pk})


class InvoiceTemplate(GenericModel):
    """
    Stores Invoice Template information.
    """

    def invoice_template_upload_to(self, filename) -> str:
        """Invoice Template upload path

        Args:
            filename (str): Invoice Template filename

        Returns:
            str: Invoice Template upload path
        """
        return f"invoice_templates/{self.name}/{filename}"

    name = models.CharField(
        _("Name"),
        max_length=50,
        unique=True,
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        blank=True,
    )
    fields = models.JSONField(
        _("Fields"),
        default=dict,
        help_text=_("Fields to be included in the invoice"),
    )
    background_image = models.ImageField(
        _("Background Image"),
        upload_to=invoice_template_upload_to,
        blank=True,
        null=True,
    )
    invoice_type = models.ForeignKey(
        InvoiceType,
        on_delete=models.CASCADE,

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
