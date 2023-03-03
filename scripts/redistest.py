#!/usr/bin/env python3
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


def redis_get(data) -> int:
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
