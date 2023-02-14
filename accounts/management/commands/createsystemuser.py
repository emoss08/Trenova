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
from typing import Any

from django.core.management import BaseCommand
from django.core.management.base import CommandParser
from django.db.transaction import atomic

from accounts.models import User
from organization.models import Organization


class Command(BaseCommand):
    """
    Django command to create system user.
    """

    help = "Create system user account and organization."

    def add_arguments(self, parser: CommandParser) -> None:
        """
        Add arguments to the command.

        Args:
            parser (CommandParser):

        Returns:
            None: None
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
            default="system",
        )
        parser.add_argument(
            "--organization",
            type=str,
            help="Name of the system organization.",
            default="sys",
        )

    @atomic(using="default")
    def handle(self, *args: Any, **options: Any) -> None:
        """
        Handle the command.

        Args:
            *args (Any): Additional arguments passed to the command
            **options (Any): Additional options passed to the command

        Returns:
            None: None
        """

        username = options["username"]
        email = options["email"]
        password = options["password"]
        organization = options["organization"]

        if not Organization.objects.filter(name=organization).exists():
            Organization.objects.create(name=organization, scac_code=organization[:4])

        # Create system user account.
        if not User.objects.filter(username=username).exists():
            User.objects.create_superuser(
                username=username,
                email=email,
                password=password,
                organization=Organization.objects.get(name=organization),
            )
        self.stdout.write(self.style.SUCCESS("System user account created!"))
