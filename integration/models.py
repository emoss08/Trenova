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
        _("Is Active"), default=False, help_text=_("Is the integration active?")
    )
    name = models.CharField(
        _("Name"),
        max_length=100,
        choices=IntegrationChoices.choices,
        unique=True,
        help_text=_("Name of the integration"),
    )
    api_key = models.CharField(
        _("API Key"),
        max_length=255,
        help_text=_("API Key for the specified integration"),
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
        Returns: String representation of the Integration
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
            if not self.api_key:
                raise ValidationError(
                    {"api_key": _("API Key is required for this integration")}
                )

    def save(self, **kwargs: Any) -> None:
        """save method for the Integration

        Args:
            **kwargs (Any): Keyword arguments

        Returns: None
        """
        self.full_clean()

        if self.api_key:
            # Activate the integration if an API key is provided
            self.is_active = True

        super().save(**kwargs)

    def get_absolute_url(self) -> str:
        """
        Returns: Absolute URL for the Integration
        """
        return reverse("integration_detail", kwargs={"pk": self.pk})
