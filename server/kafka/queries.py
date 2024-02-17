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


from typing import Any, Sequence
import psycopg
from datetime import datetime
import os
from pathlib import Path

from dotenv import load_dotenv


# Environment Variables
dotenv_path = Path(__file__).resolve().parent.parent / ".env"
print(dotenv_path)
load_dotenv(dotenv_path)

CONN_PARAMS = {
    "dbname": os.environ.get("DB_NAME"),
    "user": os.environ.get("DB_USER"),
    "password": os.environ.get("DB_PASSWORD"),
    "host": os.environ.get("DB_HOST"),
    "port": os.environ.get("DB_PORT"),
}


class DictRowFactory:
    def __init__(self, cursor):
        self.fields = [c.name for c in cursor.description]

    def __call__(self, values: Sequence[Any]) -> dict[str, Any]:
        return dict(zip(self.fields, values))


def get_active_kafka_table_change_alerts():
    try:
        conn = psycopg.connect(**CONN_PARAMS, row_factory=DictRowFactory)
        cur = conn.cursor()

        query = """
        SELECT * FROM table_change_alert
        WHERE 
            status = 'A' AND
            source = 'KAFKA' AND
            (
                (effective_date <= %s OR effective_date IS NULL) AND
                (expiration_date >= %s OR expiration_date IS NULL)
            );
        """

        # CUrrent timestamp
        now = datetime.now()

        cur.execute(query, (now, now))

        return cur.fetchall()
    except psycopg.OperationalError as e:
        print(f"Error connecting to database: {e}")
        raise e
