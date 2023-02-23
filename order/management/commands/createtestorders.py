"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""

from typing import Any, Tuple

from django.core.management import BaseCommand
from django.core.management.base import CommandParser
from django.db import transaction
from django.utils import timezone

from accounts.models import User, JobTitle
from customer.models import Customer
from equipment.models import EquipmentType
from location.models import Location
from order.models import OrderType, Order
from organization.models import Organization

DESCRIPTION = "GENERATED FROM CREATE TEST ORDERS COMMAND"


class Command(BaseCommand):
    help = "Create a number of test orders."

    def add_arguments(self, parser: CommandParser) -> None:
        parser.add_argument(
            "--organization",
            type=str,
            help="Name of the system organization.",
            default="sys",
        )

    def create_system_organization(self, organization_name: str) -> Organization:
        defaults = {"scac_code": organization_name[:4]}
        organization, created = Organization.objects.get_or_create(
            name=organization_name, defaults=defaults
        )
        return organization

    def create_user(self, organization) -> User:
        user, created = User.objects.get_or_create(
            organization=organization,
            username="walle",
            password="0&7Wj4Htiqwv3HAF1!",
            email=f"walle@{organization.name}.com",
        )
        return user

    def create_location(self, organization) -> Tuple[Location, Location]:
        defaults = {
            "description": DESCRIPTION,
            "city": "New York",
            "state": "NY",
            "zip_code": "10001",
        }
        location_1, created = Location.objects.get_or_create(
            organization=organization,
            code="test1",
            address_line_1="123 Main St",
            defaults=defaults,
        )
        location_2, created = Location.objects.get_or_create(
            organization=organization,
            code="test2",
            address_line_1="456 Main St",
            defaults=defaults,
        )
        return location_1, location_2

    def create_order_type(self, organization) -> OrderType:
        defaults = {"description": DESCRIPTION}
        order_type, created = OrderType.objects.get_or_create(
            organization=organization, name="Test Order", defaults=defaults
        )
        return order_type

    def create_customer(self, organization) -> Customer:
        defaults = {
            "is_active": True,
            "name": "Test Customer",
            "address_line_1": "123 Main St",
            "city": "New York",
            "state": "NY",
            "zip_code": "10001",
        }
        customer, created = Customer.objects.get_or_create(
            organization=organization, code="test", defaults=defaults
        )
        return customer

    def create_equipment_type(self, organization) -> EquipmentType:
        defaults = {"description": DESCRIPTION}
        equipment_type, created = EquipmentType.objects.get_or_create(
            organization=organization, name="test", defaults=defaults
        )
        return equipment_type

    def create_system_job_title(self, organization: Organization) -> JobTitle:
        defaults = {
            "description": "System job title.",
            "job_function": JobTitle.JobFunctionChoices.SYS_ADMIN,
        }
        job_title, created = JobTitle.objects.get_or_create(
            organization=organization, name="System", defaults=defaults
        )
        return job_title

    @transaction.atomic
    def handle(self, *args: Any, **options: Any) -> None:
        order_count_answer = input("How many orders would you like to create? ")
        order_count = int(order_count_answer)
        organization_name = options["organization"]
        organization = self.create_system_organization(organization_name)
        location_1, location_2 = self.create_location(organization)
        order_type = self.create_order_type(organization)
        customer = self.create_customer(organization)
        equipment_type = self.create_equipment_type(organization)
        user = self.create_user(organization)

        for _ in range(order_count):
            Order.objects.create(
                organization=organization,
                order_type=order_type,
                customer=customer,
                origin_location=location_1,
                origin_appointment=timezone.now(),
                freight_charge_amount=100,
                destination_location=location_2,
                destination_appointment=timezone.now(),
                equipment_type=equipment_type,
                entered_by=user,
                bol_number="123456789",
                comment=DESCRIPTION,
            )

        self.stdout.write(
            self.style.SUCCESS(
                f"Successfully created {order_count} orders for {organization_name}"
            )
        )
