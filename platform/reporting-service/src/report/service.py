from typing import List, Optional

from report.exceptions import DataFrameCreationError, InvalidDeliveryMethodError
from report.schemas import Relationship
from report.utils import generate_report_file, prepare_dataframe, upload_file

REPORT_BUCKET_NAME = "trenova-user-reports"


def report_generate(
    *,
    table_name: str,
    columns: List[str],
    relationships: Optional[List[Relationship]],
    organization_id: str,
    business_unit_id: str,
    file_format: str,
    delivery_method: str,
) -> str:
    """Generates a report based on the provided database table, formats it, and delivers it via the specified method.

    This function extracts data from the specified `table_name` using the columns listed in `columns`. It then formats
    the data into a report in the specified `file_format` and delivers it using the method specified in `delivery_method`.
    The supported delivery methods are 'email' and 'local'. If the `delivery_method` is not supported, it raises an
    InvalidDeliveryMethodError.

    The report is temporarily saved to a local file in the "/tmp" directory, then uploaded to a specified cloud storage
    bucket (defined in REPORT_BUCKET_NAME). Finally, the path to the uploaded file in the bucket is returned.

    Args:
        table_name (str): The name of the database table from which to extract data.
        columns (List[str]): A list of column names to be included in the report.
        organization_id: str: The organization id to filter the data.
        business_unit_id: str: The business unit id to filter the data.
        file_format (str): The format of the report file. Examples include 'csv', 'xlsx', etc.
        delivery_method (str): The method of delivering the report. Currently supports 'email' and 'local'.

    Returns:
        str: The full path to the uploaded report file in the cloud storage bucket.

    Raises:
        InvalidDeliveryMethodError: If the `delivery_method` is neither 'email' nor 'local'.
        DataFrameCreationError: If there is an issue creating the DataFrame from the database.

    Note:
        - The function uses 'prepare_dataframe' to create a DataFrame from the database.
        - 'generate_report_file' is used to convert the DataFrame into a file.
        - 'upload_file' uploads the file to the cloud storage bucket defined by REPORT_BUCKET_NAME.

    Example:
        >>> generate(table_name="user_data", columns=["id", "name"], file_format="csv", delivery_method="email")
        'gs://trenova-user-reports/path_to_file.csv'
    """
    if delivery_method not in ["email", "local"]:
        raise InvalidDeliveryMethodError(f"Invalid delivery method: {delivery_method}")

    df = prepare_dataframe(
        columns=columns,
        organization_id=organization_id,
        relationships=relationships,
        business_unit_id=business_unit_id,
        table_name=table_name,
    )
    if df is None:
        raise DataFrameCreationError("An error occurred while creating the DataFrame.")

    report_buffer, file_name = generate_report_file(
        df=df, table_name=table_name, file_format=file_format
    )

    temp_file_path = f"/tmp/{file_name}"
    report_buffer.seek(0)
    with open(temp_file_path, "wb") as file:
        file.write(report_buffer.read())

    file_path = upload_file(file_path=temp_file_path, bucket_name=REPORT_BUCKET_NAME)

    return file_path
