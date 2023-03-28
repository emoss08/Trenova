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

from timeit import default_timer as timer

from django.conf import settings
from django.core.cache import caches, BaseCache


class CacheBackendHealthCheck:
    """
    Class to check the health of cache backends defined in Django settings.
    """

    @staticmethod
    def check_cache(cache_name: str, default_cache: BaseCache) -> dict:
        """
        Check the health of a single cache backend.

        Returns:
            dict: A dictionary indicating the health and time of the cache backend.
        """
        start = timer()
        default_cache.set("health_check", "health_check")
        end = timer()
        result_time = end - start
        if default_cache.get("health_check") != "health_check":
            return {"name": cache_name, "status": "Offline", "time": result_time}
        if result_time > 0.01:
            return {"name": cache_name, "status": "Slow", "time": result_time}
        default_cache.delete("health_check")
        return {"name": cache_name, "status": "Online", "time": result_time}

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
            default_cache = caches[cache_name]
            result = CacheBackendHealthCheck.check_cache(cache_name, default_cache)
            results.append(result)
        return results
