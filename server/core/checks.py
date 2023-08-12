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

import locale
import shutil
import socket
from timeit import default_timer as timer

from celery.exceptions import TaskRevokedError, TimeoutError
from core.tasks import add
from django.conf import settings
from django.core.cache import BaseCache, caches
from django.core.files.base import ContentFile
from django.core.files.storage import Storage, get_storage_class
from django.db import DatabaseError, connections
from kafka.managers import KafkaManager
from redis import exceptions, from_url
from utils.types import DiskUsage, HealthStatus, HealthStatusAndTime

host = socket.gethostname()
locale.setlocale(locale.LC_ALL, "en_US.UTF-8")


def check_database() -> dict:
    """
    Check the health of the database by sending a ping request.

    Returns:
        dict: A dictionary indicating the health of the database, including the status and time taken
        to perform the check.The status will be either "online", "Offline", or "slow".

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
                return {"status": "Offline", "time": result_time}
        end = timer()
        result_time = end - start
        return {"status": "Online", "time": result_time}
    except DatabaseError:
        end = timer()
        result_time = end - start
        return {"status": "Offline", "time": result_time}


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
        result = check_cache(cache_name, default_cache)
        results.append(result)
    return results


def check_celery() -> dict[str, str | float]:
    """
    The `check_celery` method is used to check the health of a Celery instance.
    It returns a dictionary indicating the status and time taken to perform the check.
    The status will be either `'Working'` if it is Working properly, `'Offline'` if it
    is not Working properly, or `'slow'` if it is taking too long to respond.

    Returns:
        Dict[str, Union[str, float]]: The result of the Celery health check, including the status
        and time taken to perform the check.
    """
    start = timer()
    try:
        result = add.delay(3, 5)
        result.get(timeout=3)
        end = timer()
        result_time = end - start
        if result.result != 8:
            return {"status": "Offline", "time": result_time}
    except TimeoutError:
        end = timer()
        result_time = end - start
        return {"status": "Offline", "time": result_time}
    except TaskRevokedError:
        end = timer()
        result_time = end - start
        return {"status": "Offline", "time": result_time}
    return {"status": "Working", "time": result_time}


def compare_disk_usage() -> DiskUsage:
    """
    Get the total, used, and free disk space in gigabytes.

    Returns:
        Tuple[int, int, int]: A tuple containing the total, used, and free disk space in gigabytes.
    """
    total, used, free = shutil.disk_usage("/")
    total = total // (2**30)
    used = used // (2**30)
    free = free // (2**30)
    return total, used, free


def check_disk_usage(self) -> HealthStatus:
    """
    Check the disk usage and return a dictionary indicating the status and disk usage information.

    Returns:
        HealthStatus: A dictionary containing the disk usage status and the total, used, and free disk space in gigabytes.
    """
    total, used, free = self.compare_disk_usage()
    if free < 5:
        return {"status": "Critical", "total": total, "used": used, "free": free}
    return (
        {"status": "Low", "total": total, "used": used, "free": free}
        if free < 10
        else {"status": "Online", "total": total, "used": used, "free": free}
    )


def check_disk_usage_and_time() -> HealthStatusAndTime:
    """
    Check the disk usage and time taken to get the disk usage information and return a dictionary
    indicating the status, disk usage information, and time taken.

    Returns:
        HealthStatusAndTime: A dictionary containing the disk usage status, the total,
        used, and free disk space in gigabytes, and the time taken to get the disk usage information.
    """
    start = timer()
    total, used, free = compare_disk_usage()
    end = timer()
    if free < 5:
        return {
            "status": "Critical",
            "total": total,
            "used": used,
            "free": free,
            "time": end - start,
        }
    if free < 10:
        return {
            "status": "Low",
            "total": total,
            "used": used,
            "free": free,
            "time": end - start,
        }
    return (
        {
            "status": "Slow",
            "total": total,
            "used": used,
            "free": free,
            "time": end - start,
        }
        if end - start > 0.01
        else {
            "status": "Online",
            "total": total,
            "used": used,
            "free": free,
            "time": end - start,
        }
    )


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
            return {"status": "Online", "time": end - start}
    except exceptions.ConnectionError:
        end = timer()
        return {"status": "Offline", "time": end - start}
    except exceptions.TimeoutError:
        end = timer()
        return {"status": "Slow", "time": end - start}


def check_file_storage() -> dict:
    """
    Check the health of the file storage by writing and reading a file.

    Returns:
        dict: A string indicating the health of the file storage.
    """
    test_file_content = "test_content"
    test_file_name = "test_file.txt"
    storage_class: Storage = get_storage_class()()

    start = timer()
    try:
        # Write a test file
        storage_class.save(test_file_name, ContentFile(test_file_content))

        # Read the test file
        file = storage_class.open(test_file_name, "r")
        content = file.read()
        file.close()

        # Check the contents of the test file
        if content != test_file_content:
            end = timer()
            return {"status": "Corrupted", "time": end - start}
    except Exception:
        end = timer()
        return {"status": "Offline", "time": end - start}
    finally:
        # Delete the test file
        if storage_class.exists(test_file_name):
            storage_class.delete(test_file_name)

    end = timer()
    return {"status": "Online", "time": end - start}


def check_kafka() -> dict[str, str | float]:
    """Check the health of Kafka by sending a ping request.

    Returns:
        dict: A string indicating the health of Kafka, either "online", "offline", or "slow".
    """
    manager = KafkaManager()

    start = timer()
    try:
        manager.is_kafka_available()
        end = timer()
        return {"status": "Online", "time": end - start}
    except Exception:
        end = timer()
        return {"status": "Offline", "time": end - start}
