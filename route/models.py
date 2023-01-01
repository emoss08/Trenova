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

import uuid
import textwrap
from typing import final

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from utils.models import ChoiceField, GenericModel


class Route(GenericModel):
    """
    Stores route information related to `orders.Orders` model
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    origin = models.CharField(
        _("Origin"),
        max_length=255,
        blank=True,
        help_text=_("Origin of the route"),
    )
    destination = models.CharField(
        _("Destination"),
        max_length=255,
        blank=True,
        help_text=_("Destination of the route"),
    )
    total_mileage = models.DecimalField(
        _("Total Mileage"),
        max_digits=10,
        decimal_places=2,
        blank=True,
        null=True,
        help_text=_("Total Mile from origin to destination"),
    )
    duration = models.PositiveIntegerField(
        _("Duration"),
        null=True,
        blank=True,
        help_text=_("Duration of route from origin to destination"),
    )

    class Meta:
        """
        Metaclass for Route
        """

        verbose_name = _("Route")
        verbose_name_plural = _("Routes")
        ordering: tuple[str, ...] = ("origin", "destination")
        indexes: list[models.Index] = [
            models.Index(fields=["total_mileage", "duration"]),
        ]

    def __str__(self) -> str:
        """Route string representation

        Returns:
            str: Route string representation
        """
        return textwrap.wrap(f"{self.origin} - {self.destination}", 50)[0]

    def get_absolute_url(self) -> str:
        """Route absolute URL

        Returns:
            str: Route absolute URL
        """
        return reverse("route:detail", kwargs={"pk": self.pk})


class RouteControl(GenericModel):
    """
    Stores Route Control information related to a `organization.Organization`
    """

    @final
    class RouteAvoidanceChoices(models.TextChoices):
        """
        Google Route Avoidance Choices
        """

        TOLLS = "tolls", "Tolls"
        HIGHWAYS = "highways", "Highways"
        FERRIES = "ferries", "Ferries"

    @final
    class RouteModelChoices(models.TextChoices):
        """
        Google Route Model Choices
        """

        BEST_GUESS = "best_guess", "Best Guess"
        OPTIMISTIC = "optimistic", "Optimistic"
        PESSIMISTIC = "pessimistic", "Pessimistic"

    @final
    class RouteDistanceUnitChoices(models.TextChoices):
        """
        Google Route Distance Unit Choices
        """

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
        related_name="route_controls",
        related_query_name="route_control",
        verbose_name=_("Organization"),
        help_text=_("Organization related to this route control"),
    )
    mileage_unit = ChoiceField(
        _("Mileage Unit"),
        choices=RouteDistanceUnitChoices.choices,
        default=RouteDistanceUnitChoices.IMPERIAL,
        help_text=_("The mileage unit that the organization uses"),
    )
    traffic_model = ChoiceField(
        _("Traffic Model"),
        choices=RouteModelChoices.choices,
        default=RouteModelChoices.BEST_GUESS,
        help_text=_("The traffic model that the organization uses"),
    )
    generate_routes = models.BooleanField(
        _("Generate Routes"),
        default=False,
        help_text=_("Automatically generate routes for orders"),
    )
    avoid_tolls = models.BooleanField(
        _("Avoid Tolls"),
        default=True,
        help_text=_("Avoid tolls when generating routes"),
    )
    avoid_highways = models.BooleanField(
        _("Avoid Highways"),
        default=False,
        help_text=_("Avoid highways when generating routes"),
    )
    avoid_ferries = models.BooleanField(
        _("Avoid Ferries"),
        default=True,
        help_text=_("Avoid ferries when generating routes"),
    )

    class Meta:
        """
        Metaclass for RouteControl
        """

        verbose_name = _("Route Control")
        verbose_name_plural = _("Route Controls")
        ordering: tuple[str, ...] = ("organization",)

    def __str__(self) -> str:
        """Route Control string representation

        Returns:
            str: Route Control string representation
        """
        return str(self.organization)

    def get_absolute_url(self) -> str:
        """Route Control absolute URL

        Returns:
            str: Route Control absolute URL
        """
        return reverse("route:control-detail", kwargs={"pk": self.pk})
