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

from collections.abc import Iterable
from typing import Union, Tuple, Optional

from django.contrib.sessions.models import Session
from django.db import connection
from django.db.models import Q, QuerySet
from django.utils import timezone

from accounts.models import User
from organization import models


def get_active_sessions() -> Optional[QuerySet[Session]]:
    """Returns an iterable of active sessions, or None if no sessions are active.

    Returns:
        QuerySet[Sessions]: An iterable of active sessions, or None if no sessions are active.
    """

    active_sessions = Session.objects.filter(expire_date__gte=timezone.now())
    return active_sessions if active_sessions.exists() else None


def get_active_table_alerts() -> Optional[QuerySet[models.TableChangeAlert]]:
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

    query: Q = Q(is_active=True) & Q(effective_date__lte=timezone.now()) | Q(
        effective_date__isnull=True
    ) & Q(Q(expiration_date__gte=timezone.now()) | Q(expiration_date__isnull=True))

    active_alerts = models.TableChangeAlert.objects.filter(query)
    return active_alerts if active_alerts.exists() else None


def get_active_triggers() -> Union[Iterable[Tuple], None]:
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
