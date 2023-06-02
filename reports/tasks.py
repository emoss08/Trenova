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

from celery import shared_task
from celery_singleton import Singleton
from django.core.mail import EmailMessage

from core.exceptions import ServiceException
from reports import selectors
from reports.services import generate_excel_report_as_file

if TYPE_CHECKING:
    from celery.app.task import Task


@shared_task(
    name="send_scheduled_report",
    bind=True,
    max_retries=3,
    default_retry_delay=60,
    base=Singleton,
)
def send_scheduled_report(self: "Task", *, report_id: str) -> None:
    """A Celery task that sends a scheduled report to the user who created it.

    This task generates an Excel file from the report and sends it to the user who created the report.

    Args:
        self (celery.app.task.Task): The task object.
        report_id (str): The ID of the scheduled report.

    Returns:
        None: This function does not return anything.
    """
    try:
        scheduled_report = selectors.get_scheduled_report_by_id(report_id=report_id)

        if not scheduled_report.is_active:
            return

        report = scheduled_report.custom_report
        user = scheduled_report.user

        excel_file = generate_excel_report_as_file(report)

        email = EmailMessage(
            subject=f"Your scheduled report: {report.name}",
            body=f"Hi {user.profile.first_name},\n\nAttached is your scheduled report: {report.name}.",
            from_email="reports@monta.io",
            to=[user.email],  # TODO(Wolfred): Add support for multiple recipients
        )

        email.attach(
            f"{report.name}.xlsx",
            excel_file.getvalue(),
            "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        )
        email.send()
    except ServiceException as exc:
        raise self.retry(exc=exc) from exc
