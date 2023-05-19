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

from django.db import connection


class TableChoiceService:
    """A service for retrieving table and column information from a Django database.

    This service provides methods for retrieving the names of all tables in the
    database, as well as the names of the columns in a specific table.

    Attributes:
        connection (django.db.Connection): The database connection.

    """

    def __init__(self) -> None:
        """Initializes a new instance of the TableChoiceService class.

        The database engine, connection, and cursor are retrieved from the
        Django settings.

        """
        self.connection = connection

    def get_all_table_names(self) -> list[str]:
        """Gets the names of all tables in the database.

        Returns:
            list: A list of strings, where each string is the name of a table
                in the database.

        """

        names = self.connection.introspection.table_names()
        for table_name in names:
            excluded_names = (
                "silk_",
                "django",
                "auth_",
                "contenttypes_",
                "sessions_",
                "notifications_",
                "plugin",
                "auditlog",
            )
            if table_name.startswith(excluded_names):
                names.remove(table_name)
        return names

    def get_column_names(self, *, table_name: str) -> list[str]:
        """Gets the names of all columns in a specified table.

        Args:
            table_name (str): The name of the table to retrieve column names
                for.

        Returns:
            str: The name of the first column in the table.

        Notes:
            You have to pass an open cursor to the get_table_description otherwise,
            you will get an error like this:
                >>> django.db.utils.ProgrammingError: cursor already closed
            This is because the cursor is closed when the connection is closed.
        """

        return [
            column.name
            for column in self.connection.introspection.get_table_description(
                self.connection.cursor(), table_name
            )
        ]


table_names: list[str] = TableChoiceService().get_all_table_names()
TABLE_NAME_CHOICES = [(table_name, table_name) for table_name in table_names]
