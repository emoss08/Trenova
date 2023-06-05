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

import threading
import time

import redis

redis_client = redis.Redis(host="localhost", port=6379, db=0, decode_responses=True)


def redis_set(data: dict) -> int:
    for key, value in data.items():
        redis_client.set(key, value)
    return len(data)


def redis_get(data: dict) -> int:
    count = 0
    for key in data.keys():
        val = redis_client.get(key)
        if val:
            count += 1
    return count


def run_test(test, data):
    start: float = time.time()
    ops: int = test(data)
    end: float = time.time()
    elapsed_time: float = end - start
    ops_per_sec: float = ops / elapsed_time
    total_ops: int = len(data) * 2  # assuming each test does one operation per item in data
    print(
        f"{test.__name__} elapsed time: {elapsed_time:.4f} seconds, {ops} ops, {ops_per_sec:.2f} ops/sec, {total_ops} total ops"
    )


if __name__ == "__main__":
    num = 100_000  # Change this to a larger number to see the difference
    data = {f"key{str(i)}": "val" + str(i) * 100 for i in range(num)}

    # Create threads
    t1 = threading.Thread(target=run_test, args=(redis_set, data))
    t2 = threading.Thread(target=run_test, args=(redis_get, data))

    # Start threads
    t1.start()
    t2.start()

    # Wait until threads have finished their task
    t1.join()
    t2.join()
