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
from django_lifecycle import LifecycleModelMixin, hook, AFTER_CREATE

from utils.models import ChoiceField, GenericModel


@final
class IntegrationChoices(models.TextChoices):
    """
    Integration Choices
    """

    GOOGLE_MAPS = "google_maps", _("Google Maps")
    GOOGLE_PLACES = "google_places", _("Google Places")


@final
class IntegrationAuthTypes(models.TextChoices):
    """
    Integration Auth Types
    """

    NO_AUTH = "no_auth", _("No Auth")
    API_KEY = "api_key", _("API Key")
    BEARER_TOKEN = "bearer_token", _("Bearer Token")
    BASIC_AUTH = "basic_auth", _("Basic Auth")


class IntegrationVendor(LifecycleModelMixin, GenericModel):
    """
    Stores Integration vendor information related to an :model:`organization.Organization`.
    """

    id = models.UUIDField(
        _("ID"),
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    is_active = models.BooleanField(
        _("Is Active"),
        default=True,
        help_text=_("Designates whether this integration vendor is active."),
    )
    name = ChoiceField(
        _("Name"),
        choices=IntegrationChoices.choices,
        help_text=_("Name of the integration vendor."),
    )

    class Meta:
        """
        Metaclass for IntegrationVendor
        """

        verbose_name = _("Integration Vendor")
        verbose_name_plural = _("Integration Vendors")
        ordering = ["name"]

    def __str__(self) -> str:
        """Returns the name of the integration vendor.

        Returns:
            str: Name of the integration vendor.
        """
        return textwrap.shorten(self.name, width=50, placeholder="...")

    @hook(AFTER_CREATE)
    def create_integration_after_create(self) -> None:
        """Creates an Integration after creating an IntegrationVendor.

        Returns:
            None: None
        """
        Integration.objects.create(integration_vendor=self, organization=self.organization)

    def get_absolute_url(self) -> str:
        """Returns the absolute url for the integration vendor.

        Returns:
            str: Absolute url for the integration vendor.
        """
        return reverse("integration:integration_vendor_detail", kwargs={"pk": self.pk})


class Integration(GenericModel):
    """
    Stores Integration details related to an :model:`organization.Organization`
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    integration_vendor = models.OneToOneField(
        IntegrationVendor,
        on_delete=models.CASCADE,
        related_name="integration",
        verbose_name=_("Integration Vendor"),
        help_text=_("Integration Vendor for the Integration"),
        blank=True,
        null=True,
    )
    auth_type = ChoiceField(
        _("Auth Type"),
        choices=IntegrationAuthTypes.choices,
        help_text=_("Authentication type for the integration"),
        default=IntegrationAuthTypes.NO_AUTH,
    )
    login_url = models.URLField(
        _("Login URL"),
        max_length=255,
        blank=True,
        help_text=_("Login URL for the integration"),
    )
    auth_token = models.CharField(
        _("Auth Token"),
        max_length=255,
        help_text=_("Api or Bearer Token  for the specified integration"),
        blank=True,
    )
    username = models.CharField(
        _("Username"),
        max_length=255,
        help_text=_("Username for the specified integration"),
        blank=True,
    )
    password = models.CharField(
        _("Password"),
        max_length=255,
        help_text=_("Password for the specified integration"),
        blank=True,
    )
    client_id = models.CharField(
        _("Client ID"),
        max_length=255,
        blank=True,
        help_text=_("Client ID for the specified integration"),
    )
    client_secret = models.CharField(
        _("Client Secret"),
        max_length=255,
        blank=True,
        help_text=_("Client Secret for the specified integration"),
    )

    class Meta:
        """
        Metaclass for Integration
        """

        verbose_name = _("Integration")
        verbose_name_plural = _("Integrations")
        ordering = ["integration_vendor"]

    def __str__(self) -> str:
        """String representation of the Integration Model

        Returns:
            str: String representation of the Integration
        """
        return textwrap.wrap(self.integration_vendor.name, 50)[0]  # type: ignore

    def clean(self) -> None:
        """Clean method to validate the Integration Model

        Returns:
            None

        Raises:
            ValidationError: Validation Errors for the Integration Model
        """
        if (
            self.integration_vendor.name  # type: ignore
            in [
                IntegrationChoices.GOOGLE_MAPS,
                IntegrationChoices.GOOGLE_PLACES,
            ]
            and self.auth_type != IntegrationAuthTypes.API_KEY
        ):
            raise ValidationError(
                {
                    "auth_type": ValidationError(
                        _("API Key is required for Google Integrations"),
                        code="invalid",
                    )
                }
            )

        if (
            self.auth_type
            in [
                IntegrationAuthTypes.BEARER_TOKEN,
                IntegrationAuthTypes.API_KEY,
            ]
            and not self.auth_token
        ):
            raise ValidationError(
                {
                    "auth_token": ValidationError(
                        _("Auth Token required for Authentication Type."),
                        code="required",
                    )
                }
            )

        if self.auth_type == IntegrationAuthTypes.BASIC_AUTH and (
            not self.username or not self.password
        ):
            raise ValidationError(
                {
                    "username": ValidationError(
                        _("Username and Password required for Authentication Type."),
                        code="required",
                    ),
                    "password": ValidationError(
                        _("Username and Password required for Authentication Type."),
                        code="required",
                    ),
                }
            )

        if self.auth_type == IntegrationAuthTypes.NO_AUTH and (
            self.auth_token or self.username or self.password
        ):
            raise ValidationError(
                {
                    "auth_token": ValidationError(
                        _("Auth Token not required for Authentication Type."),
                        code="invalid",
                    ),
                    "username": ValidationError(
                        _("Username not required for Authentication Type."),
                        code="invalid",
                    ),
                    "password": ValidationError(
                        _("Password not required for Authentication Type."),
                        code="invalid",
                    ),
                }
            )

    def get_absolute_url(self) -> str:
        """
        Returns:
            str: Absolute URL for the Integration
        """
        return reverse("integration:integration-detail", kwargs={"pk": self.pk})


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

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    organization = models.OneToOneField(
        "organization.Organization",
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
