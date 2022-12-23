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

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    is_active = True
    name = factory.Faker("job")
    description = factory.Faker("text")

    class Meta:
        """
        Metaclass for JobTitleFactory
        """

        model = "accounts.JobTitle"


class UserFactory(factory.django.DjangoModelFactory):
    """
    User factory
    """

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    username = factory.Faker("user_name")
    password = factory.Faker("password")
    email = factory.Faker("email")
    is_staff = True
    date_joined = factory.Faker("date_time", tzinfo=timezone.get_current_timezone())

    class Meta:
        """
        Meta class
        """

        model = "accounts.User"

    @factory.post_generation
    def profile(self, create, extracted, **kwargs):
        """
        Create profile
        """
        if not create:
            return

        if extracted:
            self.profile = extracted
        else:
            self.profile = ProfileFactory(user=self)


class ProfileFactory(factory.django.DjangoModelFactory):
    """
    Profile Factory
    """

    user = factory.SubFactory(UserFactory)
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    title = factory.SubFactory(JobTitleFactory)
    first_name = factory.Faker("first_name")
    last_name = factory.Faker("last_name")
    address_line_1 = factory.Faker("street_address", locale="en_US")
    city = factory.Faker("city")
    state = "NC"
    zip_code = factory.Faker("zipcode")

    class Meta:
        """
        Meta class
        """

        model = "accounts.UserProfile"


class TokenFactory(factory.django.DjangoModelFactory):
    """
    Token factory
    """

    user = factory.SubFactory(UserFactory)

    class Meta:
        """
        Meta class
        """

        model = "accounts.Token"
