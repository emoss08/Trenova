# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
import io
import logging
import typing
from io import BytesIO

import pandas as pd
from asgiref.sync import async_to_sync
from channels.layers import get_channel_layer
from django.apps import apps
from django.core.mail import EmailMessage
from django.db.models import Model
from django.template.loader import render_to_string
from django.utils import timezone
from notifications.signals import notify
from weasyprint import CSS, HTML

from accounts.models import User
from reports import constants, exceptions, models

if typing.TYPE_CHECKING:
    from utils.types import ModelUUID

channel_layer = get_channel_layer()

logger = logging.getLogger(__name__)


def deliver_local_report(
    *, user: User, model_name: str, report_obj: models.UserReport
) -> None:
    """
    This function will deliver the local report to the user.
    """
    try:
        notify.send(
            user,
            recipient=user,
            verb="New Report is available",
            description=f"New {model_name} report is available for download.",
            public=False,
            action_object=report_obj,
        )

        async_to_sync(channel_layer.group_send)(
            user.username,
            {
                "type": "send_notification",
                "recipient": user.username,
                "attr": "report",
                "event": "New Report is available",
                "description": f"New {model_name} report is available for download.",
            },
        )
    except exceptions.LocalDeliveryException as l_exception:
        logger.error(f"Local report delivery did not succeed: {l_exception}")


def delivery_email_report(
    *,
    user: User,
    model_name: str,
    email_recipients: list[str] | None,
    file_name: str,
    buffer: io.BytesIO,
) -> None:
    """
    This function will deliver the report via email to the user.
    """

    if not email_recipients:
        logger.error("Email delivery found: No email recipients found.")
        return

    try:
        email = EmailMessage(
            subject=f"New {model_name} report is available for download.",
            body=f"New {model_name} report is available for download.",
            from_email="noreply@trenova.app",
            to=email_recipients,
            attachments=[(file_name, buffer.getvalue(), "application/octet-stream")],
        )
        email.send()
        async_to_sync(get_channel_layer().group_send)(
            user.username,
            {
                "type": "send_notification",
                "recipient": user.username,
                "attr": "report",
                "event": "New Report is available",
                "description": f"New {model_name} report is has been sent to your email.",
            },
        )
    except exceptions.EmailDeliveryException as e_exception:
        logger.error(
            f"Failed to deliver email report to {', '.join(email_recipients)}: {e_exception}"
        )


def generate_pdf(*, df: pd.DataFrame, buffer: BytesIO, user: User) -> None:
    """Generate a PDF file from a DataFrame.

    This function will generate a PDF file from a DataFrame and save it to the provided buffer.

    Args:
        df (pd.DataFrame): The DataFrame to generate the PDF from.
        buffer (BytesIO): The buffer to save the PDF file to.
        user (User): The user object, required for context matching.

    Returns:
        None: This function does not return anything.
    """
    # Prepare the data for context matching.
    context = {
        "data": df.to_dict(orient="records"),
        "organization": user.organization,
        "user": user,
        "report_generation_date": timezone.now(),
    }

    # Render the HTML template with the context.
    html_string = render_to_string("reports/pdf_template.html", context)

    # Define CSS for landscape orientation
    landscape_css = CSS(
        string="@page { size: landscape; } @media print { body { -webkit-print-color-adjust: exact; } }"
    )

    # Convert the html string to a PDF in landscape format and save it to the buffer.
    HTML(string=html_string).write_pdf(target=buffer, stylesheets=[landscape_css])


def validate_model(model_name: str) -> None:
    """Validate if the provided model name is allowed.

    Args:
        model_name (str): The name of the model to validate.

    Returns:
        None: this function does not return anything.

    Raises:
        InvalidModelException: If the model name is not in the allowed models list.
    """
    if model_name not in constants.ALLOWED_MODELS:
        raise exceptions.InvalidModelException(f"Model {model_name} is not allowed.")


def get_user_and_model(
    user_id: "ModelUUID", model_name: str
) -> tuple[User, type[Model]]:
    """Retrieve the user object and model class based on user ID and model name.

    Args:
        user_id (ModelUUID): The unique identifier of the user.
        model_name (str): The name of the model to retrieve.

    Returns:
        tuple: A tuple containing the user object and the model class.

    Raises:
        InvalidModelException: If the model name is invalid.
    """
    user = User.objects.get(pk__exact=user_id)
    model = apps.get_model(
        constants.ALLOWED_MODELS[model_name]["app_label"], model_name
    )
    if not model:
        raise exceptions.InvalidModelException("Invalid model name")
    return user, model


def prepare_dataframe(
    model: type[Model], columns: list[str], user: User, model_name: str
) -> pd.DataFrame:
    """Prepare a pandas DataFrame based on the model, columns, and user information.

    Args:
        model (type[Model]): The model class from which to generate the DataFrame.
        columns (list[str]): A list of column names to include in the DataFrame.
        user (User): The user object to filter the data based on the organization.
        model_name (str): The name of the model, used for column renaming.

    Returns:
        pd.DataFrame: The prepared DataFrame.
    """
    queryset = (
        model.objects.filter(organization_id=user.organization_id)
        .select_related(*get_related_fields(columns))
        .values(*columns)
    )
    df = pd.DataFrame.from_records(queryset)
    convert_datetime_columns(df)
    rename_columns(df, model_name)
    return df


def get_related_fields(columns: list[str]) -> list[str]:
    """Extract related field names from a list of columns.

    Args:
        columns (list[str]): A list of column names, possibly including related field references.

    Returns:
        list[str]: A list of the base names of related fields.
    """
    return [field.split("__")[0] for field in columns if "__" in field]


def convert_datetime_columns(df: pd.DataFrame) -> None:
    """Convert timezone-aware datetime columns in a DataFrame to naive datetimes.

    Args:
        df (pd.DataFrame): The DataFrame whose datetime columns are to be converted.

    Returns:
        None: this function does not return anything.
    """
    for column in df.columns:
        if isinstance(df[column].dtype, pd.DatetimeTZDtype):
            df[column] = df[column].dt.tz_convert(None)


def rename_columns(df: pd.DataFrame, model_name: str) -> None:
    """Rename columns in a DataFrame based on allowed fields for a model.

    Args:
        df (pd.DataFrame): The DataFrame whose columns are to be renamed.
        model_name (str): The model name used to determine the new column names.

    Returns:
        None: this function does not return anything.
    """
    allowed_fields_dict = {
        field["value"]: field["label"]
        for field in constants.ALLOWED_MODELS[model_name]["allowed_fields"]
    }
    df.rename(columns=allowed_fields_dict, inplace=True)


def generate_csv(*, df: pd.DataFrame, report_buffer: BytesIO) -> None:
    """Generate a CSV file from a DataFrame.

    Args:
        df (pd.DataFrame): The DataFrame to convert to CSV.
        report_buffer (BytesIO): The buffer to write the CSV file to.

    Returns:
        None: this function does not return anything.
    """
    df.to_csv(report_buffer, index=False)


def generate_excel(*, df: pd.DataFrame, report_buffer: BytesIO) -> None:
    """Generate an Excel file from a DataFrame.

    Args:
        df (pd.DataFrame): The DataFrame to convert to Excel.
        report_buffer (BytesIO): The buffer to write the Excel file to.

    Returns:
        None: this function does not return anything.
    """
    # TODO(Wolfred): Add support for multiple sheets.
    df.to_excel(report_buffer, index=False)


def generate_report_file(
    df: pd.DataFrame, model_name: str, file_format: str, user: User
) -> tuple[BytesIO, str]:
    """Generate a report file in the specified format from a DataFrame.

    Args:
        df (pd.DataFrame): The DataFrame to generate the report from.
        model_name (str): The name of the model, used for naming the report file.
        file_format (str): The format of the report file (csv, xlsx, pdf).
        user (User): The user object, required for some report formats.

    Returns:
        tuple: A tuple containing the BytesIO buffer of the report and the report file name.

    Raises:
        ValueError: If the specified file format is not supported.
    """
    report_buffer = BytesIO()
    file_name = f"{model_name}-report.{file_format}"

    format_functions = {
        "csv": lambda: generate_csv(df=df, report_buffer=report_buffer),
        "xlsx": lambda: generate_excel(df=df, report_buffer=report_buffer),
        "pdf": lambda: generate_pdf(df=df, buffer=report_buffer, user=user),
    }

    generate_func = format_functions.get(file_format.lower())
    if not generate_func:
        raise ValueError("Invalid file format")

    generate_func()

    return report_buffer, file_name
