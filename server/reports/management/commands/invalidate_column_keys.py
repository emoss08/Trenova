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
import typing

import redis
from django.core.management.base import BaseCommand


class Command(BaseCommand):
    help = "Deletes all keys matching the pattern."

    def handle(self, *args: typing.Any, **options: typing.Any) -> None:
        """Handle method for invalidating column keys in Redis.

        Args:
            *args: Variable length argument list.
            **options: Arbitrary keyword arguments.

        Returns:
            None: This function does not return anything.
        """

        # Connect to Redis
        redis_client = redis.Redis(host="localhost", port=6379, db=1)

        # Define the key pattern
        pattern = ":1:allowed_fields_*"

        # Initialize cursor for SCAN
        cursor = 0

        while True:
            # Scan for keys matching the pattern
            cursor, keys = redis_client.scan(cursor, match=pattern, count=1000)
            if keys:
                # Delete keys found in this scan iteration
                redis_client.delete(*keys)
                self.stdout.write(
                    self.style.NOTICE(f"Deleted {len(keys)} keys matching the pattern.")
                )

            # Break the loop if cursor returned is 0 (end of scan)
            if cursor == 0:
                break

        self.stdout.write(
            self.style.SUCCESS("Successfully deleted all keys matching the pattern.")
        )
