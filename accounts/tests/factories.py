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
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    is_active = True
    name = factory.Faker("pystr", max_chars=100)
    description = factory.Faker("text")
    job_function = "SYS_ADMIN"


class UserFactory(factory.django.DjangoModelFactory):
    """
    User factory
    """

    class Meta:
        """
        Metaclass for UserFactory
        """

        model = "accounts.User"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    username = factory.Faker("user_name")
    password = factory.Faker("password")
    email = factory.Faker("email")
    is_staff = True
    is_superuser = True
    date_joined = factory.Faker("date_time", tzinfo=timezone.get_current_timezone())

    @factory.post_generation
    def profile(self, create, extracted, **kwargs):
        """
        Create profile
        """
        if not create:
            return

        self.profile = extracted or ProfileFactory(user=self)


class ProfileFactory(factory.django.DjangoModelFactory):
    """
    Profile Factory
    """

    user = factory.SubFactory(UserFactory)
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    job_title = factory.SubFactory(JobTitleFactory)
    first_name = factory.Faker("first_name")
    last_name = factory.Faker("last_name")
    address_line_1 = factory.Faker("street_address", locale="en_US")
    city = factory.Faker("city")
    state = "NC"
    zip_code = factory.Faker("zipcode")

    class Meta:
        """
        Metaclass for ProfileFactory
        """

        model = "accounts.UserProfile"
        django_get_or_create = (
            "organization",
            "job_title",
            "user",
        )


class TokenFactory(factory.django.DjangoModelFactory):
    """
    Token factory
    """

    class Meta:
        """
        Metaclass for TokenFactory
        """

        model = "accounts.Token"
        django_get_or_create = ("user",)

    user = factory.SubFactory(UserFactory)
