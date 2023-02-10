"""
COPYRIGHT 2022 MONTA

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

from timeit import default_timer as timer

from django.db import DatabaseError, connections


class DatabaseHealthCheck:
    """
    Class to check the health of the database.
    """

    @staticmethod
    def check_database() -> dict:
        """
        Check the health of the database by sending a ping request.

        Returns:
            dict: A dictionary indicating the health of the database, including the status and time taken
            to perform the check.The status will be either "online", "offline", or "slow".

        Raises:
            ConnectionError: If the database is not reachable.
            TimeoutError: If the database is slow to respond.
        """
        start = timer()
        try:
            with connections["default"].cursor() as cursor:
                cursor.execute("SELECT 1")
                if cursor.fetchone() != (1,):
                    end = timer()
                    result_time = end - start
                    return {"status": "offline", "time": result_time}
            end = timer()
            result_time = end - start
            return {"status": "working", "time": result_time}
        except DatabaseError:
            end = timer()
            result_time = end - start
            return {"status": "offline", "time": result_time}
