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

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from django_extensions.db.models import TimeStampedModel  # type: ignore
from localflavor.us.models import USStateField, USZipCodeField  # type: ignore
from phonenumber_field.modelfields import PhoneNumberField  # type: ignore

from .validators.organization import validate_org_timezone


class Organization(TimeStampedModel):
    """
    Organization Model Fields
    """

    @final
    class OrganizationTypes(models.TextChoices):
        """
        Organization Type Choices
        """

        ASSET = "Asset", _("Asset")
        BROKERAGE = "Brokerage", _("Brokerage")
        BOTH = "Both", _("Both")

    @final
    class LanguageChoices(models.TextChoices):
        """
        Supported Language Choices for Monta
        """

        ENGLISH = "en", _("English")
        SPANISH = "es", _("Spanish")

    name = models.CharField(_("Organization Name"), max_length=255, unique=True)
    scac_code = models.CharField(
        max_length=4,
        verbose_name=_("SCAC Code"),
        help_text=_("The SCAC code for the organization."),
    )
    org_type = models.CharField(
        max_length=10,
        choices=OrganizationTypes.choices,
        default=OrganizationTypes.ASSET,
        verbose_name=_("Organization Type"),
        help_text=_("The type of organization."),
    )
    timezone = models.CharField(
        _("Timezone"),
        max_length=255,
        default="America/New_York",
        help_text=_("The timezone of the organization"),
        validators=[validate_org_timezone],
    )
    language = models.CharField(
        _("Language"),
        max_length=2,
        choices=LanguageChoices.choices,
        default=LanguageChoices.ENGLISH,
        help_text=_("The language of the organization"),
    )
    currency = models.CharField(
        _("Currency"),
        max_length=255,
        default="USD",
        help_text=_("The currency that the organization uses"),
    )
    date_format = models.CharField(
        _("Date Format"),
        max_length=255,
        default="MM/DD/YYYY",
        help_text=_("Date Format"),
    )
    time_format = models.CharField(
        _("Time Format"),
        max_length=255,
        default="HH:mm",
        help_text=_("Time Format"),
    )
    profile_picture = models.ImageField(
        _("Profile Picture"), upload_to="organizations/", null=True, blank=True
    )

    class Meta:
        """
        Metaclass for the Organization model
        """

        verbose_name = _("Organization")
        verbose_name_plural = _("Organizations")
        ordering: list[str] = ["name"]
        indexes: list[models.Index] = [
            models.Index(fields=["name"]),
        ]

    def __str__(self) -> str:
        """
        Returns:
            str: String representation of the organization.
        """
        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """
        Returns:
            str: The absolute url for the organization.
        """
        return reverse("organization:view", kwargs={"pk": self.pk})


class Depot(TimeStampedModel):
    """
    Stores information about a specific depot inside an `Organization`.
    Depots are commonly known as terminals or yards.
    """

    organization = models.ForeignKey(
        Organization,
        on_delete=models.CASCADE,
        related_name="depots",
        related_query_name="depot",
        verbose_name=_("Organization"),
        help_text=_("The organization that the depot belongs to."),
    )
    name = models.CharField(
        _("Depot Name"),
        max_length=255,
        help_text=_("The name of the depot."),
    )
    description = models.TextField(
        _("Depot Description"),
        max_length=255,
        help_text=_("The description of the depot."),
        blank=True,
        null=True,
    )

    class Meta:
        """
        Metaclass for the Depot model
        """

        verbose_name = _("Depot")
        verbose_name_plural = _("Depots")
        ordering: list[str] = ["name"]
        indexes: list[models.Index] = [
            models.Index(fields=["name"]),
        ]

    def __str__(self) -> str:
        """Depot string representation.

        Returns:
            str: String representation of the depot.
        """
        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Depot absolute URL

        Returns:
            str: The absolute url for the depot.
        """
        return reverse("organization:depot:view", kwargs={"pk": self.pk})


class DepotDetail(TimeStampedModel):
    """
    Stores details for the `Depot` model.
    """

    organization = models.ForeignKey(
        Organization,
        on_delete=models.CASCADE,
        related_name="depot_details",
        related_query_name="depot_detail",
        verbose_name=_("Organization"),
        help_text=_("The organization that the depot detail belongs to."),
    )
    depot = models.OneToOneField(
        Depot,
        on_delete=models.CASCADE,
        related_name="depot_details",
        related_query_name="depot_detail",
        verbose_name=_("Depot"),
        help_text=_("The depot that the depot detail belongs to."),
    )
    address_line_1 = models.CharField(
        _("Address Line 1"),
        max_length=255,
        help_text=_("The address line 1 of the depot."),
    )
    address_line_2 = models.CharField(
        _("Address Line 2"),
        max_length=255,
        help_text=_("The address line 2 of the depot."),
        blank=True,
        null=True,
    )
    city = models.CharField(
        _("City"),
        max_length=255,
        help_text=_("The city of the depot."),
    )
    state = USStateField(
        _("State"),
        blank=True,
        null=True,
        help_text=_("The state of the depot."),
    )
    zip_code = USZipCodeField(
        _("Zip Code"),
        blank=True,
        null=True,
        help_text=_("The zip code of the depot."),
    )
    phone_number = PhoneNumberField(
        _("Phone Number"),
        blank=True,
        null=True,
        help_text=_("The phone number of the depot."),
    )
    alternate_phone_number = PhoneNumberField(
        _("Alternate Phone Number"),
        blank=True,
        null=True,
        help_text=_("The alternate phone number of the depot."),
    )
    fax_number = PhoneNumberField(
        _("Fax Number"),
        blank=True,
        null=True,
        help_text=_("The fax number of the depot."),
    )

    class Meta:
        """
        Metaclass for the DepotDetail model
        """

        verbose_name = _("Depot Detail")
        verbose_name_plural = _("Depot Details")
        ordering: list[str] = ["depot"]
        indexes: list[models.Index] = [
            models.Index(fields=["depot"]),
        ]

    def __str__(self) -> str:
        """DepotDetail string representation.

        Returns:
            str: String representation of the depot detail.
        """
        return textwrap.wrap(self.depot.name, 50)[0]

    def get_absolute_url(self) -> str:
        """DepotDetail absolute URL

        Returns:
            str: The absolute url for the depot detail.
        """
        return reverse("organization:depot:view", kwargs={"pk": self.depot.pk})
