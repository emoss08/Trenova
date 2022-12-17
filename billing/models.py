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
from typing import final

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
    """
    Stores Other Charge Types
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
        verbose_name = _("Charge Type")
        verbose_name_plural = _("Charge Types")
        ordering: list[str] = ["name"]

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
        verbose_name = _("Document Classification")
        verbose_name_plural = _("Document Classifications")
        ordering: list[str] = ["name"]

    def __str__(self) -> str:
        """Document classification string representation

        Returns:
            str: Document classification string representation
        """
        return textwrap.wrap(self.name, 50)[0]

    def clean(self) -> None:

        super().clean()
        if self.__class__.objects.filter(name=self.name).exclude(pk=self.pk).exists():
            raise ValidationError(
                {
                    "name": _("Document classification with this name already exists."),
                },
            )

    def get_absolute_url(self) -> str:
        """Returns the url to access a particular document classification instance

        Returns:
            str: Document classification url
        """
        return reverse("billing:document-classification-detail", kwargs={"pk": self.pk})
