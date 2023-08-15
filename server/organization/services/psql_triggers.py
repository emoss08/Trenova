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

from django.db import connection, transaction
from django.db.backends.utils import truncate_name

from utils.types import ModelUUID

from .table_choices import TableChoiceService

table_service = TableChoiceService()


def create_insert_field_string(*, fields: list[str]) -> str:
    """Creates a comma-separated string of field names for a SQL query.

    This function takes a list of field names and creates a string that can be used
    in a SQL query to specify which fields to insert, update, or select. The
    resulting string is in the format "'field_name', new.field_name" for each field,
    separated by commas. Fields with names in the excluded_fields list are not
    included in the result.

    Args:
        fields (list of str): A list of field names to include in the result.

    Returns:
        str: A string in the format "'field1', new.field1, 'field2', new.field2, ..."
        representing the specified fields.

    """
    excluded_fields: list[str] = ["id", "created", "modified", "organization_id"]
    field_strings: list[str] = [
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
) -> None:
    """Creates a PL/pgSQL trigger function that sends a notification on INSERT.

    This function creates a PL/pgSQL trigger function that sends a JSON notification
    containing the specified fields whenever a row is inserted into the associated
    table. The function is created or replaced in the database using the provided
    function_name and trigger_name. The notification is sent to a channel named
    listener_name, which can be used to listen for notifications in a separate
    process.

    Args:
        listener_name (str): The name of the channel to send notifications to.
        function_name (str): The name of the function to create or replace.
        fields (list of str): A list of field names to include in the notification.
        organization_id (ModelUUID): The ID of the organization to monitor for INSERTs.

    Returns:
        None: This function has no return value.

    Raises:
        django.db.utils.DatabaseError: If there is an error executing the SQL query.

    """
    fields_string: str = create_insert_field_string(fields=fields)
    with connection.cursor() as cursor:
        cursor.execute(
            f"""
                CREATE or REPLACE FUNCTION {function_name}()
                RETURNS trigger
                LANGUAGE 'plpgsql'
                as $BODY$
                declare
                begin
                    if (tg_op = 'INSERT' AND NEW.organization_id = '{organization_id}')  then
                    perform pg_notify('{listener_name}',
                    json_build_object(
                        {fields_string}
                    )::text);
                    end if;
                    return null;
                end
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
) -> None:
    """Creates a PL/pgSQL trigger and function for sending a notification on INSERT.

    This function creates a PL/pgSQL trigger and function that sends a JSON notification
    containing the names and values of all fields in the specified table whenever a row
    is inserted into the table. The function and trigger are created or replaced in the
    database using the provided names. The notification is sent to a channel named
    listener_name, which can be used to listen for notifications in a separate process.

    Args:
        trigger_name (str): The name of the trigger to create or replace.
        table_name (str): The name of the table to monitor for INSERTs.
        function_name (str): The name of the function to create or replace.
        listener_name (str): The name of the channel to send notifications to.
        organization_id (ModelUUID): The ID of the organization to monitor for INSERTs.

    Returns:
        None: This function has no return value.

    Raises:
        django.db.utils.DatabaseError: If there is an error executing the SQL query.

    """

    fields: list[str] = table_service.get_column_names(table_name=table_name)
    create_insert_function(
        function_name=function_name,
        fields=fields,
        listener_name=listener_name,
        organization_id=organization_id,
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


def create_update_field_string(*, fields: list[str]) -> str:
    """Creates a SQL comparison string that compares the old and new values of specified fields.

    This function receives a list of field names, removes the excluded ones, and for each resulting field
    creates a SQL comparison that checks if the old value is distinct from the new value.

    The result is a SQL condition formatted as:
    "(OLD.<field1> IS DISTINCT FROM NEW.<field1> OR OLD.<field2> IS DISTINCT FROM NEW.<field2> ...)"

    Excluded fields: `id`, `created`, `modified`, `organization_id`.

    IMPORTANT NOTE: Be careful with SQL injection risks. Although the truncate_name function is used to sanitize
                    field names, the list of fields should not directly come from user input or should be thoroughly
                    reviewed and sanitized.

    Args:
        fields (list[str]): A list of the names of the fields to be checked for differences.

    Returns:
        str: A string of SQL comparisons for distinct old and new values of provided fields, separated by OR.
    """
    excluded: set[str] = {"id", "created", "modified", "organization_id"}
    comparisons: list[str] = [
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
) -> None:
    """Creates or replaces a PostgreSQL function that can be executed whenever specific trigger events occur.

    This function is specifically designed to be executed by a trigger after an update operation on a table.

    When the function is executed, it sends a JSON object as a PostgreSQL NOTIFY payload. The object contains the
    field names and their corresponding new values.

    It creates a string representation of database fields for a SQL insert statement and a string representation
    of a SQL conditional check to be used in the function's comparison clause `IF (TG_OP = 'UPDATE'...)`.

    To ensure data safety and avoid SQL injections, we use Django's `truncate_name` function which truncates
    the provided name to the max allowed length (depending on your database settings) and adds quotation marks
    around it.

    IMPORTANT NOTE: fields_string and comparison_string should be constructed in a way that they are safe
                    from SQL injection. If they are based on user input, they should be thoroughly reviewed and
                    sanitized.

    This function should be called in a database transaction context.

    Args:
        listener_name (str): The name of the listener that the function will notify when executed.
        function_name (str): The name of the function to be created or replaced.
        fields (list[str]): A list of the names of the fields that the function will include in the notification payload.
        organization_id (ModelUUID): The UUID of the organization associated with the function.

    Returns:
        None: This function does not return anything.
    """
    fields_string: str = create_insert_field_string(fields=fields)
    comparison_string: str = create_update_field_string(fields=fields)

    # Use Django's truncate_name to ensure the name doesn't exceed the database's max name length
    # and is safely quoted.
    quoted_function_name = truncate_name(
        function_name, connection.ops.max_name_length()
    )
    quoted_listener_name = truncate_name(
        listener_name, connection.ops.max_name_length()
    )

    # Note: fields_string and comparison_string should be constructed in a way that they are safe
    # from SQL injection. If they are based on user input, they should be carefully reviewed and sanitized.

    with connection.cursor() as cursor:
        cursor.execute(
            f"""
                 CREATE or REPLACE FUNCTION {quoted_function_name}()
                 RETURNS trigger
                 LANGUAGE 'plpgsql'
                 as $BODY$
                 declare
                 begin
                     IF (TG_OP = 'UPDATE' AND {comparison_string} AND NEW.organization_id = %s) THEN
                         PERFORM pg_notify(%s,
                         json_build_object(
                             {fields_string}
                         )::text);
                     END IF;
                     RETURN NULL;
                 END
                 $BODY$;
                 """,
            [str(organization_id), quoted_listener_name],
        )


@transaction.atomic
def create_update_trigger(
    *,
    trigger_name: str,
    table_name: str,
    function_name: str,
    listener_name: str,
    organization_id: ModelUUID,
) -> None:
    """Creates or replaces a trigger in a PostgreSQL table that executes a specific function whenever an update happens.

    This function uses Django's database and transaction management to:

    1. First call a helper method to get the column names of the table.
    2. Then create or replace a specified database function with the given column names.
    3. Afterwards, this will create or replace a trigger on the specified table.

    The trigger will be set up to run `AFTER UPDATE` for every row in the table. Whenever an update event occurs,
    the trigger runs the specified function.

    To ensure the data safety and avoid SQL injections, we use Django's `truncate_name` which truncates the name
    to the max allowed length (depending on your database settings) and adds the required quotation marks around it.

    This function must be called in a database transaction context.

    Args:
        trigger_name (str): The name of the trigger to be created or replaced.
        table_name (str): The name of the table where the trigger will be placed.
        function_name (str): The name of the function to be executed when the trigger activates.
        listener_name (str): The name of the listener associated with the trigger.
        organization_id (ModelUUID): The UUID of the organization associated with the trigger.

    Returns:
        None: This function does not return anything.
    """
    fields: list[str] = table_service.get_column_names(table_name=table_name)

    create_update_function(
        function_name=function_name,
        fields=fields,
        listener_name=listener_name,
        organization_id=organization_id,
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


@transaction.atomic
def drop_trigger_and_function(
    *, trigger_name: str, function_name: str, table_name: str
) -> None:
    """Deletes a PL/pgSQL trigger and function.

    This function drops a PL/pgSQL trigger and function from the database.

    Args:
        trigger_name (str): The name of the trigger to delete.
        function_name (str): The name of the function to delete.
        table_name (str): The name of the table the trigger is associated with.

    Returns:
        None: This function has no return value.

    Raises:
        django.db.utils.DatabaseError: If there is an error executing the SQL query.
    """

    trigger = check_trigger_exists(table_name=table_name, trigger_name=trigger_name)
    function = check_function_exists(function_name=function_name)

    if not trigger or not function:
        raise ValueError(
            f"Trigger {trigger_name} or function {function_name} does not exist."
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
                DROP TRIGGER IF EXISTS {quoted_trigger_name} ON public.{quoted_table_name};
                DROP FUNCTION IF EXISTS {quoted_function_name}();
                """
        )


def check_trigger_exists(*, table_name: str, trigger_name: str) -> bool:
    """
    Check if a trigger with the given name exists on the specified table in the database.

    Args:
        table_name (str): The name of the table to check for the trigger.
        trigger_name (str): The name of the trigger to check for.

    Returns:
        bool: True if the trigger exists on the table, False otherwise.

    Raises:
        django.db.utils.DatabaseError: If there is an error executing the SQL query.
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
    """
    Check if a function with the given name exists in the database.

    Args:
        function_name (str): The name of the function to check for.

    Returns:
        bool: True if the function exists in the database, False otherwise.

    Raises:
        django.db.utils.DatabaseError: If there is an error executing the SQL query.
    """

    with connection.cursor() as cursor:
        query = "SELECT EXISTS (SELECT 1 FROM pg_proc WHERE proname = %s)"
        cursor.execute(query, [function_name])
        return bool(cursor.fetchone()[0])
