"""
COPYRIGHT 2023 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""


from django.db import connection, transaction

from .table_choices import TableChoiceService

table_service = TableChoiceService()


def create_insert_field_string(fields: list[str]) -> str:
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
        f"'{field}', new.{field}" for field in fields if field not in excluded_fields
    ]
    return (
        ", ".join(field_strings[:-1])
        + (", " if len(field_strings) > 1 else "")
        + field_strings[-1]
    )

@transaction.atomic
def create_insert_function(
    *, listener_name: str, function_name: str, fields: list[str]
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

    Returns:
        None: This function has no return value.

    Raises:
        django.db.utils.DatabaseError: If there is an error executing the SQL query.

    """
    fields_string: str = create_insert_field_string(fields)
    with connection.cursor() as cursor:
        cursor.execute(
            f"""
                CREATE or REPLACE FUNCTION {function_name}()
                RETURNS trigger
                LANGUAGE 'plpgsql'
                as $BODY$
                declare
                begin
                    if (tg_op = 'INSERT') then
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
    *, trigger_name: str, table_name: str, function_name: str, listener_name: str
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


def create_update_field_string(fields: list[str]) -> str:
    """
    Returns a SQL WHERE clause string that compares old and new field values for use in an UPDATE statement.

    Args:
        fields: A list of field names to compare. The list should be in the same order as the corresponding columns
            in the database table.

    Returns:
        A string containing a SQL WHERE clause that compares the old and new values for each field not excluded.
        Each field comparison is separated by ' OR '.

    The resulting SQL WHERE clause can be used in an UPDATE or UPDATE Trigger statement to update a table with
    only the fields that have changed. The clause only includes comparisons for fields that are not excluded.

    For example, if `fields` is `["name", "email", "phone"]`, the resulting SQL WHERE clause string might be:

    (OLD.name IS DISTINCT FROM NEW.name OR OLD.email IS DISTINCT FROM NEW.email OR OLD.phone IS DISTINCT FROM NEW.phone)

    This would compare the old and new values for all three fields, since none of them are excluded.

    Raises:
        None.
    """
    excluded: set[str] = {"id", "created", "modified", "organization_id"}
    return f"({' OR '.join(f'OLD.{f} IS DISTINCT FROM NEW.{f}' for f in fields if f not in excluded)})"


@transaction.atomic
def create_update_function(
    *, listener_name: str, function_name: str, fields: list[str]
) -> None:
    """
    Creates a PL/pgSQL trigger function that sends a notification on UPDATE.

    This function creates a PL/pgSQL trigger function that sends a JSON notification
    containing the specified fields whenever a row is updated in the associated
    table. The function is created or replaced in the database using the provided
    function_name and trigger_name. The notification is sent to a channel named
    listener_name, which can be used to listen for notifications in a separate
    process.

    Args:
        listener_name (str): The name of the channel to send notifications to.
        function_name (str): The name of the function to create or replace.
        fields (list of str): A list of field names to include in the notification.

    Returns:
        None: This function has no return value.

    Raises:
        django.db.utils.DatabaseError: If there is an error executing the SQL query.

    """
    fields_string: str = create_insert_field_string(fields)
    with connection.cursor() as cursor:
        cursor.execute(
            f"""
                CREATE or REPLACE FUNCTION {function_name}()
                RETURNS trigger
                LANGUAGE 'plpgsql'
                as $BODY$
                declare
                begin
                    if (tg_op = 'UPDATE') then
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
def create_update_trigger(
    *, trigger_name: str, table_name: str, function_name: str, listener_name: str
) -> None:
    """Creates a PL/pgSQL trigger and function for sending a notification on UPDATE.

    This function creates a PL/pgSQL trigger and function that sends a JSON notification
    containing the names and values of all fields in the specified table whenever a row
    is updated in the table. The function and trigger are created or replaced in the
    database using the provided names. The notification is sent to a channel named
    listener_name, which can be used to listen for notifications in a separate process.

    Args:
        trigger_name (str): The name of the trigger to create or replace.
        table_name (str): The name of the table to monitor for UPDATEs.
        function_name (str): The name of the function to create or replace.
        listener_name (str): The name of the channel to send notifications to.

    Returns:
        None: This function has no return value.

    Raises:
        django.db.utils.DatabaseError: If there is an error executing the SQL query.

    """
    fields: list[str] = table_service.get_column_names(table_name=table_name)

    create_update_function(
        function_name=function_name,
        fields=fields,
        listener_name=listener_name,
    )

    with connection.cursor() as cursor:
        cursor.execute(
            f"""
            CREATE or REPLACE TRIGGER {trigger_name}
            AFTER UPDATE ON {table_name}
            FOR EACH ROW
            EXECUTE PROCEDURE {function_name}();
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

    if trigger and function:
        with connection.cursor() as cursor:
            cursor.execute(
                f"""
            DROP TRIGGER IF EXISTS {trigger_name} ON {table_name};
            DROP FUNCTION IF EXISTS {function_name}();
            """
            )
    else:
        raise ValueError(
            f"Trigger {trigger_name} or function {function_name} does not exist."
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
        query = """
                    SELECT EXISTS(
            SELECT 1 FROM information_schema.triggers
            WHERE event_object_table = %s
            AND trigger_name = %s)
        """
        cursor.execute(query, [table_name, trigger_name])
        return cursor.fetchone()[0]


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
        query = """
            SELECT EXISTS (
                SELECT 1 FROM pg_proc
                WHERE proname = %s
            )
        """
        cursor.execute(query, [function_name])
        return cursor.fetchone()[0]
