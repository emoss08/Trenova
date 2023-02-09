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

from django.conf import settings
from redis import exceptions, from_url


class RedisHealthCheck:
    """
    Class to check the health of Redis.
    """

    @staticmethod
    def check_redis() -> dict:
        """
        Check the health of Redis by sending a ping request.

        Returns:
            str: A string indicating the health of Redis, either "online", "offline", or "slow".

        Raises:
            ConnectionError: If Redis is not reachable.
            TimeoutError: If Redis is slow to respond.
        """
        redis_url = getattr(settings, "REDIS_URL", "redis://localhost:6379/0")
        start = timer()
        try:
            with from_url(redis_url) as redis:
                redis.ping()
                end = timer()
                return {"status": "working", "time": end - start}
        except exceptions.ConnectionError:
            end = timer()
            return {"status": "offline", "time": end - start}
        except exceptions.TimeoutError:
            end = timer()
            return {"status": "slow", "time": end - start}
