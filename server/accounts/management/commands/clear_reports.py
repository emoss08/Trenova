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
import logging
import os
import typing

from django.core.management import BaseCommand
from django.db import transaction
from django.utils import timezone
from datetime import timedelta

from reports import models

logger = logging.getLogger(__name__)


class Command(BaseCommand):
    help = "Django Command to clear out old reports."

    def handle(self, *args: typing.Any, **options: typing.Any) -> None:
        user_reports = models.UserReport.objects.filter(
            created=timezone.now() - timedelta(days=30)
        ).iterator()

        for user_report in user_reports:
            with transaction.atomic():
                # Delete the media file associated with the report
                if user_report.report:
                    document_path = user_report.report.path
                    document_dir = os.path.dirname(document_path)

                    # Delete the file if it exists
                    if os.path.isfile(document_path):
                        try:
                            os.remove(document_path)
                            self.stdout.write(
                                self.style.SUCCESS(f"Deleted file: {document_path}")
                            )
                        except Exception as e:
                            self.stdout.write(
                                self.style.ERROR(
                                    f"Error deleting file {document_path}: {e}"
                                )
                            )
                            continue

                    # Check if the directory is empty and delete if so
                    if not os.listdir(document_dir):
                        try:
                            os.rmdir(document_dir)
                            self.stdout.write(
                                self.style.SUCCESS(f"Deleted directory: {document_dir}")
                            )
                        except Exception as e:
                            self.stdout.write(
                                self.style.ERROR(
                                    f"Error deleting directory {document_dir}: {e}"
                                )
                            )

                # Delete the report record
                user_report.delete()
