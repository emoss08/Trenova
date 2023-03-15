# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  Monta is free software: you can redistribute it and/or modify                                   -
#  it under the terms of the GNU General Public License as published by                            -
#  the Free Software Foundation, either version 3 of the License, or                               -
#  (at your option) any later version.                                                             -
#                                                                                                  -
#  Monta is distributed in the hope that it will be useful,                                        -
#  but WITHOUT ANY WARRANTY; without even the implied warranty of                                  -
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                                   -
#  GNU General Public License for more details.                                                    -
#                                                                                                  -
#  You should have received a copy of the GNU General Public License                               -
#  along with Monta.  If not, see <https://www.gnu.org/licenses/>.                                 -
# --------------------------------------------------------------------------------------------------
import random
import string
from typing import Any

from django.contrib.auth import get_user_model
from django.core.management.base import BaseCommand, CommandError
from django.db import transaction

from accounts.models import JobTitle, UserProfile
from organization.models import Organization


class Command(BaseCommand):
    """
        A Django management command to create test users for a given organization.

    Prompts the user for the name of the organization and the number of users to create.
    If the organization does not exist, raises a CommandError.
    If the number of users entered is not a valid integer, raises a CommandError.
    Creates the specified number of test users with unique usernames and email addresses
    and assigns them to the specified organization. The user profiles are also created
    with the job title set to "Test User" and the first name set to "Test" and last name
    set to "User-i", where i is the user number starting from 0.

    Usage: python manage.py create_test_users <organization_name> <number_of_users>
    """

    help = "Create test users."

    @transaction.atomic
    def handle(self, *args: Any, **options: Any) -> None:
        """
        The main method that is called when the command is run.

        Prompts the user for the name of the organization and the number of users to create.
        If the organization does not exist, raises a CommandError.
        If the number of users entered is not a valid integer, raises a CommandError.
        Creates the specified number of test users with unique usernames and email addresses
        and assigns them to the specified organization. The user profiles are also created
        with the job title set to "Test User" and the first name set to "Test" and last name
        set to "User-i", where i is the user number starting from 0.

        Args:
            *args: Variable length argument list.
            **options: Variable length keyword argument list.

        Returns:
            None.

        Raises:
            CommandError: If the organization does not exist or if the number of users is not a valid integer.
        """
        system_org_answer = input(
            "What is the name of organization you'd like to add the test users to? "
        )
        number_of_users_answer = input("How many test users would you like to create? ")

        try:
            system_org = Organization.objects.get(name=system_org_answer)
        except Organization.DoesNotExist:
            raise CommandError(f"Organization {system_org_answer} does not exist.")

        try:
            number_of_users = int(number_of_users_answer)
        except ValueError:
            raise CommandError(f"{number_of_users_answer} is not a valid number.")

        User = get_user_model()

        job_title, created = JobTitle.objects.get_or_create(
            name="Test User",
            organization=system_org,
            job_function=JobTitle.JobFunctionChoices.TEST,
        )

        usernames = [f"testuser-{i}" for i in range(number_of_users)]
        existing_users = User.objects.filter(username__in=usernames).values_list(
            "username", flat=True
        )

        new_users = []
        for i in range(number_of_users):
            if usernames[i] in existing_users:
                self.stdout.write(
                    self.style.WARNING(
                        f"User {usernames[i]} already exists. Skipping..."
                    )
                )
                continue

            email = f"testuser-{i}@monta.io"
            password = "testuser".join(
                random.choices(string.ascii_uppercase + string.digits, k=8)
            )

            new_user = User(
                username=usernames[i],
                email=email,
                password=password,
                organization=system_org,
            )
            new_users.append(new_user)

        User.objects.bulk_create(new_users)

        new_profiles = []
        for i in range(number_of_users):
            if usernames[i] in existing_users:
                continue

            user = User.objects.get(username=usernames[i])
            new_profile = UserProfile(
                user=user,
                organization=system_org,
                title=job_title,
                first_name="Test",
                last_name=f"User-{i}",
                address_line_1="123 Test Street",
                city="Test City",
                state="CA",
                zip_code="12345",
                phone_number="123-456-7890",
            )
            new_profiles.append(new_profile)

        UserProfile.objects.bulk_create(new_profiles)

        self.stdout.write(
            self.style.SUCCESS(f"Successfully created {number_of_users} test users.")
        )
