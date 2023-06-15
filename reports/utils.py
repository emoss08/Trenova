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
from io import BytesIO

import pandas as pd
from django.apps import apps
from django.core.files import File
from django.db.models import Model
from django.shortcuts import get_object_or_404
from django_celery_beat.models import CrontabSchedule
from reportlab.lib.pagesizes import landscape, letter
from reportlab.platypus import SimpleDocTemplate, Table

from accounts.models import User
from reports import exceptions, models
from reports.helpers import ALLOWED_MODELS
from utils.types import ModelUUID


def get_crontab_schedule(
    *, schedule_type: str, instance: models.ScheduledReport
) -> tuple[CrontabSchedule, str]:
    """Get or create a CrontabSchedule object based on the schedule type and scheduled report instance.

    Args:
        schedule_type (models.ScheduleType): The schedule type (DAILY, WEEKLY, MONTHLY) of the scheduled report.
        instance (models.ScheduledReport): The scheduled report instance.

    Returns:
        Tuple[CrontabSchedule, str]: A tuple containing the CrontabSchedule object and the task type ("crontab").

    Raises:
        exceptions.InvalidScheduleTypeException: If the schedule type is not valid.
    """
    if schedule_type == models.ScheduleType.DAILY:
        schedule_filters = {
            "hour": instance.time.hour,
            "minute": instance.time.minute,
            "timezone": instance.timezone,
        }
    elif schedule_type == models.ScheduleType.WEEKLY:
        weekdays = ",".join([str(weekday.id) for weekday in instance.day_of_week.all()])
        schedule_filters = {
            "day_of_week": weekdays,
            "hour": instance.time.hour,
            "minute": instance.time.minute,
            "timezone": instance.timezone,
        }
    elif schedule_type == models.ScheduleType.MONTHLY:
        schedule_filters = {
            "day_of_month": instance.day_of_month,
            "hour": instance.time.hour,
            "minute": instance.time.minute,
            "timezone": instance.timezone,
        }
    else:
        raise exceptions.InvalidScheduleTypeException("Invalid schedule type.")

    schedule, created = CrontabSchedule.objects.get_or_create(**schedule_filters)
    return schedule, "crontab"


def generate_pdf(*, df: pd.DataFrame, buffer: BytesIO) -> None:
    # Transform dataframe to a list of lists (records) which ReportLab can work with
    data = [df.columns.to_list()] + df.values.tolist()

    # Create a PDF with ReportLab
    pdf = SimpleDocTemplate(buffer, pagesize=landscape(letter))
    table = Table(data)

    # Add table to the elements to be added to the PDF
    elements = [table]

    # Build the PDF
    pdf.build(elements)


def generate_report(
    *, model_name: str, columns: list[str], user_id: ModelUUID, file_format: str
) -> str:
    allowed_model = ALLOWED_MODELS[model_name]

    user = get_object_or_404(User, id=user_id)
    model = apps.get_model(allowed_model["app_label"], model_name)

    related_fields = [field.split("__")[0] for field in columns if "__" in field]

    queryset = model.objects.select_related(*related_fields).values(*columns)  # type: ignore
    df = pd.DataFrame.from_records(queryset)

    for column in df.columns:
        if pd.api.types.is_datetime64tz_dtype(df[column]):
            df[column] = df[column].dt.tz_convert(None)

    buffer = BytesIO()

    if file_format.lower() == "csv":
        df.to_csv(buffer, index=False)
        file_name = "report.csv"
    elif file_format.lower() == "xlsx":
        df.to_excel(buffer, index=False)
        file_name = "report.xlsx"
    elif file_format.lower() == "pdf":
        generate_pdf(df=df, buffer=buffer)
        file_name = "report.pdf"
    else:
        raise ValueError("Invalid file format")

    buffer.seek(0)
    django_file = File(buffer, name=file_name)

    return models.UserReport.objects.create(
        organization=user.organization, user=user, report=django_file
    )
