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
from django.utils import timezone
from django.utils.translation import gettext_lazy as _
from django_lifecycle import BEFORE_CREATE, LifecycleModelMixin, hook

from integration.models import IntegrationChoices
from organization.models import Organization
from utils.models import ChoiceField, GenericModel


class DispatchControl(GenericModel):
    """
    Stores the dispatch control information for a related :model:`organization.Organization`.
    """

    @final
    class ServiceIncidentControlChoices(models.TextChoices):
        """
        Service Incident Control Choices
        """

        NEVER = "Never", _("Never")
        PICKUP = "Pickup", _("Pickup")
        DELIVERY = "Delivery", _("Delivery")
        PICKUP_DELIVERY = "Pickup and Delivery", _("Pickup and Delivery")
        ALL_EX_SHIPPER = "All except shipper", _("All except shipper")

    @final
    class DistanceMethodChoices(models.TextChoices):
        """
        Distance method choices for Order model
        """

        GOOGLE = "Google", _("Google")
        MONTA = "Monta", _("Monta")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    organization = models.OneToOneField(
        Organization,
        on_delete=models.CASCADE,
        verbose_name=_("Organization"),
        related_name="dispatch_control",
        related_query_name="dispatch_controls",
    )
    record_service_incident = ChoiceField(
        _("Record Service Incident"),
        choices=ServiceIncidentControlChoices.choices,
        default=ServiceIncidentControlChoices.NEVER,
    )
    grace_period = models.PositiveIntegerField(
        _("Grace Period"),
        default=0,
        help_text=_("Grace period for the service incident in minutes."),
    )
    deadhead_target = models.DecimalField(
        _("Deadhead Target"),
        max_digits=5,
        decimal_places=2,
        default=0.00,
        help_text=_("Deadhead Mileage target for the company."),
    )
    driver_assign = models.BooleanField(
        _("Enforce Driver Assign"),
        default=True,
        help_text=_("Enforce driver assign to orders for the company."),
    )
    trailer_continuity = models.BooleanField(
        _("Enforce Trailer Continuity"),
        default=False,
        help_text=_("Enforce trailer continuity for the company."),
    )
    distance_method = ChoiceField(
        _("Distance Method"),
        choices=DistanceMethodChoices.choices,
        default=DistanceMethodChoices.MONTA,
        help_text=_("Distance method for the company."),
    )
    dupe_trailer_check = models.BooleanField(
        _("Enforce Duplicate Trailer Check"),
        default=False,
        help_text=_("Enforce duplicate trailer check for the company."),
    )
    regulatory_check = models.BooleanField(
        _("Enforce Regulatory Check"),
        default=False,
        help_text=_("Enforce regulatory check for the company."),
    )
    prev_orders_on_hold = models.BooleanField(
        _("Prevent Orders On Hold"),
        default=False,
        help_text=_("Prevent dispatch of orders on hold for the company."),
    )
    generate_routes = models.BooleanField(
        _("Generate Routes"),
        default=False,
        help_text=_("Generate routes for the company."),
    )

    class Meta:
        """
        Metaclass for DispatchControl
        """

        verbose_name = _("Dispatch Control")
        verbose_name_plural = _("Dispatch Controls")
        ordering = ["organization"]

    def __str__(self) -> str:
        """Dispatch control string representation

        Returns:
            str: Dispatch control string representation
        """
        return textwrap.wrap(self.organization.name, 50)[0]

    def clean(self) -> None:
        """Dispatch control clean method

        Returns:
            None

        Raises:
            ValidationError: If the dispatch control is not valid.
        """
        super().clean()

        if self.distance_method == self.DistanceMethodChoices.GOOGLE and all(
            integration.integration_type != IntegrationChoices.GOOGLE_MAPS
            for integration in self.organization.integrations.all()
        ):
            raise ValidationError(
                {
                    "distance_method": _(
                        "Google Maps integration is not configured for the organization."
                        " Please configure the integration before selecting Google as "
                        "the distance method."
                    ),
                },
                code="invalid",
            )

    def get_absolute_url(self) -> str:
        """Dispatch control absolute URL

        Returns:
            str: Dispatch control absolute URL
        """
        return reverse("dispatch-control-detail", kwargs={"pk": self.pk})


class DelayCode(GenericModel):
    """
    Store Delay code information that can be used by :model:`ServiceIncident`.
    """

    code = models.CharField(
        _("Delay Code"),
        max_length=4,
        primary_key=True,
        unique=True,
        help_text=_("Delay code for the service incident."),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("Description for the delay code."),
    )
    f_carrier_or_driver = models.BooleanField(
        _("Fault of Carrier or Driver"),
        default=False,
        help_text=_("Fault is carrier or driver."),
    )

    class Meta:
        """
        Metaclass for DelayCode
        """

        verbose_name = _("Delay Code")
        verbose_name_plural = _("Delay Codes")
        ordering: list[str] = ["code"]

    def __str__(self) -> str:
        """Delay code string representation

        Returns:
            str: Delay code string representation
        """
        return textwrap.wrap(self.code, 50)[0]

    def get_absolute_url(self) -> str:
        """Delay code absolute URL

        Returns:
            str: Delay code absolute URL
        """
        return reverse("delay-codes-detail", kwargs={"pk": self.pk})


class FleetCode(GenericModel):
    """
    Stores the Fleet Code information.
    """

    code = models.CharField(
        _("Fleet Code"),
        max_length=4,
        primary_key=True,
        unique=True,
        help_text=_("Fleet code for the service incident."),
    )
    description = models.CharField(
        _("Description"),
        max_length=100,
        help_text=_("Description for the fleet code."),
    )
    is_active = models.BooleanField(
        _("Is Active"),
        default=True,
        help_text=_("Is the fleet code active."),
    )
    revenue_goal = models.DecimalField(
        _("Revenue Goal"),
        max_digits=10,
        decimal_places=2,
        default=0.00,
        help_text=_("Revenue goal for the fleet code."),
    )
    deadhead_goal = models.DecimalField(
        _("Deadhead Goal"),
        max_digits=10,
        decimal_places=2,
        default=0.00,
        help_text=_("Deadhead goal for the fleet code."),
    )
    mileage_goal = models.DecimalField(
        _("Mileage Goal"),
        max_digits=10,
        decimal_places=2,
        default=0.00,
        help_text=_("Mileage goal for the fleet code."),
    )

    class Meta:
        """
        Metaclass for FleetCode
        """

        verbose_name = _("Fleet Code")
        verbose_name_plural = _("Fleet Codes")
        ordering: list[str] = ["code"]

    def __str__(self) -> str:
        """Fleet code string representation

        Returns:
            str: Fleet code string representation
        """
        return textwrap.wrap(self.code, 50)[0]

    def get_absolute_url(self) -> str:
        """Fleet code absolute URL

        Returns:
            str: Fleet code absolute URL
        """
        return reverse("fleet-codes-detail", kwargs={"pk": self.pk})


class CommentType(GenericModel):
    """
    Stores the comment type information for a related :model:`organization.Organization`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
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
        return reverse("comment-types-detail", kwargs={"pk": self.pk})


class Rate(LifecycleModelMixin, GenericModel):
    """Stores the Rate information for a related :model:`customer.Customer`.

    The rate model stores the rate information for a related Customer. It is used to
    store information such as general billing information, lane specific billing information,
    commodity specific billing information and more.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    rate_number = models.CharField(
        _("Rate Number"),
        max_length=10,
        unique=True,
        editable=False,
        help_text=_("Rate Number for Rate"),
    )
    customer = models.ForeignKey(
        "customer.Customer",
        on_delete=models.SET_NULL,
        verbose_name=_("Customer"),
        related_name="rates",
        null=True,
        blank=True,
        help_text=_("Customer for Rate"),
    )
    effective_date = models.DateField(
        _("Effective Date"),
        help_text=_("Effective Date for Rate"),
        default=timezone.now,
    )
    expiration_date = models.DateField(
        _("Expiration Date"),
        help_text=_("Expiration Date for Rate"),
        default=timezone.now,
    )
    commodity = models.ForeignKey(
        "commodities.Commodity",
        on_delete=models.SET_NULL,
        verbose_name=_("Commodity"),
        related_name="rates",
        null=True,
        blank=True,
        help_text=_("Commodity for Rate"),
    )
    order_type = models.ForeignKey(
        "order.OrderType",
        on_delete=models.SET_NULL,
        verbose_name=_("Order Type"),
        related_name="rates",
        null=True,
        blank=True,
    )
    equipment_type = models.ForeignKey(
        "equipment.EquipmentType",
        on_delete=models.SET_NULL,
        verbose_name=_("Equipment Type"),
        related_name="rates",
        null=True,
        blank=True,
    )

    class Meta:
        """
        Metaclass for Rate
        """

        verbose_name = _("Rate")
        verbose_name_plural = _("Rates")
        ordering = ["rate_number"]

    def __str__(self) -> str:
        """Rate string representation

        Returns:
            str: Rate string representation
        """
        return textwrap.wrap(self.rate_number, 50)[0]

    def get_absolute_url(self) -> str:
        """Rate absolute url

        Returns:
            str: Rate absolute url
        """
        return reverse("rates-detail", kwargs={"pk": self.pk})

    @hook(BEFORE_CREATE)
    def set_rate_number_before_create(self) -> None:
        """Set rate number before create

        Returns:
            None
        """
        self.rate_number = self.generate_rate_number()

    @staticmethod
    def generate_rate_number() -> str:
        """Rate number generator

        Returns:
            str: Rate number
        """
        count = Rate.objects.count() + 1
        rate_number = f"R{count:05d}"

        while Rate.objects.filter(rate_number=rate_number).exists():
            count += 1
            rate_number = f"R{count:05d}"

        return rate_number
