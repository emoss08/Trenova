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

import factory
from django.utils import timezone


class JobTitleFactory(factory.django.DjangoModelFactory):
    """
    Job title factory
    """

    class Meta:
        """
        Metaclass for JobTitleFactory
        """

        model = "accounts.JobTitle"

    organization = factory.SubFactory(
        "organization.factories.organization.OrganizationFactory"
    )
    name = factory.Faker("job")
    description = factory.Faker("text")


class UserFactory(factory.django.DjangoModelFactory):
    """
    User factory
    """

    class Meta:
        """
        Meta class
        """

        model = "accounts.User"

    organization = factory.SubFactory(
        "organization.factories.organization.OrganizationFactory"
    )
    username = factory.Faker("user_name")
    password = factory.Faker("password")
    email = factory.Faker("email")
    is_staff = True
    date_joined = factory.Faker("date_time", tzinfo=timezone.get_current_timezone())

    @factory.post_generation
    def profile(self, create, extracted, **kwargs):
        if not create:
            return None

        if extracted:
            for profile in extracted:
                self.profile.add(profile)


class ProfileFactory(factory.django.DjangoModelFactory):
    """
    Profile Factory
    """

    class Meta:
        """
        Meta class
        """

        model = "accounts.UserProfile"

    user = factory.SubFactory(UserFactory)
    organization = factory.SubFactory(
        "organization.factories.organization.OrganizationFactory"
    )
    title = factory.SubFactory(JobTitleFactory)
    first_name = factory.Faker("first_name")
    last_name = factory.Faker("last_name")
    phone = factory.Faker("phone_number")
    city = factory.Faker("city")
    state = factory.Faker("state_abbr")
    zip_code = factory.Faker("zipcode")
