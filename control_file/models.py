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

from core.models import ChoiceField, GenericModel
from organization.models import Organization


class GoogleAPI(GenericModel):
    """
    Stores the Google API information for a related :model:`organization.Organization`.
    """

    @final
    class GoogleRouteAvoidanceChoices(models.TextChoices):
        """Google Route Avoidance Choices"""

        TOLLS = "tolls", "Tolls"
        HIGHWAYS = "highways", "Highways"
        FERRIES = "ferries", "Ferries"

    @final
    class GoogleRouteModelChoices(models.TextChoices):
        """Google Route Model Choices"""

        BEST_GUESS = "best_guess", "Best Guess"
        OPTIMISTIC = "optimistic", "Optimistic"
        PESSIMISTIC = "pessimistic", "Pessimistic"

    @final
    class GoogleRouteDistanceUnitChoices(models.TextChoices):
        """Google Route Distance Unit Choices"""

        METRIC = "metric", "Metric"
        IMPERIAL = "imperial", "Imperial"

    organization = models.OneToOneField(
        Organization,
        on_delete=models.CASCADE,
        verbose_name=_("Organization"),
        related_name="google_api",
        related_query_name="google_apis",
    )
    api_key = models.CharField(
        _("API Key"),
        max_length=255,
        help_text=_("Google API Key for the organization."),
    )
    mileage_unit = ChoiceField(
        _("Mileage Unit"),
        choices=GoogleRouteDistanceUnitChoices.choices,
        default=GoogleRouteDistanceUnitChoices.IMPERIAL,
        help_text=_("The mileage unit that the organization uses"),
    )
    traffic_model = ChoiceField(
        _("Traffic Model"),
        choices=GoogleRouteModelChoices.choices,
        default=GoogleRouteModelChoices.BEST_GUESS,
        help_text=_("The traffic model that the organization uses"),
    )
    add_customer_location = models.BooleanField(
        _("Add Customer Location"),
        default=False,
        help_text=_("Add customer location through google places"),
    )
    add_location = models.BooleanField(
        _("Add Location"),
        default=False,
        help_text=_("Add location through google places"),
    )

    class Meta:
        """
        Metaclass for GoogleAPI
        """

        verbose_name = _("Google API")
        verbose_name_plural = _("Google APIs")
        ordering: list[str] = ["organization"]

    def __str__(self) -> str:
        """Google API string representation

        Returns:
            str: Google API string representation
        """
        return textwrap.wrap(self.organization.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Google API absolute url

        Returns:
            str: Google API absolute url
        """
        return reverse("google_api:detail", kwargs={"pk": self.pk})


class CommentType(GenericModel):
    """
    Stores the comment type information for a related :model:`organization.Organization`.
    """

    name = models.CharField(
        _("Name"),
        max_length=255,
        help_text=_("Comment type name"),
    )
    description = models.TextField(
        _("Description"),
        max_length=255,
        help_text=_("Comment type description"),
    )

    class Meta:
        """
        Metaclass for CommentType
        """

        verbose_name = _("Comment Type")
        verbose_name_plural = _("Comment Types")
        ordering: list[str] = ["organization"]

    def __str__(self) -> str:
        """Comment type string representation

        Returns:
            str: Comment type string representation
        """
        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Comment type absolute url

        Returns:
            str: Comment type absolute url
        """
        return reverse("comment_type:detail", kwargs={"pk": self.pk})
