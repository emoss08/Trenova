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

import pytz
from django.db import models
from django.db.models.functions import Lower
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from organization.services.table_choices import TABLE_NAME_CHOICES
from utils.models import GenericModel, Weekdays


@final
class ScheduleType(models.TextChoices):
    """
    The schedule type for a scheduled report.
    """

    DAILY = "DAILY", _("Daily")
    WEEKLY = "WEEKLY", _("Weekly")
    MONTHLY = "MONTHLY", _("Monthly")


class Weekday(models.Model):
    """
    Stores the weekdays for a weekly scheduled report.

    Attributes:
        name (str): The name of the weekday.

    Methods:
        __str__: String representation of the weekday.
        get_absolute_url: Returns the absolute URL for the weekday.
    """

    name = models.PositiveIntegerField(_("Name"), choices=Weekdays.choices, unique=True)

    class Meta:
        """
        Metaclass for the Weekday model.

        Attributes:
            verbose_name (str): The verbose name for the model.
            verbose_name_plural (str): The plural verbose name for the model.
            ordering (list): The ordering of the model.
            db_table (str): The database table name for the model.
        """

        verbose_name = _("Weekday")
        verbose_name_plural = _("Weekdays")
        ordering = ("name",)
        db_table = "weekday"

    def __str__(self) -> str:
        """String representation of the weekday.

        Returns:
            str: The name of the weekday.
        """
        return textwrap.shorten(self.get_name_display(), width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """Returns the absolute URL for the weekday.

        Returns:
            str: The absolute URL for the weekday.
        """
        return reverse("weekday-detail", kwargs={"pk": self.pk})


class CustomReport(GenericModel):
    """
    Stores the custom reports information for related :model:`organization.Organization`.

    Attributes:
        id (UUID): The ID of the custom report.
        name (str): The name of the custom report.
        table (str): The table that the table change alert is for.

    Methods:
        __str__: String representation of the custom report.
        get_absolute_url: Returns the absolute URL for the custom report.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        verbose_name="ID",
    )
    name = models.CharField(
        _("Name"),
        max_length=255,
        unique=True,
    )
    table = models.CharField(
        _("Table"),
        max_length=255,
        help_text=_("The table that the table change alert is for."),
        choices=TABLE_NAME_CHOICES,
    )

    class Meta:
        """
        Metaclass for the CustomReport model.

        Attributes:
            verbose_name (str): The verbose name of the model.
            verbose_name_plural (str): The verbose plural name of the model.
            ordering (tuple): The ordering of the model.
            db_table (str): The database table name of the model.
            constraints (list): The constraints of the model.
        """

        verbose_name = _("Custom Report")
        verbose_name_plural = _("Custom Reports")
        ordering = ("name",)
        db_table = "custom_report"
        constraints = [
            models.UniqueConstraint(
                Lower("name"), "organization", name="unique_report_name_organization"
            )
        ]

    def __str__(self) -> str:
        """String representation of the custom report.

        Returns:
            str: The name of the custom report.
        """
        return textwrap.shorten(self.name, width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """Returns the absolute URL for the custom report.

        Returns:
            str: The absolute URL for the custom report.
        """
        return reverse("report-detail", kwargs={"pk": self.pk})


class ReportColumn(GenericModel):
    """
    Stores the columns for a related :model:`reports.CustomReport`.

    Attributes:
        id (UUID): The ID of the report column.
        custom_report (:model:`reports.CustomReport`): The report that the column is for.
        column_name (str): The name of the column to be displayed in the report.
        column_shipment (int): The shipment of the column to be displayed in the report.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        verbose_name="ID",
    )
    custom_report = models.ForeignKey(
        CustomReport,
        on_delete=models.CASCADE,
        related_name="columns",
        verbose_name=_("Report"),
    )
    column_name = models.CharField(
        _("Column Name"),
        max_length=255,
        help_text=_("The name of the column to be displayed in the report."),
    )
    column_shipment = models.PositiveIntegerField(
        _("Column shipment"),
        help_text=_("The shipment of the column to be displayed in the report."),
    )

    class Meta:
        """
        Metaclass for the ReportColumn model.

        Attributes:
            verbose_name (str): The verbose name of the model.
            verbose_name_plural (str): The verbose plural name of the model.
            ordering (tuple): The ordering of the model.
            db_table (str): The database table name of the model.
            constraints (list): The constraints of the model.
        """

        verbose_name = _("Report Column")
        verbose_name_plural = _("Report Columns")
        ordering = ("column_shipment",)
        db_table = "report_column"
        constraints = [
            models.UniqueConstraint(
                fields=["custom_report", "column_name", "column_shipment"],
                name="unique_report_column_name_shipment",
            )
        ]

    def __str__(self) -> str:
        """String representation of the report column.

        Returns:
            str: The name of the report column.
        """
        return textwrap.shorten(self.column_name, width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """Returns the absolute URL for the report column.

        Returns:
            str: The absolute URL for the report column.
        """
        return reverse("report-column-detail", kwargs={"pk": self.pk})


class ScheduledReport(GenericModel):
    """
    Stores the scheduled reports information for related :model:`organization.Organization`.

    Attributes:
        id (UUID): The ID of the scheduled report.
        is_active (bool): Whether the scheduled report is active.
        custom_report (:model:`reports.CustomReport`): The report that the scheduled report is for.
        user (:model:`accounts.User`): The user that the scheduled report is for.
        schedule_type (str): The type of schedule for the scheduled report.
        time (TimeField): The time of the scheduled report.
        day_of_week (str): The day of the week for the scheduled report.
        day_of_month (int): The day of the month for the scheduled report.
        timezone (str): The timezone for the scheduled report.

    Methods:
        __str__: String representation of the scheduled report.
        get_absolute_url: Returns the absolute URL for the scheduled report.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        help_text=_("Unique ID for the user."),
    )
    is_active = models.BooleanField(
        _("Active"),
        default=True,
        help_text=_("Whether the scheduled report is active."),
    )
    custom_report = models.ForeignKey(
        CustomReport,
        on_delete=models.CASCADE,
        related_name="scheduled_reports",
        verbose_name=_("Report"),
    )
    user = models.ForeignKey(
        "accounts.User",
        on_delete=models.CASCADE,
        related_name="scheduled_reports",
        verbose_name=_("User"),
    )
    schedule_type = models.CharField(
        _("Schedule Type"),
        max_length=255,
        choices=ScheduleType.choices,
        default=ScheduleType.DAILY,
    )
    time = models.TimeField(
        _("Time"),
        help_text=_("The time of day to send the report."),
    )
    day_of_week = models.ManyToManyField(
        Weekday,
        blank=True,
        help_text=_("The day of the week to send the report."),
        verbose_name=_("Day of Week"),
    )
    day_of_month = models.PositiveIntegerField(
        _("Day of Month"),
        help_text=_("The day of the month to send the report."),
        null=True,
        blank=True,
    )
    timezone = models.CharField(
        _("Timezone"),
        max_length=62,
        choices=[(tz, tz) for tz in pytz.all_timezones],
        default="UTC",
    )

    class Meta:
        """
        Metaclass for the ScheduledReport model.

        Attributes:
            verbose_name (str): The verbose name of the model.
            verbose_name_plural (str): The verbose plural name of the model.
            ordering (tuple): The ordering of the model.
            db_table (str): The database table name of the model.
            constraints (list): The constraints of the model.
        """

        verbose_name = _("Scheduled Report")
        verbose_name_plural = _("Scheduled Reports")
        ordering = ("custom_report",)
        db_table = "scheduled_report"
        constraints = [
            models.UniqueConstraint(
                fields=["custom_report", "organization"],
                name="unique_scheduled_report_report_organization",
            )
        ]

    def __str__(self) -> str:
        """String representation of the scheduled report.

        Returns:
            str: The name of the scheduled report.
        """
        return textwrap.shorten(self.custom_report.name, width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """Returns the absolute URL for the scheduled report.

        Returns:
            str: The absolute URL for the scheduled report.
        """
        return reverse("scheduled-report-detail", kwargs={"pk": self.pk})


class UserReport(GenericModel):
    """
    Stores the user reports information for related :model:`accounts.User`.
    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
    )
    user = models.ForeignKey(
        "accounts.User",
        on_delete=models.CASCADE,
        related_name="user_reports",
        help_text=_("The user that the report belongs to"),
        verbose_name=_("User"),
    )
    report = models.FileField(
        _("Report"),
        upload_to="reports/user/",
        help_text=_("The report file"),
    )

    class Meta:
        """
        Metaclass for the UserReport model
        """

        verbose_name = _("User Report")
        verbose_name_plural = _("User Reports")
        db_table = "user_report"
        ordering = ("-created",)

    def __str__(self) -> str:
        """UserReport string representation.

        Returns:
            str: String representation of the UserReport Model.
        """
        return textwrap.shorten(
            f"{self.user.username} ({self.report.name})", width=30, placeholder="..."
        )

    def get_absolute_url(self) -> str:
        """Absolute URL for the UserReport.

        Returns:
            str: Get the absolute url of the user report
        """
        return reverse("user-report-detail", kwargs={"pk": self.pk})
