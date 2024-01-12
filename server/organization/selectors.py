# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

from django.contrib.sessions.models import Session
from django.db import connection
from django.db.models import F, Q, QuerySet
from django.utils import timezone

from organization import models


def get_active_sessions() -> QuerySet[Session] | None:
    """Returns an iterable of active sessions, or None if no sessions are active.

    Returns:
        QuerySet[Sessions]: An iterable of active sessions, or None if no sessions are active.
    """

    active_sessions = Session.objects.filter(expire_date__gte=timezone.now())
    return active_sessions if active_sessions.exists() else None


def get_active_psql_table_change_alerts() -> QuerySet[models.TableChangeAlert]:
    """Returns a queryset of active TableChangeAlert objects.

    This function is decorated with the `cached_as()` decorator from the `cacheops` package.
    This decorator caches the result of this function for 60 seconds, and keeps the cache fresh
    by invalidating the cache whenever a TableChangeAlert object is saved or deleted.

    Returns:
        A queryset of active TableChangeAlert objects. The queryset can be empty if no alerts are active.
    """

    current_time = timezone.now()
    query = (
        Q(is_active=True)
        & (Q(effective_date__lte=current_time) | Q(effective_date__isnull=True))
        & (Q(expiration_date__gte=current_time) | Q(expiration_date__isnull=True))
    )

    return models.TableChangeAlert.objects.filter(query, source="POSTGRES")


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


def get_active_kafka_table_change_alerts() -> QuerySet[models.TableChangeAlert] | None:
    """Get Active Table Change Alerts where source is Kafka.

    Returns:
        QuerySet[models.TableChangeAlert]: A queryset of all active table change alerts.
    """
    query: Q = Q(is_active=True) & Q(effective_date__lte=timezone.now()) | Q(
        effective_date__isnull=True
    ) & (Q(expiration_date__gte=timezone.now()) | Q(expiration_date__isnull=True)) & Q(
        source="KAFKA"
    )

    active_alerts = models.TableChangeAlert.objects.filter(query)
    return active_alerts if active_alerts.exists() else None


def get_organization_feature_flags(
    *, organization_id: str
) -> QuerySet[models.OrganizationFeatureFlag] | None:
    """Get all organization feature flags.

    Returns:
        QuerySet[models.FeatureFlag]: A queryset of all organization feature flags.
    """
    return models.OrganizationFeatureFlag.objects.filter(
        organization_id=organization_id
    ).annotate(
        name=F("feature_flag__name"),
        code=F("feature_flag__code"),
        description=F("feature_flag__description"),
        beta=F("feature_flag__beta"),
        paid_only=F("feature_flag__paid_only"),
    )


def get_organization_by_id(
    *, organization_id: str
) -> QuerySet[models.Organization] | None:
    """Get organization by id.

    Returns:
        QuerySet[models.Organization]: A queryset of organization.
    """
    return models.Organization.objects.get(id__exact=organization_id)
