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

from django.db import connection

EXCLUDED_NAMES = (
    "silk_",
    "django",
    "auth_",
    "contenttypes_",
    "sessions_",
    "notifications_",
    "plugin",
    "auditlog",
    "admin_",
    "flag",
    "waffle_",
    "edi",
    "a_group",
    "states",
    "document",
    "accounting_control",
    "billing_control",
    "doc_template_customization",
    "scheduled_report",
    "report",
    "user",
    "weekday",
    "audit",
    "auth",
    "notification_setting",
    "notification_type",
    "route_control",
    "feasibility_tool_control",
    "feature_flag",
    "google_api",
    "integration",
    "organization",
    "shipment_control",
    "formula_template",
    "dispatch_control",
    "email_control",
    "invoice_control",
    "table_change_alert",
    "tax_rate",
    "template",
    "custom_report",
)


def get_all_table_names() -> list[str]:
    """Gets the names of all tables in the database, excluding those that start
    with specified prefixes.

    Returns:
        list[str]: A list of strings, where each string is the name of a table
                   in the database, excluding tables with specified prefixes.
    """

    names = connection.introspection.table_names()

    return [
        name
        for name in names
        if not any(name.startswith(excluded) for excluded in EXCLUDED_NAMES)
    ]


def get_all_table_names_dict() -> list[dict[str, str]]:
    """Gets the names of all tables in the database, excluding those that start
    with specified prefixes.

    Returns:
        list[str]: A list of strings, where each string is the name of a table
                   in the database, excluding tables with specified prefixes.
    """

    names = connection.introspection.table_names()

    return [
        {"value": name, "label": name}
        for name in names
        if not any(name.startswith(excluded) for excluded in EXCLUDED_NAMES)
    ]


def get_column_names(*, table_name: str) -> list[str]:
    """Gets the names of all columns in a specified table.

    Args:
        table_name (str): The name of the table to retrieve column names
            for.

    Returns:
        list[str]: A list of strings, where each string is the name of a column
            in the specified table.

    Notes:
        You have to pass an open cursor to the get_table_description otherwise,
        you will get an error like this:
            django.db.utils.ProgrammingError: cursor already closed
    """

    return [
        column.name
        for column in connection.introspection.get_table_description(
            connection.cursor(), table_name
        )
    ]


table_names = get_all_table_names()

TABLE_NAME_CHOICES = [(table_name, table_name) for table_name in table_names]
