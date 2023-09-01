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
import random
import string
from typing import Any

from django.contrib.auth import get_user_model
from django.core.management.base import BaseCommand, CommandError
from django.db import transaction
from rich.progress import Progress

from accounts.models import JobTitle, UserProfile
from organization.models import Organization
from utils.helpers import get_or_create_business_unit


class Command(BaseCommand):
    """A Django management command to create test users for a given organization.

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

        business_unit = get_or_create_business_unit(bs_name="Monta Transportation")

        system_org_answer = input(
            "What is the name of organization you'd like to add the test users to? (Scac Code) "
        )
        number_of_users_answer = input("How many test users would you like to create? ")

        try:
            system_org = Organization.objects.get(scac_code=system_org_answer)
        except Organization.DoesNotExist as e:
            raise CommandError(
                f"Organization {system_org_answer} does not exist."
            ) from e

        try:
            number_of_users = int(number_of_users_answer)
        except ValueError as e:
            raise CommandError(
                f"{number_of_users_answer} is not a valid number."
            ) from e

        User = get_user_model()

        job_title, created = JobTitle.objects.get_or_create(
            name="Test User",
            organization=system_org,
            business_unit=business_unit,
            job_function=JobTitle.JobFunctionChoices.TEST,
        )

        usernames = [f"testuser-{i}" for i in range(number_of_users)]
        existing_users = User.objects.filter(username__in=usernames).values_list(
            "username", flat=True
        )

        new_users = []
        with Progress() as progress:
            task = progress.add_task(
                "[cyan]Creating test users...", total=number_of_users
            )

            for i in range(number_of_users):
                if usernames[i] in existing_users:
                    progress.update(task, advance=1)
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
                    business_unit=business_unit,
                )
                new_users.append(new_user)
                progress.update(task, advance=1)

        User.objects.bulk_create(new_users)

        new_profiles = []
        with Progress() as progress:
            task = progress.add_task(
                "[cyan]Creating user profiles...", total=number_of_users
            )

            for i in range(number_of_users):
                if usernames[i] in existing_users:
                    progress.update(task, advance=1)
                    continue

                user = User.objects.get(username=usernames[i])
                new_profile = UserProfile(
                    user=user,
                    organization=system_org,
                    business_unit=business_unit,
                    job_title=job_title,
                    first_name="Test",
                    last_name=f"User-{i}",
                    address_line_1="123 Test Street",
                    city="Test City",
                    state="CA",
                    zip_code="12345",
                    phone_number="123-456-7890",
                )
                new_profiles.append(new_profile)
                progress.update(task, advance=1)

        UserProfile.objects.bulk_create(new_profiles)

        self.stdout.write(
            self.style.SUCCESS(f"Successfully created {number_of_users} test users.")
        )
