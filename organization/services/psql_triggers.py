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
from typing import List

from django.db import connection
from .table_choices import TableChoiceService


def create_field_string(fields: List[str]) -> str:
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
    excluded_fields: List[str] = ["id", "created", "modified", "organization_id"]
    field_strings: List[str] = [
        f"'{field}', new.{field}" for field in fields if field not in excluded_fields
    ]
    return (
        ", ".join(field_strings[:-1])
        + (", " if len(field_strings) > 1 else "")
        + field_strings[-1]
    )


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
    fields_string: str = create_field_string(fields)
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
    fields: List[str] = TableChoiceService().get_column_names(table_name)
    create_insert_function(
        function_name=function_name,
        fields=fields,
        listener_name=listener_name,
    )

    with connection.cursor() as cursor:
        cursor.execute(
            f"""
            CREATE or REPLACE TRIGGER {trigger_name}
            AFTER INSERT
            ON {table_name}
            FOR EACH ROW
            EXECUTE PROCEDURE {function_name}();
            """
        )

def drop_trigger(*, trigger_name: str, function_name: str, table_name: str) -> None:
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
    with connection.cursor() as cursor:
        cursor.execute(
            f"""
            DROP TRIGGER IF EXISTS {trigger_name} ON {table_name};
            DROP FUNCTION IF EXISTS {function_name}();
            """
        )
