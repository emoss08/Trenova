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
import typing

from django.contrib.auth.models import Group
from django.core.management import call_command
from django.core.management.base import BaseCommand
from django.db import models
from django.utils.translation import gettext_lazy as _


class Command(BaseCommand):
    """
    Django Command to add Monta specific fields to Django's base auth Group model.
    """

    def handle(self, *args: typing.Any, **options: typing.Any) -> None:
        self.stdout.write("Overriding Django's base auth Group model...")
        Group.add_to_class(
            "business_unit",
            models.ForeignKey(
                "organization.BusinessUnit",
                on_delete=models.CASCADE,
                related_name="groups",
                related_query_name="group",
                verbose_name=_("Business Unit"),
            ),
        )
        Group.add_to_class(
            "organization",
            models.ForeignKey(
                "organization.Organization",
                on_delete=models.CASCADE,
                related_name="groups",
                related_query_name="group",
                verbose_name=_("Organization"),
            ),
        )

        # Remove unique constraint on name field
        Group._meta.get_field("name")._unique = False

        # re-add contraint with business_unit and organization

        call_command("makemigrations", "auth")

        call_command("migrate", "auth")

        self.stdout.write(
            self.style.SUCCESS(
                "Successfully overridden Django's base auth Group model!"
            )
        )
