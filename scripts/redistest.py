#!/usr/bin/env python3

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

import time

import redis

redis_client = redis.Redis(host="localhost", port=6379, db=0, decode_responses=True)


def redis_set(data: dict) -> int:
    """
    Sets key-value pairs in Redis.

    Args:
        data: A dictionary of key-value pairs to be set in Redis.

    Returns:
        The number of key-value pairs set in Redis.
    """
    for key, value in data.items():
        redis_client.set(key, value)
    return len(data)


def redis_get(data: dict) -> int:
    """
    Retrieves values for given keys from Redis.

    Args:
        data: A dictionary containing keys for which to retrieve values from Redis.

    Returns:
        The number of values retrieved from Redis.
    """
    count = 0
    for key in data.keys():
        val = redis_client.get(key)
        if val:
            count += 1
    return count


def run_tests(num: int, tests: list) -> None:
    """
    Runs a set of tests on Redis.

    Args:
        num: An integer specifying the number of key-value pairs to be set in Redis.
        tests: A list of functions to be tested on Redis.

    Returns:
        None.
    """
    data = {f"key{str(i)}": "val" + str(i) * 100 for i in range(num)}
    for test in tests:
        start: float = time.time()
        ops: int = test(data)
        end: float = time.time()
        elapsed_time: float = end - start
        ops_per_sec: float = ops / elapsed_time
        total_ops: int = num * len(tests)
        print(
            f"{test.__name__} elapsed time: {elapsed_time:.4f} seconds, {ops} ops, {ops_per_sec:.2f} ops/sec, {total_ops} total ops"
        )


if __name__ == "__main__":
    num = 1_000  # Change this to a larger number to see the difference
    tests = [redis_set, redis_get]
    run_tests(num, tests)
