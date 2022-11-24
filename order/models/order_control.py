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

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from utils.models import GenericModel


class OrderControl(GenericModel):
    """
    Stores the order control information for a related :model:`organization.Organization`.
    """

    organization = models.OneToOneField(
        "organization.Organization",
        on_delete=models.CASCADE,
        verbose_name=_("Organization"),
        related_name="order_control",
        related_query_name="order_controls",
    )
    auto_rate_orders = models.BooleanField(
        _("Auto Rate"),
        default=True,
        help_text=_("Auto rate orders"),
    )
    calculate_distance = models.BooleanField(
        _("Calculate Distance"),
        default=True,
        help_text=_("Calculate distance for the order"),
    )
    enforce_customer = models.BooleanField(
        _("Enforce Customer"),
        default=False,
        help_text=_("Enforce Customer to being enter when entering an order."),
    )
    enforce_rev_code = models.BooleanField(
        _("Enforce Rev Code"),
        default=False,
        help_text=_("Enforce rev code code being entered when entering an order."),
    )
    enforce_shipper = models.BooleanField(
        _("Enforce Shipper"),
        default=False,
        help_text=_("Enforce shipper when putting in an order."),
    )
    enforce_cancel_comm = models.BooleanField(
        _("Enforce Voided Comm"),
        default=False,
        help_text=_("Enforce comment when cancelling an order."),
    )
    generate_routes = models.BooleanField(
        _("Generate Routes"),
        default=False,
        help_text=_("Automatically generate routes for order entry."),
    )
    auto_pop_address = models.BooleanField(
        _("Auto Populate Address"),
        default=True,
        help_text=_(
            "Auto populate address from location ID " "when entering an order."
        ),
    )
    auto_sequence_stops = models.BooleanField(
        _("Auto Sequence Stops"),
        default=True,
        help_text=_("Auto Sequence stops for the order and movements."),
    )
    auto_order_total = models.BooleanField(
        _("Auto Order Total"),
        default=True,
        help_text=_("Automate the order total amount calculation."),
    )

    class Meta:
        """
        Metaclass for OrderControl
        """

        verbose_name = _("Order Control")
        verbose_name_plural = _("Order Controls")
        ordering: list[str] = ["organization"]

    def __str__(self) -> str:
        """Order control string representation

        Returns:
            str: Order control string representation
        """
        return textwrap.wrap(self.organization.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Order control absolute url

        Returns:
            str: Order control absolute url
        """
        return reverse("order_control:detail", kwargs={"pk": self.pk})
