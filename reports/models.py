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
from django.db import models
from django.urls import reverse
from django.utils.translation import gettext_lazy as _

from organization.services.table_choices import TABLE_NAME_CHOICES
from utils.models import GenericModel


class ScheduleType(models.TextChoices):
    DAILY = "DAILY", _("Daily")
    WEEKLY = "WEEKLY", _("Weekly")
    MONTHLY = "MONTHLY", _("Monthly")


class CustomReport(GenericModel):
    """
    A Django model representing a custom report.

    Attributes:
        id (models.UUIDField): The primary key field for the report ID.
        name (models.CharField): The name of the report.
        table (models.CharField): The name of the table that the report is based on.

    Meta:
        verbose_name (str): The human-readable name of the model.

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

    def __str__(self) -> str:
        """Report string representation.

        Returns:
            str: String representation of the report model.
        """
        return textwrap.shorten(self.name, width=50, placeholder="...")

    class Meta:
        """
        Metaclass for the Report model.
        """

        verbose_name = _("Custom Report")
        verbose_name_plural = _("Custom Reports")
        ordering = ("name",)
        db_table = "custom_report"
        constraints = [
            models.UniqueConstraint(
                fields=["name", "organization"], name="unique_report_name_organization"
            )
        ]

    def get_absolute_url(self) -> str:
        """Report absolute URL

        Returns:
            str: The absolute URL for the report.
        """
        return reverse("report-detail", kwargs={"pk": self.pk})


class ReportColumn(GenericModel):
    """
    A Django model representing a column in a custom report.

    Attributes:
        id (models.UUIDField): The primary key field for the report column ID.
        report (models.ForeignKey): The foreign key field linking the column to a custom report.
        column_name (models.CharField): The name of the column to be displayed in the report.
        column_order (models.PositiveIntegerField): The order of the column to be displayed in the report.

    Meta:
        ordering (list): The default ordering for query sets.

    """

    id = models.UUIDField(
        primary_key=True,
        default=uuid.uuid4,
        editable=False,
        unique=True,
        verbose_name="ID",
    )
    report = models.ForeignKey(
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
    column_order = models.PositiveIntegerField(
        _("Column Order"),
        help_text=_("The order of the column to be displayed in the report."),
    )

    def __str__(self) -> str:
        """Report column string representation.

        Returns:
            str: String representation of the report column model.
        """
        return textwrap.shorten(self.column_name, width=50, placeholder="...")

    class Meta:
        """
        Metaclass for the ReportColumn model.
        """

        verbose_name = _("Report Column")
        verbose_name_plural = _("Report Columns")
        ordering = ("column_order",)
        db_table = "report_column"
        constraints = [
            models.UniqueConstraint(
                fields=["report", "column_name", "column_order"],
                name="unique_report_column_name_order",
            )
        ]

    def get_absolute_url(self) -> str:
        """Report column absolute URL

        Returns:
            str: The absolute URL for the report column.
        """
        return reverse("report-column-detail", kwargs={"pk": self.pk})


class ScheduledReport(GenericModel):
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
    report = models.ForeignKey(
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
    day_of_week = models.PositiveIntegerField(
        _("Day of Week"),
        help_text=_("The day of the week to send the report."),
        null=True,
        blank=True,
    )
    day_of_month = models.PositiveIntegerField(
        _("Day of Month"),
        help_text=_("The day of the month to send the report."),
        null=True,
        blank=True,
    )

    class Meta:
        """
        Metaclass for the ScheduledReport model.
        """

        verbose_name = _("Scheduled Report")
        verbose_name_plural = _("Scheduled Reports")
        ordering = ("report",)
        db_table = "scheduled_report"
        constraints = [
            models.UniqueConstraint(
                fields=["report", "organization"],
                name="unique_scheduled_report_report_organization",
            )
        ]

    def __str__(self) -> str:
        """Scheduled report string representation.

        Returns:
            str: String representation of the scheduled report model.
        """
        return textwrap.shorten(self.report.name, width=50, placeholder="...")

    def get_absolute_url(self) -> str:
        """Scheduled report absolute URL

        Returns:
            str: The absolute URL for the scheduled report.
        """
        return reverse("scheduled-report-detail", kwargs={"pk": self.pk})
