# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
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
import random
import string
from datetime import timedelta
from typing import Any

from django.core.management.base import BaseCommand, CommandParser
from django.utils import timezone

from accounts.models import User
from dispatch.models import FleetCode
from organization.models import Organization
from utils.helpers import get_or_create_business_unit
from worker.models import Worker, WorkerHOS


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
    def create_fleet_code(organization: Organization) -> FleetCode:
        fleet_code, created = FleetCode.objects.get_or_create(
            code="GEN",
            description="TEST1",
            organization=organization,
            business_unit=organization.business_unit,
        )

        return fleet_code

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

    @staticmethod
    def create_user(organization: Organization) -> User:
        random_string = "".join(random.choice(string.ascii_letters) for _ in range(10))

        user, created = User.objects.get_or_create(
            organization=organization,
            business_unit=organization.business_unit,
            username=f"walle-{random_string}",
            password="0&7Wj4Htiqwv3HAF1!",
            email=f"walle@{random_string}.com",
        )
        return user

    def create_workers(
        self, organization: Organization, fleet_code: FleetCode, count: int, user: User
    ) -> list[Worker]:
        print(f"Creating {count} workers")
        workers = [
            Worker(
                business_unit=organization.business_unit,
                organization=organization,
                entered_by=user,
                manager=user,
                fleet_code=fleet_code,
                code=f"TEST{i}",
                first_name=f"Test{i}",
                last_name=f"Worker{i}",
                address_line_1="123 Test St",
                city="Test City",
                state="NC",
                zip_code="12345",
            )
            for i in range(count + 1)
        ]
        # Use bulk_create to efficiently create workers
        Worker.objects.bulk_create(workers)

        return Worker.objects.filter(
            business_unit=organization.business_unit,
            organization=organization,
            entered_by=user,
            manager=user,
            fleet_code=fleet_code,
        )[:count]

    def create_worker_hos(self, worker: Worker, organization: Organization) -> None:
        WorkerHOS.objects.create(
            business_unit=organization.business_unit,
            organization=organization,
            worker=worker,
            drive_time=39600,  # 11 hours
            seventy_hour_time=252000,  # 70 hours
            on_duty_time=50400,  # 14 hours
            off_duty_time=0,
            sleeper_berth_time=0,
            violation_time=0,
            current_status="driving",
            current_location="123 Test St, Test City, TS 12345",
            miles_driven=10,
            log_date=timezone.now().date(),
            last_reset_date=timezone.now().date() - timedelta(days=7),
        )

    def handle(self, *args: Any, **options: Any) -> None:
        organization_name = options["organization"]
        count = options["count"]

        organization = self.create_system_organization(organization_name)
        fleet_code = self.create_fleet_code(organization)
        user = self.create_user(organization)
        workers = self.create_workers(organization, fleet_code, count, user)

        for worker in workers:
            self.create_worker_hos(worker, organization)
