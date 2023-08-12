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

import textwrap
import uuid
from typing import final

from django.core.exceptions import ValidationError
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
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


class IntegrationVendor(GenericModel):
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
        db_table = "integration_vendor"

    def __str__(self) -> str:
        """Returns the name of the integration vendor.

        Returns:
            str: Name of the integration vendor.
        """
        return textwrap.shorten(self.name, width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """Returns the absolute url for the integration vendor.

        Returns:
            str: Absolute url for the integration vendor.
        """
        return reverse("integration-vendors-detail", kwargs={"pk": self.pk})


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
        related_name="integration_vendor",
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
        db_table = "integration"

    def __str__(self) -> str:
        """String representation of the Integration Model

        Returns:
            str: String representation of the Integration
        """
        return textwrap.wrap(self.integration_vendor.name, 50)[0]  # type: ignore

    def get_absolute_url(self) -> str:
        """
        Returns:
            str: Absolute URL for the Integration
        """
        return reverse("integrations-detail", kwargs={"pk": self.pk})

    def clean(self) -> None:
        """Clean method to validate the Integration Model

        Returns:
            None

        Raises:
            ValidationError: Validation Errors for the Integration Model
        """

        errors = {}

        if (
            self.integration_vendor.name  # type: ignore
            in [
                IntegrationChoices.GOOGLE_MAPS,
                IntegrationChoices.GOOGLE_PLACES,
            ]
            and self.auth_type != IntegrationAuthTypes.API_KEY
        ):
            errors["auth_type"] = _("API Key is required for Google Integrations")

        if (
            self.auth_type
            in [
                IntegrationAuthTypes.BEARER_TOKEN,
                IntegrationAuthTypes.API_KEY,
            ]
            and not self.auth_token
        ):
            errors["auth_token"] = _("Auth Token required for Authentication Type.")

        if self.auth_type == IntegrationAuthTypes.BASIC_AUTH and (
            not self.username or not self.password
        ):
            errors["username"] = _(
                "Username and Password required for Authentication Type."
            )
            errors["password"] = _(
                "Username and Password required for Authentication Type."
            )

        if self.auth_type == IntegrationAuthTypes.NO_AUTH and (
            self.auth_token or self.username or self.password
        ):
            errors["auth_token"] = _("Auth Token not required for Authentication Type.")
            errors["username"] = _("Username not required for Authentication Type.")
            errors["password"] = _("Password not required for Authentication Type.")

        if errors:
            raise ValidationError(errors)


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
    name = models.CharField(
        _("Name"),
        max_length=50,
        help_text=_("Name of the Google API"),
        default="Google API",
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
    auto_geocode = models.BooleanField(
        _("Auto Geocode"),
        default=False,
        help_text=_(
            "This determines if locations will automatically be geocoded, once they are created/updated."
        ),
    )

    class Meta:
        """
        Metaclass for GoogleAPI
        """

        verbose_name = _("Google API")
        verbose_name_plural = _("Google APIs")
        db_table = "google_api"

    def __str__(self) -> str:
        """Google API string representation

        Returns:
            str: Google API string representation
        """
        return textwrap.shorten(self.name, width=30, placeholder="...")

    def get_absolute_url(self) -> str:
        """Google API absolute url

        Returns:
            str: Google API absolute url
        """
        return reverse("google-api-detail", kwargs={"pk": self.pk})
