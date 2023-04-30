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

import re
from typing import Any

from django.core.management import BaseCommand
from django.core.management.base import CommandParser
from django.db.transaction import atomic

from accounts.models import JobTitle, User, UserProfile
from organization.models import Organization


class Command(BaseCommand):
    """
    Django command to create a system user account and organization.

    This command creates a system user account with a specified username, email address, and password, and creates
    a corresponding system organization with a specified name. If the specified username or email address already
    exists, the command exits with an error message.

    Attributes:
        help (str): The short description of the command that is displayed in the list of available commands.

    Methods:
        add_arguments(parser): Adds the command line arguments for this command.
        create_system_organization(organization_name): Creates a system organization with the specified name.
        create_system_job_title(organization): Creates a system job title for the specified organization.
        handle(*args, **options): Handles the command execution.

    """

    help = "Create system user account and organization."

    def add_arguments(self, parser: CommandParser) -> None:
        """
        Adds the command line arguments for this command.

        Args:
            parser (CommandParser): The parser object that will be used to parse the command line arguments.

        Returns:
            None
        """
        parser.add_argument(
            "--username",
            type=str,
            help="Username for the system user account.",
            default="sys",
        )

        parser.add_argument(
            "--email",
            type=str,
            help="Email address for the system user account.",
            default="system@monta.io",
        )

        parser.add_argument(
            "--password",
            type=str,
            help="Password for the system user account.",
            required=True,
        )

        parser.add_argument(
            "--organization",
            type=str,
            help="Name of the system organization.",
            default="sys",
        )

    @staticmethod
    def create_system_organization(organization_name: str) -> Organization:
        """
        Creates a system organization with the specified name.

        If an organization with the specified name already exists, this method returns that organization.
        Otherwise, it creates a new organization with the specified name and a default SCAC code, and returns it.

        Args:
            organization_name (str): The name of the system organization to create.

        Returns:
            (Organization): The newly created or existing system organization.
        """
        organization: Organization
        created: bool

        organization, created = Organization.objects.get_or_create(
            name=organization_name,
            defaults={"scac_code": organization_name[:4]},
        )
        return organization

    @staticmethod
    def create_system_job_title(organization: Organization) -> JobTitle:
        """
        Creates a system job title for the specified organization.

        If a job title with the name "System" already exists for the specified organization, this method returns
        that job title. Otherwise, it creates a new job title with the name "System" and a default description, and
        returns it.

        Args:
            organization (Organization): The organization to create the system job title for.

        Returns:
            (JobTitle): The newly created or existing system job title.
        """
        job_title: JobTitle
        created: bool

        job_title, created = JobTitle.objects.get_or_create(
            organization=organization,
            name="System",
            defaults={"description": "System job title."},
            job_function=JobTitle.JobFunctionChoices.SYS_ADMIN,
        )
        return job_title

    @atomic(using="default")
    def handle(self, *args: Any, **options: Any) -> None:
        """
        Handles the execution of the command.

        This method is called when the command is executed. It creates a new system user account with the specified
        username, email address, and password, and associates it with a system organization and system job title.
        If the specified username or email address already exists, the command exits with an error message.

        Args:
            *args (Any): Additional arguments passed to the command.
            **options (Any): Additional options passed to the command.

        Returns:
            None: This method does not return a value.
        """
        username = options["username"]
        email = options["email"]
        password = options["password"]
        organization_name = options["organization"]

        if not re.match(r"[^@]+@[^@]+\.[^@]+", email):
            self.stderr.write(self.style.ERROR("Invalid email address"))
            return

        if not re.match(r"^[a-zA-Z0-9_]+$", username):
            self.stderr.write(self.style.ERROR("Invalid username"))
            return

        organization = self.create_system_organization(organization_name)
        job_title = self.create_system_job_title(organization)

        if User.objects.filter(username=username).exists():
            self.stderr.write(self.style.ERROR(f"User {username} already exists"))
            return

        user = User.objects.create_superuser(
            username=username,
            email=email,
            password=password,
            organization=organization,
        )

        UserProfile.objects.get_or_create(
            organization=organization,
            user=user,
            job_title=job_title,
            defaults={
                "first_name": "System",
                "last_name": "User",
                "address_line_1": "1234 Main St.",
                "city": "Anytown",
                "state": "NY",
                "zip_code": "12345",
                "phone_number": "123-456-7890",
            },
        )

        self.stdout.write(self.style.SUCCESS("System user account created!"))
