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

from collections.abc import Iterable

from django.db import connection
from django.db.models import Q
from django.utils import timezone

from organization.models import TableChangeAlert


def get_active_table_alerts() -> Iterable[TableChangeAlert] | None:
    """
    Returns an iterable of active TableChangeAlert objects, or None if no alerts are active.

    An alert is considered active if it meets the following conditions:
    - The 'is_active' flag is True
    - The 'effective_date' is less than or equal to the current time, or is null
    - The 'expiration_date' is greater than or equal to the current time, or is null

    This function is decorated with the `cached_as()` decorator from the `cacheops` package. This decorator
    caches the result of this function for 60 seconds, and keeps the cache fresh by invalidating the cache
    whenever a TableChangeAlert object is saved or deleted.

    Returns:
        An iterable of active TableChangeAlert objects, or None if no alerts are active.

    Raises:
        None.

    Example usage:
        alerts = get_active_table_alerts()
        for alert in alerts:
            # Do something with the alert object
    """

    query = Q(is_active=True) & Q(effective_date__lte=timezone.now()) | Q(
        effective_date__isnull=True
    ) & Q(Q(expiration_date__gte=timezone.now()) | Q(expiration_date__isnull=True))

    active_alerts = TableChangeAlert.objects.filter(query)
    return active_alerts if active_alerts.exists() else None


def get_active_triggers() -> Iterable[tuple] | None:
    """
    Returns a list of active triggers in the PostgreSQL database.

    Raises:
        NotImplementedError: If the database engine is not PostgreSQL.

    Returns:
        List[Tuple]: A list of tuples representing the rows from the result set.
        If the query returns an empty result set, this function returns `None`.
    """

    with connection.cursor() as conn:
        conn.execute("SELECT * FROM information_schema.triggers")
        return conn.fetchall() if conn.rowcount > 0 else None
