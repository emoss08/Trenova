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
from typing import Any, final

from django.db import models
from django.urls import reverse
from django.utils.functional import cached_property
from django.utils.translation import gettext_lazy as _
from django_extensions.db.models import TimeStampedModel
from localflavor.us.models import USStateField, USZipCodeField
from phonenumber_field.modelfields import PhoneNumberField

from .services.table_choices import TABLE_NAME_CHOICES
from .validators import validate_format_string, validate_org_timezone


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
    dot_number = models.PositiveIntegerField(
        _("DOT Number"),
        null=True,
        blank=True,
        help_text=_("The DOT number for the organization."),
    )
    address_line_1 = models.CharField(
        _("Address line 1"),
        max_length=255,
        help_text=_("The address line 1 of the organization."),
        blank=True,
    )
    address_line_2 = models.CharField(
        _("Address line 2"),
        max_length=255,
        blank=True,
        help_text=_("The address line 2 of the organization."),
    )
    city = models.CharField(
        _("City"),
        max_length=255,
        help_text=_("The city of the organization."),
        blank=True,
    )
    state = USStateField(
        _("State"),
        help_text=_("The state of the organization."),
        blank=True,
    )
    zip_code = USZipCodeField(
        _("zip code"),
        help_text=_("The zip code of the organization."),
        blank=True,
    )
    phone_number = PhoneNumberField(
        _("Phone Number"),
        help_text=_("The phone number of the organization."),
        blank=True,
        region="US",
    )
    website = models.URLField(
        _("Website"),
        blank=True,
        help_text=_("The website of the organization."),
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
    token_expiration_days = models.PositiveIntegerField(
        _("Token Expiration Days"),
        default=30,
        help_text=_("The number of days before a token expires."),
    )

    class Meta:
        """
        Metaclass for the Organization model
        """

        verbose_name = _("Organization")
        verbose_name_plural = _("Organizations")
        ordering = ["name"]
        db_table = "organization"

    def __str__(self) -> str:
        """
        Returns:
            str: String representation of the organization.
        """
        return textwrap.wrap(self.name, 50)[0]

    def save(self, **kwargs: Any) -> None:
        """
        Organization save method.  Ensures that the scac_code is always uppercase.

        Args:
            **kwargs (Any): Keyword arguments

        Returns:
            None: This function does not return anything.
        """

        self.scac_code = self.scac_code.upper()
        super().save(**kwargs)

    @cached_property
    def get_address_combination(self) -> str:
        """
        Returns:
            str: String representation of the organization address.
        """
        return f"{self.address_line_1} {self.address_line_2} {self.city} {self.state} {self.zip_code}"

    @cached_property
    def get_address(self) -> str:
        """
        Returns:
            str: String representation of the organization address.
        """
        return f"{self.address_line_1} {self.address_line_2}"

    @cached_property
    def get_city_state_zip(self) -> str:
        """
        Returns:
            str: String representation of the organization address.
        """
        return f"{self.city}, {self.state} {self.zip_code}"

    def get_absolute_url(self) -> str:
        """
        Returns:
            str: The absolute url for the organization.
        """
        return reverse("organizations-detail", kwargs={"pk": self.pk})


class Depot(TimeStampedModel):
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
        db_table = "depot"
        constraints = [
            models.UniqueConstraint(
                fields=["name", "organization"],
                name="unique_depot_name_organization",
            )
        ]

    def __str__(self) -> str:
        """Depot string representation.

        Returns:
            str: String representation of the depot.
        """
        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """Depot absolute URL

        Returns:
            str: The absolute url for the depot.
        """
        return reverse("organization-depot-detail", kwargs={"pk": self.pk})


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
        help_text=_("The state of the depot."),
    )
    zip_code = USZipCodeField(
        _("Zip Code"),
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
        db_table = "depot_detail"

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

        return reverse("organization-depot-detail", kwargs={"pk": self.depot.pk})


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
        db_table = "department"

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

        return reverse("organization-department-detail", kwargs={"pk": self.pk})


class EmailProfile(TimeStampedModel):
    """
    Stores the email control information for a related :model:`organization.Organization`
    """

    @final
    class EmailProtocolChoices(models.TextChoices):
        """
        Choices that will be used for Email Protocol
        """

        TLS = "TLS", _("TLS")
        SSL = "SSL", _("SSL")
        UNENCRYPTED = "UNENCRYPTED", _("Unencrypted SMTP")

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
        db_table = "email_profile"

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

        return reverse("email-profiles-detail", kwargs={"pk": self.pk})


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
    rate_expiration_email_profile = models.ForeignKey(
        EmailProfile,
        on_delete=models.SET_NULL,
        verbose_name=_("Rate Expiration Email Profile"),
        related_name="rate_expiration_email_control",
        help_text=_("The email profile that will be used for rate expiration emails."),
        null=True,
        blank=True,
    )

    class Meta:
        """
        Metaclass for the EmailControl model
        """

        verbose_name = _("Email Control")
        verbose_name_plural = _("Email Controls")
        db_table = "email_control"

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

        return reverse("email-control-detail", kwargs={"pk": self.pk})


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
        db_table = "email_log"

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


class TaxRate(TimeStampedModel):
    """
    Stores the tax rate information for a related :model:`organization.Organization`
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
        verbose_name=_("Organization"),
        related_name="tax_rates",
        help_text=_("The organization that the tax rate belongs to."),
    )
    name = models.CharField(
        _("Name"),
        max_length=255,
        help_text=_("The name of the tax rate."),
    )
    rate = models.DecimalField(
        _("Rate"),
        max_digits=5,
        decimal_places=2,
        help_text=_("The rate of the tax rate."),
    )

    class Meta:
        """
        Metaclass for the TaxRate model
        """

        verbose_name = _("Tax Rate")
        verbose_name_plural = _("Tax Rates")
        ordering = ["name"]
        db_table = "tax_rate"

    def __str__(self) -> str:
        """TaxRate string representation.

        Returns:
            str: String representation of the tax rate.
        """

        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """TaxRate absolute URL

        Returns:
            str: The absolute url for the tax rate.
        """

        return reverse("tax-rates-detail", kwargs={"pk": self.pk})


class TableChangeAlert(TimeStampedModel):
    """
    Stores the table changes alert information for a related :model:`organization.Organization`
    """

    @final
    class DatabaseActionChoices(models.TextChoices):
        """
        Database action choices
        """

        INSERT = "INSERT", _("Insert")
        UPDATE = "UPDATE", _("Update")
        BOTH = "BOTH", _("Both")

    ACTION_NAMES = {
        "INSERT": {
            "function": "notify_new",
            "trigger": "after_insert",
            "listener": "new_added",
        },
        "UPDATE": {
            "function": "notify_updated",
            "trigger": "after_update",
            "listener": "updated",
        },
        "BOTH": {
            "function": "notify_new_or_updated",
            "trigger": "after_insert_or_update",
            "listener": "new_or_updated",
        },
    }

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    organization = models.ForeignKey(
        Organization,
        on_delete=models.CASCADE,
        verbose_name=_("Organization"),
        related_name="table_change_alerts",
        help_text=_("The organization that the tax rate belongs to."),
    )
    is_active = models.BooleanField(
        _("Is Active"),
        default=True,
        help_text=_("Whether the table change alert is active."),
    )
    name = models.CharField(
        _("Name"),
        max_length=50,
        help_text=_("The name of the table change alert."),
    )
    database_action = models.CharField(
        _("Database Action"),
        max_length=50,
        help_text=_("The database action that the table change alert is for."),
        choices=DatabaseActionChoices.choices,
        default=DatabaseActionChoices.INSERT,
    )
    table = models.CharField(
        _("Table"),
        max_length=255,
        help_text=_("The table that the table change alert is for."),
        choices=TABLE_NAME_CHOICES,
    )
    description = models.TextField(
        _("Description"),
        blank=True,
        help_text=_("The description of the table change alert."),
    )
    email_profile = models.ForeignKey(
        EmailProfile,
        on_delete=models.CASCADE,
        verbose_name=_("Email Profile"),
        related_name="table_change_alerts",
        help_text=_("The email profile that the table change alert will use."),
        blank=True,
        null=True,
    )
    function_name = models.CharField(
        _("Function Name"),
        max_length=50,
        help_text=_("The function name that the table change alert will use."),
        blank=True,
    )
    trigger_name = models.CharField(
        _("Trigger Name"),
        max_length=50,
        help_text=_("The trigger name that the table change alert will use."),
        blank=True,
    )
    listener_name = models.CharField(
        _("Listener Name"),
        max_length=50,
        help_text=_("The listener name that the table change alert will use."),
        blank=True,
    )
    effective_date = models.DateField(
        _("Effective Date"),
        help_text=_("The effective date of the table change alert."),
        blank=True,
        null=True,
    )
    expiration_date = models.DateField(
        _("Expiration Date"),
        help_text=_("The expiration date of the table change alert."),
        blank=True,
        null=True,
    )

    class Meta:
        """
        Metaclass for the TableChangeAlert model
        """

        verbose_name = _("Table Change Alert")
        verbose_name_plural = _("Table Change Alerts")
        ordering = ("name",)
        db_table = "table_change_alert"
        constraints = [
            models.UniqueConstraint(
                fields=["name", "organization"],
                name="unique_name_organization_table_change_alert",
            )
        ]

    def __str__(self) -> str:
        """TableChangeAlert string representation.

        Returns:
            str: String representation of the table change alert.
        """

        return textwrap.wrap(self.name, 50)[0]

    def get_absolute_url(self) -> str:
        """TableChangeAlert absolute URL

        Returns:
            str: The absolute url for the table change alert.
        """

        return reverse("table-change-alerts-detail", kwargs={"pk": self.pk})


class NotificationType(TimeStampedModel):
    """
    Stores the notification type information for a related :model:`organization.Organization`
    """

    class NotificationChoices(models.TextChoices):
        """
        Notification types choices
        """

        RATE_EXPIRATION = "RATE_EXPIRATION", _("Rate Expiration Notification")

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    organization = models.ForeignKey(
        Organization,
        on_delete=models.CASCADE,
        verbose_name=_("Organization"),
        related_name="notification_types",
        help_text=_("The organization that the notification type belongs to."),
    )
    name = models.CharField(
        max_length=255,
        unique=True,
        help_text=_("The name of the notification type."),
        choices=NotificationChoices.choices,
    )
    description = models.TextField(
        blank=True,
        help_text=_("The description of the notification type."),
    )

    class Meta:
        """
        Metaclass for the NotificationType model
        """

        verbose_name = _("Notification Type")
        verbose_name_plural = _("Notification Types")
        ordering = ("name",)
        db_table = "notification_type"
        constraints = [
            models.UniqueConstraint(
                fields=["name", "organization"],
                name="unique_name_organization_notification_type",
            )
        ]

    def __str__(self) -> str:
        """NotificationType string representation.

        Returns:
            str: String representation of the notification type.
        """

        return textwrap.shorten(self.name, width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """NotificationType absolute URL

        Returns:
            str: The absolute url for the notification type.
        """

        return reverse("notification-types-detail", kwargs={"pk": self.pk})


class NotificationSetting(TimeStampedModel):
    """
    Stores notification settings related to a :model:`organization.NotificationType`.
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
        related_name="notification_setting",
        help_text=_("The organization that the notification setting belongs to."),
    )
    notification_type = models.ForeignKey(
        NotificationType,
        on_delete=models.CASCADE,
        verbose_name=_("Notification Type"),
        related_name="notification_settings",
        help_text=_("The notification type that the notification setting belongs to."),
    )
    send_notification = models.BooleanField(
        _("Send Notification"),
        default=True,
        help_text=_("Whether the notification setting will send notifications."),
    )
    email_recipients = models.TextField(
        _("Email Recipients"),
        help_text=_("The email recipients that the notification setting will use."),
        default="",
        blank=True,
    )
    email_profile = models.ForeignKey(
        EmailProfile,
        on_delete=models.CASCADE,
        verbose_name=_("Email Profile"),
        related_name="notification_settings",
        help_text=_("The email profile that the notification setting will use."),
        blank=True,
        null=True,
    )
    custom_subject = models.CharField(
        _("Custom Subject"),
        max_length=255,
        help_text=_("The custom subject that the notification setting will use."),
        blank=True,
        validators=[validate_format_string],
    )
    custom_content = models.TextField(
        _("Custom Content"),
        help_text=_("The custom content that the notification setting will use."),
        blank=True,
        validators=[validate_format_string],
    )

    class Meta:
        """
        Metaclass for the NotificationSetting model
        """

        verbose_name = _("Notification Setting")
        verbose_name_plural = _("Notification Settings")
        ordering = ("organization", "notification_type")
        db_table = "notification_setting"
        constraints = [
            models.UniqueConstraint(
                fields=["organization", "notification_type"],
                name="unique_organization_notification_type_notification_setting",
            )
        ]

    def __str__(self) -> str:
        """NotificationSetting string representation.

        Returns:
            str: String representation of the notification setting.
        """

        return textwrap.shorten(
            f"{self.organization} - {self.notification_type}",
            width=50,
            placeholder="...",
        )

    def get_email_recipients(self) -> list[str]:
        """Get the email recipients as a list of strings.

        Returns:
            list[str]: The email recipients as a list of strings.
        """
        return [
            email.strip() for email in self.email_recipients.split(",") if email.strip()
        ]

    def get_absolute_url(self) -> str:
        """NotificationSetting absolute URL

        Returns:
            str: The absolute url for the notification setting.
        """
        return reverse("notification-settings-detail", kwargs={"pk": self.pk})
