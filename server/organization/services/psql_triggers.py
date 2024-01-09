# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 MONTA                                                                         -
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
from organization.services.conditional_logic import OPERATION_MAPPING
from organization.services.table_choices import get_column_names
from utils.types import ModelUUID


# fmt: off
def build_conditional_logic_sql(conditional_logic: dict) -> str:
    conditions = []
    for condition in conditional_logic["conditions"]:
        column = f"new.{condition['column']}"
        operation = OPERATION_MAPPING[condition["operation"]]
        value = condition["value"]

        if condition["operation"] in ["contains", "icontains"]:
            value = f"'%{value}%'"
        elif condition["operation"] == "in":
            value = f"({','.join(map(lambda x: f"'{x}'", value))})" if isinstance(value, list) else f"('{value}')"
        elif condition["operation"] == "isnull":
            conditions.append(f"{column} {operation}")
        elif condition["operation"] == "eq":
            conditions.append(f"{column} = '{value}'")
            continue

        conditions.append(f"{column} {operation} {value}")

    return " AND ".join(conditions)


# fmt: on
def create_insert_field_string(*, fields: list[str]) -> str:
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
    conditional_logic: dict = None,
) -> None:
    fields_string = create_insert_field_string(fields=fields)
    where_clause = (
        build_conditional_logic_sql(conditional_logic) if conditional_logic else "TRUE"
    )

    print(f"Where Clause: {where_clause}")

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
    conditional_logic: dict = None,
) -> None:
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


def create_update_field_string(*, fields: list[str]) -> str:
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
    conditional_logic: dict = None,
) -> None:
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


@transaction.atomic
def create_update_trigger(
    *,
    trigger_name: str,
    table_name: str,
    function_name: str,
    listener_name: str,
    organization_id: ModelUUID,
    conditional_logic: dict = None,
) -> None:
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


@transaction.atomic
def drop_trigger_and_function(
    *, trigger_name: str, function_name: str, table_name: str
) -> None:
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
    with connection.cursor() as cursor:
        query = """SELECT EXISTS(
            SELECT 1 FROM information_schema.triggers
            WHERE event_object_table = %s
            AND trigger_name = %s)
        """
        cursor.execute(query, [table_name, trigger_name])
        return bool(cursor.fetchone()[0])


def check_function_exists(*, function_name: str) -> bool:
    with connection.cursor() as cursor:
        query = "SELECT EXISTS (SELECT 1 FROM pg_proc WHERE proname = %s)"
        cursor.execute(query, [function_name])
        return bool(cursor.fetchone()[0])
