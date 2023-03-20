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

from django.core.mail import EmailMessage

from backend.celery import app
from reports import models
from reports.services import generate_excel_report_as_file


@app.task(name="send_scheduled_report")
def send_scheduled_report(report_id: str):
    scheduled_report = models.ScheduledReport.objects.get(pk=report_id)
    if not scheduled_report.is_active:
        return

    report = scheduled_report.report
    user = scheduled_report.user

    excel_file = generate_excel_report_as_file(report)

    email = EmailMessage(
        subject=f"Your scheduled report: {report.name}",
        body=f"Hi {user.profile.first_name},\n\nAttached is your scheduled report: {report.name}.",
        from_email="reports@monta.io",
        to=[user.email],
    )

    email.attach(
        f"{report.name}.xlsx",
        excel_file.getvalue(),
        "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    )
    email.send()
