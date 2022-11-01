# -*- coding: utf-8 -*-
"""
A File full of things that  I wrote for no reason other than I was bored.

I may or may not use these in the app. I just wanted to write them to see if I could.

I wrote all of these while watching techygrrrl stream resident evil 7 on twitch. I highly recommend you check her out.
https://www.twitch.tv/techygrrrl
"""

import math


def longest_common_substring(s1: str, s2: str) -> str:
    """Find the longest common substring between two strings.

    This was implemented for absolutely no reason other than I was bored.
`
    Args:
        s1 (str): The first string.
        s2 (str): The second string.

    Returns:
        str: The longest common substring.

    Typical Usage Example:
        >>> longest_common_substring("SIKE", "SIKE YOU THOUGHT")
    """
    m = [[0] * (1 + len(s2)) for i in range(1 + len(s1))]
    longest, x_longest = 0, 0
    for x in range(1, 1 + len(s1)):
        for y in range(1, 1 + len(s2)):
            if s1[x - 1] == s2[y - 1]:
                m[x][y] = m[x - 1][y - 1] + 1
                if m[x][y] > longest:
                    longest = m[x][y]
                    x_longest = x
            else:
                m[x][y] = 0
    return s1[x_longest - longest: x_longest]


def haversine(
        lat1: float, lon1: float, lat2: float, lon2: float, unit: str = "km"
) -> float:
    """Find the distance between two points on a sphere.

    This was implemented for absolutely no reason other than I was bored.

    Args:
        lat1 (float): The latitude of the first point.
        lon1 (float): The longitude of the first point.
        lat2 (float): The latitude of the second point.
        lon2 (float): The longitude of the second point.
        unit (str, optional): The unit of measurement. Defaults to "km".

    Returns:
        float: The distance between the two points.

    Typical Usage Example:
        >>> haversine(0, 0, 0, 0)
    """
    r = 6371
    dlat: float = math.radians(lat2 - lat1)
    dlon: float = math.radians(lon2 - lon1)
    a: float = (
            math.sin(dlat / 2) * math.sin(dlat / 2)
            + math.cos(math.radians(lat1))
            * math.cos(math.radians(lat2))
            * math.sin(dlon / 2)
            * math.sin(dlon / 2)
    )
    c: float = 2 * math.atan2(math.sqrt(a), math.sqrt(1 - a))
    d: float = r * c
    if unit == "km":
        return d
    elif unit == "mi":
        return d * 0.621371
    elif unit == "m":
        return d * 1000
    elif unit == "nm":
        return d * 0.539957
    else:
        return d


def rpn(expression: str) -> float:
    """Evaluate a reverse polish notation expression.

    Args:
        expression (str): The expression to evaluate.

    Returns:
        float: The result of the expression.

    Typical Usage Example:
        >>> rpn("2 3 +")
        5
    """
    stack = []
    for token in expression.split():
        if token in ["+", "-", "*", "/"]:
            arg2 = stack.pop()
            arg1 = stack.pop()
            result = eval(str(arg1) + token + str(arg2))
            stack.append(result)
        else:
            stack.append(float(token))
    return stack.pop()


def regex_match(pattern: str, string: str) -> bool:
    """Match a string against a regular expression.

    Args:
        pattern (str): The regular expression to match against.
        string (str): The string to match.

    Returns:
        bool: Whether the string matches the regular expression.

    Typical Usage Example:
        >>> regex_match("a*b", "aaaaab")
        True
    """
    if not pattern:
        return not string
    first_match = bool(string) and pattern[0] in {string[0], "."}
    if len(pattern) >= 2 and pattern[1] == "*":
        return (
                regex_match(pattern[2:], string)
                or first_match and regex_match(pattern, string[1:])
        )
    else:
        return first_match and regex_match(pattern[1:], string[1:])


def regex_match_iterative(pattern: str, string: str) -> bool:
    """Match a string against a regular expression iteratively.

    Args:
        pattern (str): The regular expression to match against.
        string (str): The string to match.

    Returns:
        bool: Whether the string matches the regular expression.

    Typical Usage Example:
        >>> regex_match_iterative("a*b", "aaaaab")
        True
    """
    if not pattern:
        return not string
    first_match = bool(string) and pattern[0] in {string[0], "."}
    if len(pattern) >= 2 and pattern[1] == "*":
        return (
                regex_match_iterative(pattern[2:], string)
                or first_match and regex_match_iterative(pattern, string[1:])
        )
    else:
        return first_match and regex_match_iterative(pattern[1:], string[1:])


def regex_match_dp(pattern: str, string: str) -> bool:
    """Match a string against a regular expression using dynamic programming.

    Args:
        pattern (str): The regular expression to match against.
        string (str): The string to match.

    Returns:
        bool: Whether the string matches the regular expression.

    Typical Usage Example:
        >>> regex_match_dp("a*b", "aaaaab")
        True
    """
    dp = [[False] * (len(string) + 1) for _ in range(len(pattern) + 1)]
    dp[-1][-1] = True
    for i in range(len(pattern) - 1, -1, -1):
        if pattern[i] == "*":
            dp[i][-1] = dp[i + 1][-1]
    for i in range(len(pattern) - 1, -1, -1):
        for j in range(len(string) - 1, -1, -1):
            first_match = pattern[i] in {string[j], "."}
            if i + 1 < len(pattern) and pattern[i + 1] == "*":
                dp[i][j] = dp[i + 2][j] or first_match and dp[i][j + 1]
            else:
                dp[i][j] = first_match and dp[i + 1][j + 1]
    return dp[0][0]


def url_parser(url: str) -> dict:
    """Parse an url and return a dictionary of the url and query string.

    You would never use this in real life, but it's a fun exercise.
    Use urllib.parse.urlparse instead.

    Args:
        url (str): The url to parse.

    Returns:
        dict: A dictionary of the url and query string.

    Typical Usage Example:
        >>> url_parser("https://www.youtube.com/watch?v=1QH8t2U6X0I?search=hello&name=world")
        {
            "url": "https://www.youtube.com/watch?v=1QH8t2U6X0I",
            "query_string": {
                "search": "hello",
                "name": "world"
            }
        }
    """
    url, query_string = url.split("?")
    query_string = query_string.split("&")
    query_string = {
        key: value
        for key, value in [
            query.split("=")
            for query in query_string
        ]
    }
    return {
        "url": url,
        "query_string": query_string
    }


# Implement a really complex binary search algorithm
def binary_search(array: list, target: int) -> int:
    """Perform a binary search on a sorted array.

    Args:
        array (list): The array to search.
        target (int): The target to search for.

    Returns:
        int: The index of the target in the array.

    Typical Usage Example:
        >>> binary_search([1, 2, 3, 4, 5], 3)
        2
    """
    left = 0
    right = len(array) - 1
    while left <= right:
        middle = (left + right) // 2
        if array[middle] == target:
            return middle
        elif array[middle] < target:
            left = middle + 1
        else:
            right = middle - 1
    return -1


def binary_search_recursive(array: list, target: int) -> int:
    """Perform a binary search on a sorted array recursively.

    Args:
        array (list): The array to search.
        target (int): The target to search for.

    Returns:
        int: The index of the target in the array.

    Typical Usage Example:
        >>> binary_search_recursive([1, 2, 3, 4, 5], 3)
        2
    """

    def _binary_search_recursive(array: list, target: int, left: int, right: int) -> int:
        """Perform a binary search on a sorted array recursively.

        Args:
            array (list): The array to search.
            target (int): The target to search for.
            left (int): The left index of the array.
            right (int): The right index of the array.

        Returns:
            int: The index of the target in the array.

        """
        if left > right:
            return -1
        middle: int = (left + right) // 2
        if array[middle] == target:
            return middle
        elif array[middle] < target:
            return _binary_search_recursive(array, target, middle + 1, right)
        else:
            return _binary_search_recursive(array, target, left, middle - 1)

    return _binary_search_recursive(array, target, 0, len(array) - 1)
