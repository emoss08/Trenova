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
import random
import string
from typing import Any

from django.core.management.base import BaseCommand, CommandParser

from location.models import Location
from organization.models import Organization
from utils.helpers import get_or_create_business_unit


class Command(BaseCommand):
    help = "Generate a number of test workers"

    def add_arguments(self, parser: CommandParser) -> None:
        parser.add_argument(
            "--count",
            type=int,
            help="Number of Workers to generate (Default is 10)",
            default=10,
        )
        parser.add_argument(
            "--organization",
            type=str,
            help="Name of the organization to use for the workers (Default is Trenova Transportation)",
            default="Trenova Transportation",
        )

    @staticmethod
    def create_system_organization(organization_name: str) -> Organization:
        organization: Organization
        created: bool
        business_unit = get_or_create_business_unit(bs_name=organization_name)

        defaults = {"scac_code": organization_name[:4], "business_unit": business_unit}
        organization, created = Organization.objects.get_or_create(
            name=organization_name,
            defaults=defaults,
        )
        return organization

    def create_locations(self, organization: Organization) -> list[Location]:
        print("Creating locations")
        locations = [
            Location(
                business_unit=organization.business_unit,
                organization=organization,
                code="".join(random.choices(string.ascii_uppercase, k=3)),
                name="".join(random.choices(string.ascii_uppercase, k=10)),
                address_line_1="Test Address",
                city="Test City",
                state="CA",
                zip_code="12345",
            )
            for _ in range(50)
        ]
        Location.objects.bulk_create(locations)

    def handle(self, *args: Any, **options: Any) -> None:
        organization_name = options["organization"]
        organization = self.create_system_organization(organization_name)
        self.create_locations(organization)
