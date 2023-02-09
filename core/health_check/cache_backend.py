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
from django.core.cache import caches


class CacheBackendHealthCheck:
    """
    Class to check the health of cache backends defined in Django settings.
    """

    @staticmethod
    def check_cache(cache_name: str, cache) -> dict:
        """
        Check the health of a single cache backend.

        Returns:
            dict: A dictionary indicating the health and time of the cache backend.
        """
        start = timer()
        cache.set("health_check", "health_check")
        end = timer()
        result_time = end - start
        if cache.get("health_check") != "health_check":
            return {"name": cache_name, "status": "offline", "time": result_time}
        if result_time > 0.01:
            return {"name": cache_name, "status": "slow", "time": result_time}
        cache.delete("health_check")
        return {"name": cache_name, "status": "working", "time": result_time}

    @staticmethod
    def check_caches_and_time() -> list:
        """
        Check the health of all cache backends defined in the `settings.CACHES` dictionary.
        Returns a list of dictionaries indicating the health and time of each cache backend.

        Returns:
            list: A list of dictionaries indicating the health and time of all cache backends.
        """
        cache_names = list(settings.CACHES.keys())
        results = []
        for cache_name in cache_names:
            cache = caches[cache_name]
            result = CacheBackendHealthCheck.check_cache(cache_name, cache)
            results.append(result)
        return results
