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

from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _
from django_extensions.db.models import TimeStampedModel
from django_lifecycle import LifecycleModelMixin, hook, AFTER_SAVE
from localflavor.us.models import USStateField, USZipCodeField
from phonenumber_field.modelfields import PhoneNumberField

from .validators.organization import validate_org_timezone


class Organization(LifecycleModelMixin, TimeStampedModel):
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

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    name = models.CharField(
        _("Organization Name"),
        max_length=255,
    )
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
    logo = models.ImageField(
        _("Logo"), upload_to="organizations/logo/", null=True, blank=True
    )

    class Meta:
        """
        Metaclass for the Organization model
        """

        verbose_name = _("Organization")
        verbose_name_plural = _("Organizations")
        ordering = ["name"]

    def __str__(self) -> str:
        """
        Returns:
            str: String representation of the organization.
        """
        return textwrap.wrap(self.name, 50)[0]

    @hook(AFTER_SAVE, when="created")  # type: ignore
    def create_dispatch_control_after_save(self) -> None:
        """Create a dispatch control after the organization is created.

        Returns:
            None
        """
        from dispatch.models import DispatchControl
        if not DispatchControl.objects.filter(organization=self).exists():
            DispatchControl.objects.create(organization=self)

    @hook(AFTER_SAVE, when="created")  # type: ignore
    def create_order_control_after_save(self) -> None:
        """Create an order control after the organization is created.

        Returns:
            None
        """
        from order.models import OrderControl
        if not OrderControl.objects.filter(organization=self).exists():
            OrderControl.objects.create(organization=self)

    @hook(AFTER_SAVE, when="created")  # type: ignore
    def create_billing_control_after_save(self) -> None:
        """Create a billing control after the organization is created.

        Returns:
            None
        """
        from billing.models import BillingControl
        if not BillingControl.objects.filter(organization=self).exists():
            BillingControl.objects.create(organization=self)

    @hook(AFTER_SAVE, when="created")  # type: ignore
    def create_email_control_after_save(self) -> None:
        """Create an email control after the organization is created.

        Returns:
            None: None
        """
        if not EmailControl.objects.filter(organization=self).exists():
            EmailControl.objects.create(organization=self)

    def get_absolute_url(self) -> str:
        """
        Returns:
            str: The absolute url for the organization.
        """
        return reverse("organization:details", kwargs={"pk": self.pk})


class Depot(LifecycleModelMixin, TimeStampedModel):
    """
    Stores information about a specific depot inside a :model:`organization.Organization`
    Depots are commonly known as terminals or yards.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
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
        unique=True,
        help_text=_("The name of the depot."),
    )
    description = models.TextField(
        _("Depot Description"),
        max_length=255,
        help_text=_("The description of the depot."),
        blank=True,
    )

    class Meta:
        """
        Metaclass for the Depot model
        """

        verbose_name = _("Depot")
        verbose_name_plural = _("Depots")
        ordering = ["name"]

    def __str__(self) -> str:
        """Depot string representation.

        Returns:
            str: String representation of the depot.
        """
        return textwrap.wrap(self.name, 50)[0]

    @hook(AFTER_SAVE, when="created")  # type: ignore
    def create_depot_details_after_save(self) -> None:
        """Create a depot detail after the depot is created.

        Returns:
            None
        """
        if not DepotDetail.objects.filter(depot=self).exists():
            DepotDetail.objects.create(organization=self.organization, depot=self)

    def get_absolute_url(self) -> str:
        """Depot absolute URL

        Returns:
            str: The absolute url for the depot.
        """
        return reverse("organization:depot-detail", kwargs={"pk": self.pk})


class DepotDetail(TimeStampedModel):
    """
    Stores details for the :model:`organization.Depot` model.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
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
        related_name="details",
        related_query_name="detail",
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
        ordering = ["depot"]

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

        return reverse("organization:depot-details", kwargs={"pk": self.depot.pk})


class Department(models.Model):
    """
    Stores information about a department
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    organization = models.ForeignKey(
        Organization,
        on_delete=models.CASCADE,
        related_name="departments",
        related_query_name="department",
        verbose_name=_("Organization"),
        help_text=_("The organization that the department belongs to."),
    )
    depot = models.ForeignKey(
        Depot,
        on_delete=models.CASCADE,
        related_name="departments",
        related_query_name="department",
        verbose_name=_("Depot"),
        help_text=_("The depot that the department belongs to."),
        null=True,
        blank=True,
    )
    name = models.CharField(
        _("Name"),
        max_length=255,
        help_text=_("The name of the department"),
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("The description of the department"),
    )

    class Meta:
        """
        Metaclass for the Department model
        """

        verbose_name = _("Department")
        verbose_name_plural = _("Departments")

    def __str__(self) -> str:
        """Department string representation

        Returns:
            str: String representation of the Department
        """

        return textwrap.wrap(self.name, 30)[0]

    def get_absolute_url(self) -> str:
        """Absolute URL for the Department.

        Returns:
            str: Get the absolute url of the Department
        """

        return reverse("organization:department-detail", kwargs={"pk": self.pk})


class EmailProfile(TimeStampedModel):
    """
    Stores the email control information for a related :model:`organization.Organization`
    """

    @final
    class EmailProtocolChoices(models.TextChoices):
        """
        Choices that will be used for Email Protocol
        """

        SMTP = "SMTP", _("SMTP")
        UNENCRYPTED = "UNENCRYPTED", _("Unencrypted SMTP")
        STARTTLS = "STARTTLS", _("STARTTLS")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    name = models.CharField(
        _("Name"),
        max_length=255,
        help_text=_("The name of the Email Profile."),
    )
    organization = models.ForeignKey(
        Organization,
        on_delete=models.CASCADE,
        verbose_name=_("Organization"),
        related_name="email_profiles",
        help_text=_("The organization that the email profile belongs to."),
    )
    email = models.EmailField(
        _("Email"),
        max_length=255,
        help_text=_("The email address that will be used for outgoing email."),
    )
    protocol = models.CharField(
        _("Protocol"),
        choices=EmailProtocolChoices.choices,
        help_text=_("The protocol that will be used for outgoing email."),
        blank=True,
        max_length=12,
    )
    host = models.CharField(
        _("Host"),
        max_length=255,
        help_text=_("The host that will be used for outgoing email."),
        blank=True,
    )
    port = models.PositiveIntegerField(
        _("Port"),
        help_text=_("The port that will be used for outgoing email."),
        blank=True,
        null=True,
    )
    username = models.CharField(
        _("Username"),
        max_length=255,
        help_text=_("The username that will be used for outgoing email."),
        blank=True,
    )
    password = models.CharField(
        _("Password"),
        max_length=255,
        help_text=_("The password that will be used for outgoing email."),
        blank=True,
    )

    class Meta:
        """
        Metaclass for the EmailProfile model
        """

        verbose_name = _("Email Profile")
        verbose_name_plural = _("Email Profiles")
        ordering = ["email"]

    def __str__(self) -> str:
        """EmailProfile string representation.

        Returns:
            str: String representation of the email profile.
        """

        return textwrap.wrap(self.email, 50)[0]

    def get_absolute_url(self) -> str:
        """EmailProfile absolute URL

        Returns:
            str: The absolute url for the email profile.
        """

        return reverse("organization:email-profile-detail", kwargs={"pk": self.pk})


class EmailControl(TimeStampedModel):
    """
    Stores the email control information for a related :model:`organization.Organization`
    """

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
        related_name="email_control",
        help_text=_("The organization that the email control belongs to."),
    )
    billing_email_profile = models.ForeignKey(
        EmailProfile,
        on_delete=models.SET_NULL,
        verbose_name=_("Billing Email Profile"),
        related_name="billing_email_control",
        help_text=_("The email profile that will be used for billing emails."),
        null=True,
        blank=True,
    )

    class Meta:
        """
        Metaclass for the EmailControl model
        """

        verbose_name = _("Email Control")
        verbose_name_plural = _("Email Controls")

    def __str__(self) -> str:
        """EmailControl string representation.

        Returns:
            str: String representation of the email control.
        """

        return textwrap.wrap(self.organization.name, 50)[0]

    def get_absolute_url(self) -> str:
        """EmailControl absolute URL

        Returns:
            str: The absolute url for the email control.
        """

        return reverse("organization:email-control-detail", kwargs={"pk": self.pk})


class EmailLog(TimeStampedModel):
    """
    Stores the email log information for a related :model:`organization.Organization`
    """

    subject = models.CharField(
        _("Subject"),
        max_length=255,
        help_text=_("The subject of the email."),
    )
    to_email = models.EmailField(
        _("To Email"),
        max_length=255,
        help_text=_("The email address that the email was sent to."),
    )
    error = models.TextField(
        _("Error"),
        blank=True,
        help_text=_("The error that was returned from the email server."),
    )

    class Meta:
        """
        Metaclass for the EmailLog model
        """

        verbose_name = _("Email Log")
        verbose_name_plural = _("Email Logs")
        ordering = ["-created"]

    def __str__(self) -> str:
        """EmailLog string representation.

        Returns:
            str: String representation of the email log.
        """

        return textwrap.wrap(self.subject, 50)[0]

    def get_absolute_url(self) -> str:
        """EmailLog absolute URL

        Returns:
            str: The absolute url for the email log.
        """

        return reverse("email-log-detail", kwargs={"pk": self.pk})
