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

from typing import TYPE_CHECKING

from auditlog.models import LogEntry
from django.db.models import Q

from reports import models

if TYPE_CHECKING:
    from django.db.models import QuerySet

    from utils.types import ModelUUID


def get_scheduled_report_by_id(report_id: "ModelUUID") -> models.ScheduledReport:
    """Get a scheduled report by its ID.

    Args:
        report_id (ModelUUID): The ID of the scheduled report.

    Returns:
        models.ScheduledReport: The scheduled report with the specified ID.
    """

    return models.ScheduledReport.objects.get(pk__exact=report_id)

def get_audit_logs_by_model_name(
    *, model_name: str, organization_id: "ModelUUID", app_label: str
) -> "QuerySet[LogEntry]":
    """Retrieves the audit logs for a specific model in an organization.

    This function queries LogEntry objects based on a model's name, app_label and the organization ID that it belongs to.
    It utilizes Django's Q objects to create complex lookups for retrieving corresponding LogEntry objects.
    The lookups are "model_name.lower()", "app_label.lower()" and "actor__organization_id=organization_id".

    Args:
        model_name (str): The name of the model in the application. Case-insensitive.
        organization_id (ModelUUID): The ID of the organization where the model resides.
        app_label (str): The application label where the model resides. Case-insensitive.

    Returns:
        QuerySet[LogEntry]: A QuerySet containing LogEntry objects that satisfy one or more of the conditions
        specified by the aforementioned lookups.

    Examples:
        >>> import uuid
        >>> logs = get_audit_logs_by_model_name(model_name='User', organization_id=uuid.uuid4(), app_label='auth')
    """
    return LogEntry.objects.filter(
        Q(content_type__model=model_name.lower())
        | Q(content_type__app_label=app_label.lower())
        | Q(actor__organization_id=organization_id)
    )
