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

from typing import Any, final

from django.core.exceptions import ValidationError
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from django_extensions.db.models import TimeStampedModel

from organization.models import Organization


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
    OAUTH = "oauth", _("OAuth")
    OAUTH2 = "oauth2", _("OAuth 2.0")
    BEARER_TOKEN = "bearer_token", _("Bearer Token")
    BASIC_AUTH = "basic_auth", _("Basic Auth")
    DIGEST_AUTH = "digest_auth", _("Digest Auth")
    HAWK_AUTH = "hawk_auth", _("Hawk Auth")
    AWS_SIG4 = "aws_sig4", _("AWS Sig4")
    NTLM_AUTH = "ntlm_auth", _("NTLM Auth")
    AKAMAI = "akamai_edgegrid", _("Akamai EdgeGrid")


class Integration(TimeStampedModel):
    """
    Integration Model Fields
    """

    organization = models.ForeignKey(
        Organization,
        on_delete=models.CASCADE,
        related_name="integrations",
        related_query_name="integration",
        verbose_name=_("Organization"),
    )
    is_active = models.BooleanField(
        _("Is Active"),
        default=False,
        help_text=_("Is the integration active?")
    )
    name = models.CharField(
        _("Name"),
        max_length=100,
        choices=IntegrationChoices.choices,
        unique=True,
        help_text=_("Name of the integration"),
    )
    auth_type = models.CharField(
        _("Auth Type"),
        max_length=100,
        choices=IntegrationAuthTypes.choices,
        help_text=_("Authentication type for the integration"),
        default=IntegrationAuthTypes.NO_AUTH,
    )
    auth_token = models.CharField(
        _("API Key"),
        max_length=255,
        help_text=_("API Key for the specified integration"),
        null=True,
        blank=True,
    )
    username = models.CharField(
        _("Username"),
        max_length=255,
        help_text=_("Username for the specified integration"),
        null=True,
        blank=True,
    )
    password = models.CharField(
        _("Password"),
        max_length=255,
        help_text=_("Password for the specified integration"),
        null=True,
        blank=True,
    )
    client_id = models.CharField(
        _("Client ID for the specified integration"),
        max_length=255,
        null=True,
        blank=True,
    )
    client_secret = models.CharField(
        _("Client Secret for the specified integration"),
        max_length=255,
        null=True,
        blank=True,
    )

    class Meta:
        """
        Metaclass for Integration
        """

        verbose_name = _("Integration")
        verbose_name_plural = _("Integrations")
        ordering: list[str] = ["name"]
        indexes: list[models.Index] = [
            models.Index(fields=["name"]),
        ]

    def __str__(self) -> str:
        """
        Returns (str): String representation of the Integration
        """
        return self.name

    def clean(self) -> None:
        """Validation for the Integrations

        Returns: None

        Raises: ValidationError
        """

        if self.name in [
            IntegrationChoices.GOOGLE_MAPS,
            IntegrationChoices.GOOGLE_PLACES,
        ]:
            if not self.auth_token:
                raise ValidationError(
                    {"auth_token": _("API Key is required for Google integrations")}
                )


def get_absolute_url(self) -> str:
    """
    Returns (str): Absolute URL for the Integration
    """
    return reverse("integration:integration_detail", kwargs={"pk": self.pk})
