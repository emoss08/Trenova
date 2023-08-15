#  COPYRIGHT(c) 2023 MONTA
#
#  This file is part of Monta.
#
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right
#  to copy, modify, and redistribute the software, but only for non-production use or with a total
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the
#  software will be made available under version 2 or later of the GNU General Public License.
#  If you use the software in violation of this license, your rights under the license will be
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all
#  warranties and conditions. If you use this license's text or the "Business Source License" name
#  and trademark, you must comply with the Licensor's covenants, which include specifying the
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
#  Grant, and not modifying the license in any other way.

import os

from celery import Celery

os.environ.setdefault("DJANGO_SETTINGS_MODULE", "backend.settings")

app = Celery("backend")

app.config_from_object("django.conf:settings", namespace="CELERY")

app.autodiscover_tasks()

app.conf.task_routes = {
    "core.tasks.delete_audit_log_records": {
        "queue": "audit_log",
        "routing_key": "audit_log",
    },
    "organization.tasks.table_change_alerts": {
        "queue": "table_changes",
        "routing_key": "table_changes",
    },
}


@app.task(bind=True)
def debug_task(self):
    print(f"Request: {self.request!r}")
