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

from accounts.models import JobTitle, User
from customer.models import Customer
from equipment.models import EquipmentType
from location.models import Location
from order.models import Order, OrderType
from organization.models import Organization

DESCRIPTION = "GENERATED FROM CREATE TEST ORDERS COMMAND"


class Command(BaseCommand):
    """
    A Django management command to create a specified number of test orders.

    The `Command` class provides a set of helper methods to create the necessary objects for the orders, including
    organizations, users, locations, order types, customers, equipment types, and job titles. It then prompts the user
    for the number of orders to create, and creates that number of orders using the created objects.

    Attributes:
        help: A string representing the command help message.

    Methods:
        add_arguments: Adds command line arguments to the command parser.
        create_system_organization: Creates a system organization with the provided name.
        create_user: Creates a new user associated with the provided organization.
        create_location: Creates two locations associated with the provided organization.
        create_order_type: Creates a new order type associated with the provided organization.
        create_customer: Creates a new customer associated with the provided organization.
        create_equipment_type: Creates a new equipment type associated with the provided organization.
        create_system_job_title: Creates a new job title associated with the provided organization.
        handle: The main method to be called when the command is run.

    This class is a Django management command that creates a specified number of test orders. It provides a set of
    helper methods to create the necessary objects for the orders, including organizations, users, locations, order
    types, customers, equipment types, and job titles. It then prompts the user for the number of orders to create, and
    creates that number of orders using the created objects.

    The `Command` class expects no arguments. The `handle` method is the main method to be called when the command
    is run. It prompts the user for the number of orders to create, and creates that number of orders using the
    created objects. It then prints a success message to the console.

    The `add_arguments` method adds a command line argument to the command parser. This argument is used to specify the
    name of the system organization to be created.

    The `create_system_organization` method creates a system organization with the provided name. It returns the new
    `Organization` object.

    The `create_user` method creates a new user associated with the provided organization. It returns the new `User`
    object.

    The `create_location` method creates two locations associated with the provided organization. It returns a tuple
    containing the two new `Location` objects.

    The `create_order_type` method creates a new order type associated with the provided organization. It returns the
    new `OrderType` object.

    The `create_customer` method creates a new customer associated with the provided organization. It returns the new
    `Customer` object.

    The `create_equipment_type` method creates a new equipment type associated with the provided organization. It
    returns the new `EquipmentType` object.

    The `create_system_job_title` method creates a new job title associated with the provided organization. It returns
    the new `JobTitle` object.

    The `handle` method is the main method to be called when the command is run. It prompts the user for the number of
    orders to create, and creates that number of orders using the created objects. It then prints a success message to
    the console.
    """

    help = "Create a number of test orders."

    def add_arguments(self, parser: CommandParser) -> None:
        """
        The add_arguments method is called when the command is run and is responsible for adding
        arguments to the command that can be set by the user. In this case, it adds an argument that
        allows the user to specify the name of the system organization to create orders for.

        Args:
            parser: The CommandParser object representing the command parser to add the argument to.

        Returns:
            None: This function does not return anything.

        This method adds a single argument to the command parser named --organization. The
        argument takes a string value that represents the name of the system organization
        to create orders for. If the argument is not provided, it defaults to the string value "sys".
        """

        parser.add_argument(
            "--organization",
            type=str,
            help="Name of the system organization.",
            default="sys",
        )

    def create_system_organization(self, organization_name: str) -> Organization:
        """
        Creates a new `Organization` object with the specified name.

        Args:
            organization_name: A string representing the name of the new organization.

        Returns:
            The new `Organization` object.

        This method creates a new `Organization` object with the specified name and a default `scac_code` based on the
        first four characters of the organization name. If the organization already exists, it returns the existing
        organization instead.
        """
        defaults = {"scac_code": organization_name[:4]}
        organization, created = Organization.objects.get_or_create(
            name=organization_name, defaults=defaults
        )
        return organization

    def create_user(self, organization) -> User:
        """
        Creates a new `User` object associated with the specified organization.

        Args:
            organization: The `Organization` object to associate the new user with.

        Returns:
            The new `User` object.

        This method creates a new `User` object associated with the specified organization. The new user is assigned a
        default username, password, and email address based on the organization name. If the user already exists, it
        returns the existing user instead.
        """
        user, created = User.objects.get_or_create(
            organization=organization,
            username="walle",
            password="0&7Wj4Htiqwv3HAF1!",
            email=f"walle@{organization.name}.com",
        )
        return user

    def create_location(self, organization) -> tuple[Location, Location]:
        """
        Creates two new `Location` objects associated with the specified organization.

        Args:
            organization: The `Organization` object to associate the new locations with.

        Returns:
            A tuple containing the two new `Location` objects.

        This method creates two new `Location` objects associated with the specified organization. The new locations
        are assigned default values for their description, city, state, and zip code. If the locations already exist,
        it returns the existing locations instead.
        """
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
        """
        Creates a new `OrderType` object associated with the specified organization.

        Args:
            organization: The `Organization` object to associate the new order type with.

        Returns:
            The new `OrderType` object.

        This method creates a new `OrderType` object associated with the specified organization. The new order type
        is assigned a default description. If the order type already exists, it returns the existing order type
        instead.
        """
        defaults = {"description": DESCRIPTION}
        order_type, created = OrderType.objects.get_or_create(
            organization=organization, name="Test Order", defaults=defaults
        )
        return order_type

    def create_customer(self, organization) -> Customer:
        """
        Creates a new `Customer` object associated with the specified organization.

        Args:
            organization: The `Organization` object to associate the new customer with.

        Returns:
            The new `Customer` object.

        This method creates a new `Customer` object associated with the specified organization. The new customer is
        assigned default values for its name, address, city, state, and zip code. If the customer already exists, it
        returns the existing customer instead.
        """
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
        """
        Creates a new `EquipmentType` object associated with the specified organization.

        Args:
            organization: The `Organization` object to associate the new equipment type with.

        Returns:
            The new `EquipmentType` object.

        This method creates a new `EquipmentType` object associated with the specified organization. The new equipment
        type is assigned a default description. If the equipment type already exists, it returns the existing
        equipment type instead.
        """
        defaults = {"description": DESCRIPTION}
        equipment_type, created = EquipmentType.objects.get_or_create(
            organization=organization, name="test", defaults=defaults
        )
        return equipment_type

    def create_system_job_title(self, organization: Organization) -> JobTitle:
        """
        Creates a new `JobTitle` object associated with the specified organization.

        Args:
            organization: The `Organization` object to associate the new job title with.

        Returns:
            The new `JobTitle` object.

        This method creates a new `JobTitle` object associated with the specified organization. The new job title is
        assigned a default description and job function of "System Administrator". If the job title already exists, it
        returns the existing job title instead.
        """
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
        """
        The main method to be called when the command is run.

        This method prompts the user for the number of orders to create, creates the necessary objects for the orders,
        and creates that number of orders using the created objects. It then prints a success message to the console.
        """
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
