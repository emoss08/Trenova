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

from django.db import connections, DEFAULT_DB_ALIAS
from django.conf import settings


class TableChoiceService:
    """A service for retrieving table and column information from a Django database.

    This service provides methods for retrieving the names of all tables in the
    database, as well as the names of the columns in a specific table.

    Attributes:
        engine (str): The name of the database engine being used.
        connection (django.db.Connection): The database connection.

    """

    def __init__(self) -> None:
        """Initializes a new instance of the TableChoiceService class.

        The database engine, connection, and cursor are retrieved from the
        Django settings.

        """
        self.engine = settings.DATABASES[DEFAULT_DB_ALIAS]["ENGINE"]
        self.connection = connections[DEFAULT_DB_ALIAS]

    def get_all_table_names(self) -> List[str]:
        """Gets the names of all tables in the database.

        Returns:
            list: A list of strings, where each string is the name of a table
                in the database.

        """

        names: List[str] = self.connection.introspection.table_names()
        for table_name in names:
            excluded_names = (
                "silk_",
                "django_",
                "auth_",
                "contenttypes_",
                "sessions_",
                "notifications_",
            )
            if table_name.startswith(excluded_names):
                names.remove(table_name)
        return names

    def get_column_names(self, *, table_name: str) -> List[str]:
        """Gets the names of all columns in a specified table.

        Args:
            table_name (str): The name of the table to retrieve column names
                for.

        Returns:
            str: The name of the first column in the table.

        """

        # NOTE: You have to pass an open cursor to the get_table_description otherwise,
        # you will get an error like this:
        # django.db.utils.ProgrammingError: cursor already closed
        # This is because the cursor is closed when the connection is closed.

        return [
            column.name
            for column in self.connection.introspection.get_table_description(
                self.connection.cursor(), table_name
            )
        ]

table_names: List[str] = TableChoiceService().get_all_table_names()
TABLE_NAME_CHOICES = [(table_name, table_name) for table_name in table_names]
