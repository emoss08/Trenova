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

import typing

import psycopg


class DictRowFactory:
    """A factory class for creating dictionary rows from a database cursor.

    This class takes a psycopg.Cursor object as input and provides a callable
    interface to create dictionary rows from a sequence of values.

    Attributes:
        fields (List[str]): The list of field names extracted from the cursor description.


    See Also:
        https://www.psycopg.org/psycopg3/docs/advanced/rows.html#creating-new-row-factories

    Example:
        cursor = psycopg.Cursor(...)
        factory = DictRowFactory(cursor)
        row = factory([1, 'John', 25])
        # row = {'id': 1, 'name': 'John', 'age': 25}
    """

    def __init__(self, cursor: psycopg.Cursor[typing.Any]):
        """Initializes the DictRowFactory object.

        Args:
            cursor (psycopg.Cursor): The database cursor object.

        Returns:
            None: this function does not return anything.
        """
        self.fields = [c.name for c in cursor.description]

    def __call__(self, values: typing.Sequence[typing.Any]) -> dict[str, typing.Any]:
        """Creates a dictionary row from a sequence of values.

        Args:
            values (Sequence[Any]): The sequence of values.

        Returns:
            dict[str, Any]: The dictionary row where the field names are the keys and the values are the corresponding values.

        Example:
            factory = DictRowFactory(...)
            row = factory([1, 'John', 25])
            # row = {'id': 1, 'name': 'John', 'age': 25}
        """
        return dict(zip(self.fields, values))
