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

import logging
import typing

from django.db import connection, transaction
from django.db.backends.utils import truncate_name
from organization.exceptions import InvalidOperationError
from organization.services.conditional_logic import (
    AVAILABLE_OPERATIONS,
    OPERATION_MAPPING,
)
from organization.services.table_choices import get_column_names
from utils.models import OperationChoices
from utils.types import ConditionalLogic, ModelUUID

logger = logging.getLogger(__name__)

info, warning, exception, debug = (
    logger.info,
    logger.warning,
    logger.exception,
    logger.debug,
)


# fmt: off
def format_value_for_operation(operation: str, value: typing.Any) -> str | None:
    """Formats a value for SQL operations based on the specified operation type.

    This function takes an operation type and a value, returning a formatted string representation of the value
    that is suitable for use in SQL queries. The formatting depends on the operation type. For example,
    for 'contains' or 'icontains', the value is formatted with percentage symbols for SQL LIKE queries.
    For 'in' or 'not_in', the value is formatted as a tuple, suitable for SQL IN clauses.

    Args:
        operation (str): The type of SQL operation. Supported operations include 'contains', 'icontains',
                         'in', 'not_in', 'isnull', and 'not_isnull'.
        value (typing.Any): The value to be formatted for the SQL operation. Can be of any type that needs to
                            be formatted into a string for SQL queries.

    Returns:
        str | None: The formatted value as a string suitable for SQL queries, or None for operations that
                    don't require a value (e.g., 'isnull', 'not_isnull').

    Raises:
        InvalidOperationError: If the 'operation' argument is not a recognized type.
    """
    if operation not in AVAILABLE_OPERATIONS:
        raise InvalidOperationError(f"Operation {operation} is not supported.")

    if operation in {OperationChoices.CONTAINS, OperationChoices.ICONTAINS}:
        return f"'%{value}%'"
    elif operation in {OperationChoices.IN, OperationChoices.NOT_IN}:
        if isinstance(value, list):
            return f"({','.join([f'\'{x}\'' for x in value])})"
        else:
            return f"('{value}')"
    elif operation in {"isnull", "not_isnull"}:
        return None
    else:
        return f"'{value}'"
# fmt: on


def build_condition_string(column: str, operation: str, value) -> str:
    """Builds a SQL condition string for a given column, operation, and value.

    This function constructs a SQL condition string using the provided column name, operation, and value.
    It utilizes the `format_value_for_operation` function to format the value based on the operation type.

    Args:
        column (str): The name of the database column.
        operation (str): The SQL operation to be performed on the column.
        value: The value to be used in the operation.

    Returns:
        str: A SQL condition string.

    Raises:
        InvalidOperationError: If the operation type is not supported.
    """
    if operation not in AVAILABLE_OPERATIONS:
        raise InvalidOperationError(f"Operation {operation} is not supported.")

    formatted_value = format_value_for_operation(operation, value)
    if operation in {OperationChoices.IS_NULL, OperationChoices.IS_NOT_NULL}:
        return f"{column} {OPERATION_MAPPING[operation]}"
    else:
        return f"{column} {OPERATION_MAPPING[operation]} {formatted_value}"


def build_conditional_logic_sql(conditional_logic: ConditionalLogic) -> str:
    """Constructs a SQL conditional logic string from a given ConditionalLogic object.

    This function takes a ConditionalLogic object, which contains conditions, and constructs a SQL string
    representing the logical 'AND' combination of these conditions.

    Args:
        conditional_logic (ConditionalLogic): An object containing a list of condition dictionaries,
                                              where each dictionary specifies a column, an operation, and a value.

    Returns:
        str: A SQL string representing the combined conditions.

    Raises:
        InvalidOperationError: If any condition contains an unsupported operation.
    """
    conditions = [
        build_condition_string(
            column=f"new.{condition['column']}",
            operation=condition["operation"],
            value=condition["value"],
        )
        for condition in conditional_logic["conditions"]
        if condition["operation"] in AVAILABLE_OPERATIONS
    ]
    return " AND ".join(conditions)


def create_insert_field_string(*, fields: list[str]) -> str:
    """Creates a string of field-value pairs for use in a SQL INSERT statement.

    This function generates a string representation of field-value pairs, excluding certain fields like 'id',
    'created', 'modified', and 'organization_id'. It is used in constructing a dynamic SQL INSERT statement.

    Args:
        fields (list[str]): A list of field names to include in the INSERT statement.

    Returns:
        str: A string of field-value pairs for a SQL INSERT statement.
    """
    excluded_fields = ["id", "created", "modified", "organization_id"]
    field_strings = [
        (
            f"'{truncate_name(field, connection.ops.max_name_length())}',"
            f" new.{truncate_name(field, connection.ops.max_name_length())}"
        )
        for field in fields
        if field not in excluded_fields
    ]
    return (
        ", ".join(field_strings[:-1])
        + (", " if len(field_strings) > 1 else "")
        + field_strings[-1]
    )


@transaction.atomic
def create_insert_function(
    *,
    listener_name: str,
    function_name: str,
    fields: list[str],
    organization_id: ModelUUID,
    conditional_logic: ConditionalLogic | None = None,
) -> None:
    """Creates a SQL INSERT function with conditional logic.

    This function programmatically creates a PostgreSQL function that performs a conditional INSERT operation.
    The function is created using provided parameters like function name, fields to insert, and conditional logic.
    The function triggers a NOTIFY command with a JSON payload of the inserted fields on successful conditional
    insertion.

    Args:
        listener_name (str): The name of the PostgreSQL notification listener.
        function_name (str): The name of the PostgreSQL function to be created.
        fields (list[str]): The list of fields to be included in the INSERT operation.
        organization_id (ModelUUID): The UUID of the organization, used in the conditional logic.
        conditional_logic (ConditionalLogic | None): Optional. The conditional logic to apply to the INSERT operation.

    Returns:
        None: This function does not return anything.
    """
    fields_string = create_insert_field_string(fields=fields)
    where_clause = (
        build_conditional_logic_sql(conditional_logic) if conditional_logic else "TRUE"
    )

    with connection.cursor() as cursor:
        cursor.execute(
            f"""
            CREATE OR REPLACE FUNCTION {function_name}()
            RETURNS trigger
            LANGUAGE 'plpgsql'
            AS $BODY$
            BEGIN
                IF TG_OP = 'INSERT' AND NEW.organization_id = '{organization_id}' AND ({where_clause}) THEN
                    PERFORM pg_notify('{listener_name}',
                        json_build_object(
                            {fields_string}
                        )::text);
                END IF;
                RETURN NULL;
            END
            $BODY$;
            """
        )


@transaction.atomic
def create_insert_trigger(
    *,
    trigger_name: str,
    table_name: str,
    function_name: str,
    listener_name: str,
    organization_id: ModelUUID,
    conditional_logic: ConditionalLogic | None = None,
) -> None:
    """Creates a database trigger for the INSERT operation on a specified table.

    This function programmatically creates a PostgreSQL trigger that is activated after an INSERT operation
    on the specified table. The trigger calls a predefined function to perform additional operations.

    Args:
        trigger_name (str): The name of the trigger to be created.
        table_name (str): The name of the table on which the trigger is to be set.
        function_name (str): The name of the function that the trigger should execute.
        listener_name (str): The name of the notification listener.
        organization_id (ModelUUID): The UUID of the organization, used in conditional logic.
        conditional_logic (ConditionalLogic | None): Optional. The conditional logic to be used in the trigger.

    Returns:
        None: This function does not return anything.

    Raises:
        OperationalError: If the trigger creation fails.
    """
    fields = get_column_names(table_name=table_name)
    create_insert_function(
        function_name=function_name,
        fields=fields,
        listener_name=listener_name,
        organization_id=organization_id,
        conditional_logic=conditional_logic,
    )

    with connection.cursor() as cursor:
        e_table_name = connection.ops.quote_name(table_name)
        e_trigger_name = connection.ops.quote_name(trigger_name)
        e_function_name = connection.ops.quote_name(function_name)
        query = f"""
            CREATE or REPLACE TRIGGER {e_trigger_name}
            AFTER INSERT
            ON {e_table_name}
            FOR EACH ROW
            EXECUTE PROCEDURE {e_function_name}();
            """
        cursor.execute(query)
    info(f"Created function {function_name} and trigger {trigger_name}.")


def create_update_field_string(*, fields: list[str]) -> str:
    """Generates a SQL string for field comparison in a PostgreSQL UPDATE trigger function.

    This function constructs a string of SQL conditions that checks if any of the specified fields
    have been changed during an UPDATE operation. Excludes certain system fields like 'id', 'created', etc.

    Args:
        fields (list[str]): A list of field names for which changes are to be checked.

    Returns:
        str: A SQL string containing the comparison logic for the specified fields.
    """
    excluded = {"id", "created", "modified", "organization_id"}
    comparisons = [
        (
            f"OLD.{truncate_name(field, connection.ops.max_name_length())} IS DISTINCT FROM "
            f"NEW.{truncate_name(field, connection.ops.max_name_length())}"
        )
        for field in fields
        if field not in excluded
    ]
    return f"({' OR '.join(comparisons)})"


@transaction.atomic
def create_update_function(
    *,
    listener_name: str,
    function_name: str,
    fields: list[str],
    organization_id: ModelUUID,
    conditional_logic: ConditionalLogic | None = None,
) -> None:
    """Creates a database function for handling UPDATE operations with conditional logic.

    This function programmatically creates a PostgreSQL function that performs a conditional notification
    on UPDATE operations. It checks for changes in specified fields and organization_id.

    Args:
        listener_name (str): The name of the notification listener.
        function_name (str): The name of the function to be created.
        fields (list[str]): The list of fields to check for changes.
        organization_id (ModelUUID): The UUID of the organization for conditional checks.
        conditional_logic (ConditionalLogic | None): Optional. Additional conditional logic for the function.

    Returns:
        None: This function does not return anything.

    Raises:
        OperationalError: If the function creation fails.
    """
    fields_string = create_insert_field_string(fields=fields)
    comparison_string = create_update_field_string(fields=fields)

    # Use Django's truncate_name to ensure the name doesn't exceed the database's max name length
    # and is safely quoted.
    quoted_function_name = truncate_name(
        function_name, connection.ops.max_name_length()
    )
    quoted_listener_name = truncate_name(
        listener_name, connection.ops.max_name_length()
    )

    with connection.cursor() as cursor:
        cursor.execute(
            f"""
            CREATE OR REPLACE FUNCTION {quoted_function_name}()
            RETURNS trigger
            LANGUAGE 'plpgsql'
            AS $BODY$
            DECLARE
            BEGIN
                IF (TG_OP = 'UPDATE' AND {comparison_string} AND NEW.organization_id = '{organization_id}') THEN
                    PERFORM pg_notify('{quoted_listener_name}',
                    json_build_object(
                        {fields_string}
                    )::text);
                END IF;
                RETURN NULL;
            END
            $BODY$;
            """
        )
    info(f"Created function {function_name}.")


@transaction.atomic
def create_update_trigger(
    *,
    trigger_name: str,
    table_name: str,
    function_name: str,
    listener_name: str,
    organization_id: ModelUUID,
    conditional_logic: ConditionalLogic | None = None,
) -> None:
    """Creates a database trigger for the UPDATE operation on a specified table.

    This function creates a PostgreSQL trigger that activates after an UPDATE operation on the specified table.
    The trigger calls a predefined function for additional processing based on the update.

    Args:
        trigger_name (str): The name of the trigger to be created.
        table_name (str): The name of the table on which the trigger is to be set.
        function_name (str): The name of the function that the trigger should execute.
        listener_name (str): The name of the notification listener.
        organization_id (ModelUUID): The UUID of the organization, used in conditional logic.
        conditional_logic (ConditionalLogic | None): Optional. Additional conditional logic for the trigger.

    Returns:
        None: This function does not return anything.

    Raises:
        OperationalError: If the trigger creation fails.
    """
    fields = get_column_names(table_name=table_name)

    create_update_function(
        function_name=function_name,
        fields=fields,
        listener_name=listener_name,
        organization_id=organization_id,
        conditional_logic=conditional_logic,
    )

    # Use Django's truncate_name to ensure the name doesn't exceed the database's max name length
    # and is safely quoted.
    quoted_trigger_name = truncate_name(trigger_name, connection.ops.max_name_length())
    quoted_table_name = truncate_name(table_name, connection.ops.max_name_length())
    quoted_function_name = truncate_name(
        function_name, connection.ops.max_name_length()
    )

    with connection.cursor() as cursor:
        cursor.execute(
            f"""
            CREATE or REPLACE TRIGGER {quoted_trigger_name}
            AFTER UPDATE ON public.{quoted_table_name}
            FOR EACH ROW
            EXECUTE PROCEDURE {quoted_function_name}();
            """
        )
    info(f"Created function {function_name} and trigger {trigger_name}.")


@transaction.atomic
def drop_trigger_and_function(
    *, trigger_name: str, function_name: str, table_name: str
) -> None:
    """Drops an existing trigger and its associated function from a specified table.

    This function removes a specified trigger and function from the database, if they exist.
    It first checks for their existence before attempting to drop them.

    Args:
        trigger_name (str): The name of the trigger to be dropped.
        function_name (str): The name of the associated function to be dropped.
        table_name (str): The table from which the trigger and function are to be removed.

    Returns:
        None: This function does not return anything.

    Raises:
        ValueError: If either the trigger or function does not exist.
    """
    trigger = check_trigger_exists(table_name=table_name, trigger_name=trigger_name)
    function = check_function_exists(function_name=function_name)

    # If the trigger or function do not exist, return early.
    if not trigger or not function:
        info(
            f"Trigger {trigger_name} or function {function_name} does not exist. Skipping drop."
        )
        return

    # Use Django's truncate_name to ensure the name doesn't exceed the database's max name length
    # and is safely quoted.
    quoted_trigger_name = truncate_name(trigger_name, connection.ops.max_name_length())
    quoted_table_name = truncate_name(table_name, connection.ops.max_name_length())
    quoted_function_name = truncate_name(
        function_name, connection.ops.max_name_length()
    )

    with connection.cursor() as cursor:
        cursor.execute(
            f"""
                DROP TRIGGER IF EXISTS {quoted_trigger_name} ON public.{quoted_table_name};
                DROP FUNCTION IF EXISTS {quoted_function_name}();
                """
        )
    info(f"Dropped trigger {trigger_name} and function {function_name}.")


def check_trigger_exists(*, table_name: str, trigger_name: str) -> bool:
    """Checks if a specified trigger exists on a given table.

    This function queries the database to determine whether a trigger with the given name
    exists on the specified table.

    Args:
        table_name (str): The name of the table to check.
        trigger_name (str): The name of the trigger to look for.

    Returns:
        bool: True if the trigger exists, False otherwise.
    """
    with connection.cursor() as cursor:
        query = """SELECT EXISTS(
            SELECT 1 FROM information_schema.triggers
            WHERE event_object_table = %s
            AND trigger_name = %s)
        """
        cursor.execute(query, [table_name, trigger_name])

        return bool(cursor.fetchone()[0])


def check_function_exists(*, function_name: str) -> bool:
    """Checks if a specified function exists in the database.

    This function queries the database to determine whether a function with the given name exists.

    Args:
        function_name (str): The name of the function to check.

    Returns:
        bool: True if the function exists, False otherwise.
    """
    with connection.cursor() as cursor:
        query = "SELECT EXISTS (SELECT 1 FROM pg_proc WHERE proname = %s)"
        cursor.execute(query, [function_name])
        return bool(cursor.fetchone()[0])


def _get_routine_definition(
    *, routine_name: str, routine_schema: str = "public"
) -> str:
    """Retrieves the definition of a specified routine from the database.

    This function queries the information_schema of the PostgreSQL database to get the definition
    of a routine (such as a function or procedure) based on its name. It only searches within the
    'public' schema.

    Args:
        routine_name (str): The name of the routine whose definition is to be retrieved.

    Returns:
        str: The definition of the routine as a string. If the routine is not found, returns None.

    Raises:
        OperationalError: If there's an issue executing the database query.

    Notes:
        - This function is intended for internal use (prefixed with an underscore).
        - The routine is expected to be within the 'public' schema of the database.
        - The function uses Django's database connection to execute the SQL query.
    """
    with connection.cursor() as cursor:
        query = """
        SELECT
            routine_name,
            routine_definition
        FROM information_schema.routines
        WHERE routine_name = %s
        AND routine_schema = %s
        """
        cursor.execute(query, [routine_name, routine_schema])
        return cursor.fetchone()[1]
