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
from io import BytesIO

import pandas as pd
from asgiref.sync import async_to_sync
from channels.layers import get_channel_layer
from django.apps import apps
from django.core.mail import EmailMessage
from django.db.models import Model
from django.shortcuts import get_object_or_404
from notifications.signals import notify
from reportlab.lib import colors
from reportlab.lib.styles import getSampleStyleSheet
from reportlab.lib.units import inch
from reportlab.platypus import Image, SimpleDocTemplate, Spacer, Table, TableStyle
from reportlab.platypus.para import Paragraph

from accounts.models import User
from organization.models import Organization
from reports import exceptions, models
from utils.types import ModelUUID

channel_layer = get_channel_layer()

logger = logging.getLogger(__name__)


# Allowed models for reports
ALLOWED_MODELS = {
    "User": {
        "app_label": "accounts",
        "allowed_fields": [
            {"value": "username", "label": "Username"},
            {"value": "email", "label": "Email"},
            {"value": "date_joined", "label": "Date Joined"},
            {"value": "is_staff", "label": "Is Staff"},
            {"value": "profiles__first_name", "label": "First Name"},
            {"value": "profiles__last_name", "label": "Last Name"},
            {"value": "profiles__address_line_1", "label": "Address Line 1"},
            {"value": "profiles__address_line_2", "label": "Address Line 2"},
            {"value": "profiles__city", "label": "City"},
            {"value": "profiles__state", "label": "State"},
            {"value": "profiles__zip_code", "label": "Zip Code"},
            {"value": "profiles__phone_number", "label": "Phone Number"},
            {
                "value": "profiles__is_phone_verified",
                "label": "Is Phone Verified",
            },
            {"value": "profiles__job_title__name", "label": "Job Title Name"},
            {
                "value": "profiles__job_title__description",
                "label": "Job Title Description",
            },
            {"value": "department__name", "label": "Department Name"},
            {
                "value": "department__description",
                "label": "Department Description",
            },
            {"value": "organization__name", "label": "Organization Name"},
        ],
    },
    "UserProfile": {
        "app_label": "accounts",
        "allowed_fields": [
            {"value": "user__username", "label": "Username"},
            {"value": "user__email", "label": "Email"},
            {"value": "user__date_joined", "label": "Date Joined"},
            {"value": "user__is_staff", "label": "Is Staff"},
            {"value": "first_name", "label": "First Name"},
            {"value": "last_name", "label": "Last Name"},
            {"value": "address_line_1", "label": "Address Line 1"},
            {"value": "address_line_2", "label": "Address Line 2"},
            {"value": "city", "label": "City"},
            {"value": "state", "label": "State"},
            {"value": "zip_code", "label": "Zip Code"},
            {"value": "phone_number", "label": "Phone Number"},
            {
                "value": "is_phone_verified",
                "label": "Is Phone Verified",
            },
        ],
    },
    "JobTitle": {
        "app_label": "accounts",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "job_function", "label": "Job Function"},
        ],
    },
    "Organization": {
        "app_label": "organization",
        "allowed_fields": [
            {"value": "name", "label": "Name"},
            {"value": "scac_code", "label": "SCAC Code"},
            {"value": "dot_number", "label": "DOT Number"},
            {"value": "address_line_1", "label": "Address Line 1"},
            {"value": "address_line_2", "label": "Address Line 2"},
            {"value": "city", "label": "City"},
            {"value": "state", "label": "State"},
            {"value": "zip_code", "label": "Zip Code"},
            {"value": "phone_number", "label": "Phone Number"},
            {"value": "website", "label": "Website"},
            {"value": "org_type", "label": "Organization Type"},
            {"value": "timezone", "label": "Timezone"},
            {"value": "language", "label": "Language"},
            {"value": "currency", "label": "Currency"},
            {"value": "date_format", "label": "Date Format"},
            {"value": "time_format", "label": "Time Format"},
            {"value": "token_expiration_days", "label": "Token Expiration Days"},
        ],
    },
    "Department": {
        "app_label": "organization",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
        ],
    },
    "DivisionCode": {
        "app_label": "accounting",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "code", "label": "Code"},
            {"value": "description", "label": "Description"},
            {"value": "cash_account__account_number", "label": "Cash Account"},
            {"value": "ap_account__account_number", "label": "AP Account"},
            {"value": "expense_account__account_number", "label": "Expense Account"},
        ],
    },
    "RevenueCode": {
        "app_label": "accounting",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "code", "label": "Code"},
            {"value": "description", "label": "Description"},
            {"value": "expense_account__account_number", "label": "Expense Account"},
            {"value": "revenue_account__account_number", "label": "Revenue Account"},
        ],
    },
    "GeneralLedgerAccount": {
        "app_label": "accounting",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "account_number", "label": "Account Number"},
            {"value": "account_type", "label": "Account Type"},
            {"value": "cash_flow_type", "label": "Cash Flow Type"},
            {"value": "account_sub_type", "label": "Account Sub Type"},
            {"value": "account_classification", "label": "Account Classification"},
            {"value": "balance", "label": "Balance"},
            {"value": "opening_balance", "label": "Opening Balance"},
            {"value": "closing_balance", "label": "Closing Balance"},
            {
                "value": "parent_account__account_number",
                "label": "Parent Account Number",
            },
            {"value": "is_reconciled", "label": "Is Reconciled"},
            {"value": "date_opened", "label": "Date Opened"},
            {"value": "date_closed", "label": "Date Closed"},
            {"value": "notes", "label": "Notes"},
            {"value": "owner__username", "label": "Owner"},
            {"value": "is_tax_relevant", "label": "Is Tax Relevant"},
            {"value": "interest_rate", "label": "Interest Rate"},
        ],
    },
    "ChargeType": {
        "app_label": "billing",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
        ],
    },
    "AccessorialCharge": {
        "app_label": "billing",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "code", "label": "Code"},
            {"value": "description", "label": "Description"},
            {"value": "is_detention", "label": "Is Detention"},
            {"value": "charge_amount", "label": "Charge Amount"},
            {"value": "method", "label": "Method"},
        ],
    },
    "HazardousMaterial": {
        "app_label": "commodities",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "hazard_class", "label": "Hazard Class"},
            {"value": "packing_group", "label": "Packing Group"},
            {"value": "erg_number", "label": "ERG Number"},
            {"value": "proper_shipping_name", "label": "Proper Shipping Name"},
        ],
    },
    "Commodity": {
        "app_label": "commodities",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "min_temp", "label": "Minimum Temperature"},
            {"value": "max_temp", "label": "Maximum Temperature"},
            {"value": "set_point_temp", "label": "Set Point Temperature"},
            {"value": "unit_of_measure", "label": "Unit of Measure"},
            {
                "value": "hazardous_material__status",
                "label": "Hazardous Material Status",
            },
            {"value": "hazardous_material__name", "label": "Hazardous Material Name"},
            {
                "value": "hazardous_material__description",
                "label": "Hazardous Material Description",
            },
            {
                "value": "hazardous_material__hazard_class",
                "label": "Hazardous Material Hazard Class",
            },
            {
                "value": "hazardous_material__packing_group",
                "label": "Hazardous Material Packing Group",
            },
            {
                "value": "hazardous_material__erg_number",
                "label": "Hazardous Material ERG Number",
            },
            {
                "value": "hazardous_material__proper_shipping_name",
                "label": "Hazardous Material Proper Shipping Name",
            },
            {"value": "is_hazmat", "label": "Is Hazardous Material"},
        ],
    },
    "Customer": {
        "app_label": "customer",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "code", "label": "Code"},
            {"value": "name", "label": "Name"},
            {"value": "address_line_1", "label": "Address Line 1"},
            {"value": "address_line_2", "label": "Address Line 2"},
            {"value": "city", "label": "City"},
            {"value": "zip_code", "label": "Zip Code"},
            {"value": "has_customer_portal", "label": "Has Customer Portal"},
            {"value": "auto_mark_ready_to_bill", "label": "Auto Mark Ready To Bill"},
        ],
    },
    "DelayCode": {
        "app_label": "dispatch",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "code", "label": "Code"},
            {"value": "description", "label": "Description"},
            {"value": "f_carrier_or_driver", "label": "F Carrier Or Driver"},
        ],
    },
    "FleetCode": {
        "app_label": "dispatch",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "code", "label": "Code"},
            {"value": "description", "label": "Description"},
            {"value": "revenue_goal", "label": "Revenue Goal"},
            {"value": "deadhead_goal", "label": "Deadhead Goal"},
            {"value": "mileage_goal", "label": "Mileage Goal"},
            {"value": "manager__username", "label": "Manager Username"},
        ],
    },
    "CommentType": {
        "app_label": "dispatch",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "created", "label": "Created At"},
            {"value": "modified", "label": "Modified At"},
        ],
    },
    "Rate": {
        "app_label": "dispatch",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "rate_number", "label": "Rate Number"},
            {"value": "customer__name", "label": "Customer Name"},
            {"value": "customer__code", "label": "Customer Code"},
            {"value": "effective_date", "label": "Effective Date"},
            {"value": "expiration_date", "label": "Expiration Date"},
            {"value": "commodity__name", "label": "Commodity Name"},
            {"value": "commodity__description", "label": "Commodity Description"},
            {"value": "shipment_type__name", "label": "shipment type Name"},
            {"value": "equipment_type__name", "label": "Equipment Type Name"},
            {"value": "origin_location__code", "label": "Origin Location Code"},
            {
                "value": "destination_location__code",
                "label": "Destination Location Code",
            },
            {"value": "rate_method", "label": "Rate Method"},
            {"value": "rate_amount", "label": "Rate Amount"},
            {"value": "distance_override", "label": "Distance Override"},
            {"value": "comments", "label": "Comments"},
        ],
    },
    "EquipmentType": {
        "app_label": "equipment",
        "allowed_fields": [
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "cost_per_mile", "label": "Cost Per Mile"},
        ],
    },
    "EquipmentManufacturer": {
        "app_label": "equipment",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "created", "label": "Created"},
            {"value": "modified", "label": "Modified"},
        ],
    },
    "LocationCategory": {
        "app_label": "location",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "created", "label": "Created"},
            {"value": "modified", "label": "Modified"},
        ],
    },
    "Location": {
        "app_label": "location",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "name", "label": "Name"},
            {"value": "location_category__name", "label": "Location Category Name"},
            {"value": "depot__name", "label": "Depot Name"},
            {"value": "address_line_1", "label": "Address Line 1"},
            {"value": "address_line_2", "label": "Address Line 2"},
            {"value": "city", "label": "City"},
            {"value": "state", "label": "State"},
            {"value": "zip_code", "label": "Zip Code"},
            {"value": "longitude", "label": "Longitude"},
            {"value": "latitude", "label": "Latitude"},
            {"value": "created", "label": "Created"},
            {"value": "modified", "label": "Modified"},
        ],
    },
    "Trailer": {
        "app_label": "equipment",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "code", "label": "Code"},
            {"value": "equipment_type__name", "label": "Equipment Type Name"},
            {"value": "make", "label": "Make"},
            {"value": "model", "label": "Model"},
            {"value": "year", "label": "Year"},
            {"value": "vin_number", "label": "Vin Number"},
            {"value": "state", "label": "State"},
            {"value": "owner", "label": "Owner"},
            {"value": "license_plate_number", "label": "License Plate #"},
            {"value": "license_plate_state", "label": "License Plate State"},
            {"value": "last_inspection", "label": "Last Inspection"},
            {"value": "is_leased", "label": "Is Leased?"},
            {"value": "leased_date", "label": "Leased Date"},
            {"value": "registration_number", "label": "Registration Number"},
            {"value": "registration_state", "label": "Registration State"},
            {"value": "registration_expiration", "label": "Registration Expiration"},
            {"value": "created", "label": "Created"},
            {"value": "modified", "label": "Modified"},
        ],
    },
    "Tractor": {
        "app_label": "equipment",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "code", "label": "Code"},
            {"value": "equipment_type__name", "label": "Equipment Type Name"},
            {"value": "model", "label": "Model"},
            {"value": "year", "label": "Year"},
            {"value": "state", "label": "State"},
            {"value": "license_plate_number", "label": "License Plate #"},
            {"value": "vin_number", "label": "Vin Number"},
            {"value": "primary_worker__code", "label": "Primary Worker Code"},
            {"value": "secondary_worker__code", "label": "Secondary Worker Code"},
            {"value": "owner_operated", "label": "Owner Operated?"},
            {"value": "fleet_code__code", "label": "Fleet Code"},
            {"value": "created", "label": "Created"},
            {"value": "modified", "label": "Modified"},
        ],
    },
    "DocumentClassification": {
        "app_label": "billing",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "name", "label": "Name"},
            {"value": "description", "label": "Description"},
            {"value": "created", "label": "Created"},
            {"value": "modified", "label": "Modified"},
        ],
    },
    "EmailProfile": {
        "app_label": "organization",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "name", "label": "Name"},
            {"value": "email", "label": "Email Address"},
            {"value": "protocol", "label": "Protocol"},
            {"value": "host", "label": "Host"},
            {"value": "port", "label": "Port"},
            {"value": "username", "label": "Username"},
            {"value": "created", "label": "Created"},
            {"value": "modified", "label": "Modified"},
        ],
    },
    "ServiceType": {
        "app_label": "shipment",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Name"},
            {"value": "code", "label": "Code"},
            {"value": "description", "label": "Description"},
            {"value": "created", "label": "Created"},
            {"value": "modified", "label": "Modified"},
        ],
    },
    "QualifierCode": {
        "app_label": "stops",
        "allowed_fields": [
            {"value": "organization__name", "label": "Organization Name"},
            {"value": "status", "label": "Status"},
            {"value": "code", "label": "Code"},
            {"value": "description", "label": "Description"},
            {"value": "created", "label": "Created"},
            {"value": "modified", "label": "Modified"},
        ],
    },
}


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


def validate_model(model_name: str) -> None:
    """Validate if the provided model name is allowed.

    Args:
        model_name (str): The name of the model to validate.

    Returns:
        None: this function does not return anything.

    Raises:
        InvalidModelException: If the model name is not in the allowed models list.
    """
    if model_name not in ALLOWED_MODELS:
        raise exceptions.InvalidModelException(f"Model {model_name} is not allowed.")


def get_user_and_model(user_id: ModelUUID, model_name: str) -> tuple[User, type[Model]]:
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
    model = apps.get_model(ALLOWED_MODELS[model_name]["app_label"], model_name)
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
        for field in ALLOWED_MODELS[model_name]["allowed_fields"]
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
        "pdf": lambda: generate_pdf(
            df=df, buffer=report_buffer, organization_id=user.organization_id
        ),
    }

    generate_func = format_functions.get(file_format.lower())
    if not generate_func:
        raise ValueError("Invalid file format")

    generate_func()

    return report_buffer, file_name
