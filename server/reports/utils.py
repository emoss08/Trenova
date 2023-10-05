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
from asgiref.sync import async_to_sync
from channels.layers import get_channel_layer
from django.apps import apps
from django.core.files import File
from django.db.models import Model
from django.shortcuts import get_object_or_404
from django_celery_beat.models import CrontabSchedule
from notifications.signals import notify
from reportlab.lib import colors
from reportlab.lib.styles import getSampleStyleSheet
from reportlab.lib.units import inch
from reportlab.platypus import Image, SimpleDocTemplate, Spacer, Table, TableStyle
from reportlab.platypus.para import Paragraph

from accounts.models import User
from organization.models import Organization
from reports import exceptions, models
from reports.helpers import ALLOWED_MODELS
from utils.types import ModelUUID

channel_layer = get_channel_layer()


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

    schedule, _ = CrontabSchedule.objects.get_or_create(**schedule_filters)
    return schedule, "crontab"


def generate_pdf(
    *, df: pd.DataFrame, buffer: BytesIO, organization_id: ModelUUID
) -> None:
    """Generates a PDF file from a pandas DataFrame and writes it to the provided buffer.

    The DataFrame is converted into a ReportLab Table which is then included in a PDF
    document. The resulting PDF also contains the logo and name of the specified
    organization at the top. The size of the PDF is designed to be wide to accommodate
    the full width of the DataFrame.

    Args:
        df (pd.DataFrame): The DataFrame to be converted to PDF.
        buffer (BytesIO): The buffer to which the PDF is written.
        organization_id (str): The organization ID of the user who is generating the PDF.

    Returns:
        None: This function does not return anything.
    """

    # Define style elements
    styles = getSampleStyleSheet()
    normal_style = styles["BodyText"]
    normal_style.fontName = "Helvetica"
    normal_style.fontSize = 10

    # Transform dataframe to a list of lists (records) which ReportLab can work with
    data = [df.columns.to_list()] + df.values.tolist()

    # Set custom page size
    page_width = 2000  # you can adjust this as needed
    page_height = 1190  # keep the height same as A3 height
    pdf = SimpleDocTemplate(buffer, pagesize=(page_width, page_height))

    elements = []

    # Add organization logo to top left corner of the PDF, if it exists. Otherwise, add organization name same place,
    # in black text.
    organization = get_object_or_404(Organization, id=organization_id)

    if organization.logo:
        logo = BytesIO(organization.logo.read())
        logo.seek(0)
        elements.append(Image(logo, width=50, height=50))
    elements.extend((Paragraph(organization.name, normal_style), Spacer(1, 0.2 * inch)))
    # Create a table where the first row is the header
    num_columns = len(data[0])
    column_width = (
        page_width - 2 * inch
    ) / num_columns  # Assume half-inch margins on the left and right
    table = Table(data, repeatRows=1, colWidths=[column_width] * num_columns)

    # Define table style
    table_style = TableStyle(
        [
            ("BACKGROUND", (0, 0), (-1, 0), colors.grey),
            ("TEXTCOLOR", (0, 0), (-1, 0), colors.whitesmoke),
            ("ALIGN", (0, 0), (-1, -1), "CENTER"),
            ("FONTNAME", (0, 0), (-1, 0), "Helvetica-Bold"),
            ("FONTSIZE", (0, 0), (-1, 0), 10),
            ("BOTTOMPADDING", (0, 0), (-1, 0), 12),
            ("BACKGROUND", (0, 1), (-1, -1), colors.white),
            ("GRID", (0, 0), (-1, -1), 1, colors.black),
        ]
    )

    # Apply the table styles
    table.setStyle(table_style)

    elements.append(table)

    # Build the PDF
    pdf.build(elements)


def generate_report(
    *, model_name: str, columns: list[str], user_id: ModelUUID, file_format: str
) -> None:
    """Generate a report in the specified format for a given user based on the specified model.

    This function accepts a model name, a list of columns, a user ID and a file format,
    and generates a report accordingly. It first checks if the given model name is
    allowed. Then it retrieves the user with the given ID and the associated model.
    After this, it creates a queryset filtering the model objects based on the organization
    id of the user. It also converts timezone aware datetime columns to naive datetime.

    The function generates a report in the specified format (CSV, XLSX, or PDF) and stores
    it in a buffer. If the specified format is not one of the allowed formats, it raises a ValueError.

    The report is then saved to the UserReport model and the instance is returned.

    Args:
        model_name (str): The name of the model on which the report is to be based.
        columns (list[str]): The list of columns to be included in the report.
        user_id (ModelUUID): The ID of the user for whom the report is being generated.
        file_format (str): The format in which the report should be generated (csv, xlsx, pdf).

    Raises:
        ValueError: If the provided file format is not among the allowed formats (csv, xlsx, pdf).
    """

    # Check if the model name is allowed
    allowed_model = ALLOWED_MODELS[model_name]

    # Get the user and the model
    user = User.objects.get(pk__exact=user_id)
    model: type[Model] = apps.get_model(allowed_model["app_label"], model_name)  # type: ignore

    if not model:
        raise exceptions.InvalidModelException("Invalid model name")

    # Get the related fields
    related_fields = [field.split("__")[0] for field in columns if "__" in field]

    # Create the queryset
    queryset = (
        model.objects.filter(organization_id=user.organization_id)
        .select_related(*related_fields)
        .values(*columns)
    )

    # Convert timezone aware datetime columns to naive datetime
    df = pd.DataFrame.from_records(queryset)
    for column in df.columns:
        if pd.api.types.is_datetime64tz_dtype(df[column]):
            df[column] = df[column].dt.tz_convert(None)

    # Extract 'value' and 'label' from each dictionary in the 'allowed_fields' list
    allowed_fields_dict = {
        field["value"]: field["label"] for field in allowed_model["allowed_fields"]  # type: ignore
    }

    # Rename the columns
    df.rename(columns=allowed_fields_dict, inplace=True)

    buffer = BytesIO()

    # Generate the report in the specified format
    if file_format.lower() == "csv":
        df.to_csv(buffer, index=False)
        file_name = f"{model_name}-report.csv"
    elif file_format.lower() == "xlsx":
        df.to_excel(buffer, index=False)
        file_name = f"{model_name}-report.xlsx"
    elif file_format.lower() == "pdf":
        generate_pdf(df=df, buffer=buffer, organization_id=user.organization_id)
        file_name = f"{model_name}-report.pdf"
    else:
        raise ValueError("Invalid file format")

    buffer.seek(0)
    django_file = File(buffer, name=file_name)

    # Save the report to the UserReport model
    new_report = models.UserReport.objects.create(
        organization=user.organization,
        user=user,
        report=django_file,
        business_unit=user.business_unit,
    )

    # Send notification to the user
    notify.send(
        user,
        recipient=user,
        verb="New Report is available",
        description=f"New {model_name} report is available for download.",
        public=False,
        action_object=new_report,
    )

    # Send Websocket message to the user
    async_to_sync(get_channel_layer().group_send)(
        user.username,
        {
            "type": "send_notification",
            "recipient": user.username,
            "attr": "report",
            "event": "New Report is available",
            "description": f"New {model_name} report is available for download.",
        },
    )
