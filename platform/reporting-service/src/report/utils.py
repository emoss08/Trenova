import logging
import os
import sys
import uuid
from io import BytesIO
from typing import Dict, List, Optional, Tuple

import pandas as pd
from minio import Minio, error
from sqlalchemy import TextClause, inspect
from sqlalchemy.exc import SQLAlchemyError
from sqlalchemy.sql import text

from database import SessionLocal, engine
from report.schemas import Relationship

client = Minio(
    "localhost:9000", access_key="minio", secret_key="minio123", secure=False
)

# Setup logging
file_handler = logging.FileHandler(filename="tmp.log")
stdout_handler = logging.StreamHandler(stream=sys.stdout)
handlers = [file_handler, stdout_handler]

logging.basicConfig(
    level=logging.DEBUG,
    format="[%(asctime)s] {%(filename)s:%(lineno)d} %(levelname)s - %(message)s",
    handlers=handlers,  # type:ignore
)
logger = logging.getLogger(__name__)


def _create_report_bucket(*, bucket_name: str) -> None:
    """Creates a new bucket with the specified name.

    Args:
        bucket_name: Name of the bucket to create.

    Returns:
        None: This function does not return anything.
    """
    try:
        if not client.bucket_exists(bucket_name):
            client.make_bucket(bucket_name)
            logger.info(f"Bucket created: {bucket_name}")
        else:
            logger.info(f"Bucket {bucket_name} already exists.")
    except error.S3Error as e:
        logger.error(f"Error occurred while creating bucket {bucket_name}: {e}")


def upload_file(*, file_path: str, bucket_name: str) -> str:
    """Uploads a file to a specified bucket and returns the URL to the file.

    Args:
        file_path: Path to the file to upload.
        bucket_name: Name of the bucket to upload the file.

    Returns:
        str: URL of the uploaded file.
    """
    try:
        if not client.bucket_exists(bucket_name):
            _create_report_bucket(bucket_name=bucket_name)

        file_name = os.path.basename(file_path)
        randomized_file_name = f"{uuid.uuid4()}_{file_name}"
        with open(file_path, "rb") as file_data:
            client.put_object(
                bucket_name, randomized_file_name, file_data, os.stat(file_path).st_size
            )

        file_url = client.presigned_get_object(bucket_name, randomized_file_name)
        logger.info(f"File uploaded successfully: {file_url}")
        return file_url

    except (error.S3Error, IOError) as e:
        logger.error(f"Failed to upload file {file_path}: {e}")
        return ""


def _convert_datetime_columns(*, df: pd.DataFrame) -> None:
    """Convert timezone-aware datetime columns in a DataFrame to naive datetimes.

    Args:
        df (pd.DataFrame): The DataFrame whose datetime columns are to be converted.

    Returns:
        None: this function does not return anything.
    """
    for column in df.columns:
        if isinstance(df[column].dtype, pd.DatetimeTZDtype):
            df[column] = df[column].dt.tz_convert(None)


def prepare_dataframe(
    *,
    columns: List[str],
    organization_id: str,
    relationships: Optional[List[Relationship]],
    business_unit_id: str,
    table_name: str,
) -> pd.DataFrame | None:
    """Executes the constructed query using a transactional session and fetches all results.

    Args:
        columns: List of column names to retrieve.
        table_name: Name of the table to query.

    Returns:
        List of rows fetched from the database.
    """
    query_str = _construct_query(
        columns=columns,
        relationships=relationships,
        table_name=table_name,
    )
    try:
        with SessionLocal() as session:
            df = pd.read_sql_query(
                sql=query_str,
                con=engine,
                params={
                    "organization_id": organization_id,
                    "business_unit_id": business_unit_id,
                },
            )

            _convert_datetime_columns(df=df)
            return df
    except SQLAlchemyError as e:
        logger.exception(f"An error occurred: {e}")
        return None


def _validate_columns(*, table_name: str, input_columns: List[str]) -> List[str]:
    """Validates that the input columns exist in the specified table in the database.

    Args:
        table_name: Name of the table to validate column names against.
        input_columns: List of column names to validate.

    Returns:
        A list of valid column names that exist both in the input list and the table.
    """
    try:
        inspector = inspect(engine)
        actual_columns = {
            column["name"] for column in inspector.get_columns(table_name)
        }
        valid_columns = [column for column in input_columns if column in actual_columns]

        # Log the validation results.
        if len(valid_columns) != len(input_columns):
            missing_columns = set(input_columns) - actual_columns
            logger.warning(
                f"Warning: The following columns do not exist in the table '{table_name}': {missing_columns}. Excluding them from the query."
            )
        return valid_columns

    except SQLAlchemyError as e:
        logger.exception(f"Error validating columns for {table_name}: {e}")
        return []


def _validate_relationship_columns(*, relationship: Relationship) -> Relationship:
    try:
        inspector = inspect(engine)
        actual_columns = {
            column["name"]
            for column in inspector.get_columns(relationship.referencedTable)
        }
        valid_columns = [
            column for column in relationship.columns if column in actual_columns
        ]

        if len(valid_columns) != len(relationship.columns):
            missing_columns = set(relationship.columns) - actual_columns
            logger.warning(
                f"Warning: The following columns do not exist in the table '{relationship.referencedTable}': {missing_columns}. Excluding them from the query."
            )

        # Check if the foreign key exists in the referenced table
        if relationship.foreignKey not in actual_columns:
            logger.warning(
                f"Warning: The foreign key '{relationship.foreignKey}' does not exist in the table '{relationship.referencedTable}'. It might be in the main table."
            )

        return Relationship(
            foreignKey=relationship.foreignKey,
            referencedTable=relationship.referencedTable,
            columns=valid_columns,
        )

    except SQLAlchemyError as e:
        logger.exception(
            f"Error validating columns for {relationship.referencedTable}: {e}"
        )
        return Relationship(
            foreignKey=relationship.foreignKey,
            referencedTable=relationship.referencedTable,
            columns=[],
        )


def _construct_query(
    *,
    columns: List[str],
    relationships: Optional[List[Relationship]],
    table_name: str,
    schema_name: str = "public",
) -> TextClause:
    validated_columns = _validate_columns(table_name=table_name, input_columns=columns)
    main_columns_str = ", ".join(
        [f'"{table_name}"."{column}"' for column in validated_columns]
    )

    join_clauses = []
    select_columns = [main_columns_str]
    alias_count = 0

    if relationships:
        for relationship in relationships:
            validated_relationship = _validate_relationship_columns(
                relationship=relationship
            )
            foreign_key = validated_relationship.foreignKey
            related_table = validated_relationship.referencedTable
            rel_columns = validated_relationship.columns

            alias = f"{related_table}_alias_{alias_count}"
            alias_count += 1

            # Check if the foreign key exists in the main table
            if foreign_key in validated_columns:
                join_clause = f'LEFT JOIN "{schema_name}"."{related_table}" AS "{alias}" ON "{table_name}"."{foreign_key}" = "{alias}"."id"'
            else:
                # If the foreign key doesn't exist in the main table, assume it's in the related table
                join_clause = f'LEFT JOIN "{schema_name}"."{related_table}" AS "{alias}" ON "{table_name}"."id" = "{alias}"."{foreign_key}"'

            join_clauses.append(join_clause)

            select_columns.extend(
                [
                    f'"{alias}"."{col}" AS "{foreign_key}.{related_table}.{col}"'
                    for col in rel_columns
                ]
            )

    columns_str = ", ".join(select_columns)
    join_str = " ".join(join_clauses)

    constructed_query = text(
        f"""SELECT {columns_str}
           FROM "{schema_name}"."{table_name}"
           {join_str}
           WHERE "{table_name}".organization_id = :organization_id
           AND "{table_name}".business_unit_id = :business_unit_id"""
    )

    # Log the constructed query
    logger.info(f"Constructed Query: {constructed_query}")

    return constructed_query


def _generate_csv(*, df: pd.DataFrame, report_buffer: BytesIO) -> None:
    """Generates a CSV file from a DataFrame and writes it to a buffer.

    Args:
        df: The DataFrame to convert to CSV.
        report_buffer: The buffer to write the CSV data to.

    Returns:
        None: This function does not return anything.
    """
    df.to_csv(report_buffer, index=False)


def _generate_excel(*, df: pd.DataFrame, report_buffer: BytesIO) -> None:
    """Generates an Excel file from a DataFrame and writes it to a buffer.

    Args:
        df: The DataFrame to convert to Excel.
        report_buffer: The buffer to write the Excel data to.

    Returns:
        None: This function does not return anything.
    """
    # TODO(Wolfred): add support for multiple sheets
    df.to_excel(report_buffer, index=False)


def generate_report_file(
    *,
    df: pd.DataFrame,
    table_name: str,
    file_format: str,
) -> Tuple[BytesIO, str]:
    """Generates a report file in the specified format from a DataFrame.

    Args:
        df: The DataFrame to convert to a report file.
        table_name: The name of the table from which the data was extracted.
        file_format: The format of the report file to generate.

    Returns:
        Tuple[BytesIO, str]: A tuple containing the report buffer and the file name.
    """
    report_buffer = BytesIO()
    file_name = f"{table_name}-report.{file_format}"

    format_functions = {
        "csv": lambda: _generate_csv(df=df, report_buffer=report_buffer),
        "xlsx": lambda: _generate_excel(df=df, report_buffer=report_buffer),
    }

    generate_func = format_functions.get(file_format.lower())
    if not generate_func:
        logger.error(f"Unsupported file format: {file_format}")
        raise ValueError(f"Unsupported file format: {file_format}")

    generate_func()

    return report_buffer, file_name


def insert_task_status():
    pass
