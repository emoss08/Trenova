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

from typing import final

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from django_extensions.db.models import TimeStampedModel

from .validators import validate_org_timezone


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
        return self.name

    def get_absolute_url(self) -> str:
        """
        Returns:
            str: The absolute url for the organization.
        """
        return reverse("organization:view", kwargs={"pk": self.pk})
