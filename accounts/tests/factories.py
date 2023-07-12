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
import secrets

import factory
from django.utils import timezone

from accounts.models import Token, User
from organization.models import BusinessUnit, Organization


class JobTitleFactory(factory.django.DjangoModelFactory):
    """
    Job title factory
    """

    class Meta:
        """
        Metaclass for JobTitleFactory
        """

        model = "accounts.JobTitle"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
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

    username = "test_user"
    email = "test_user@monta.io"

    @classmethod
    def _create(cls, model_class, *args, **kwargs):
        business_unit, b_created = BusinessUnit.objects.get_or_create(name="RNDM")
        organization, o_created = Organization.objects.get_or_create(
            name="Random Company",
            scac_code="RNDM",
            business_unit=business_unit,
        )

        user, created = User.objects.get_or_create(
            username=kwargs["username"],
            password="test_password1234%",
            email=kwargs["email"],
            is_staff=True,
            is_superuser=True,
            business_unit=business_unit,
            organization=organization,
        )
        return user


class ProfileFactory(factory.django.DjangoModelFactory):
    """
    Profile Factory
    """

    class Meta:
        """
        Metaclass for ProfileFactory
        """

        model = "accounts.UserProfile"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    user = factory.SubFactory(UserFactory)
    job_title = factory.SubFactory(JobTitleFactory)
    first_name = factory.Faker("first_name")
    last_name = factory.Faker("last_name")
    address_line_1 = factory.Faker("street_address", locale="en_US")
    city = factory.Faker("city")
    state = "NC"
    zip_code = factory.Faker("zipcode")


class TokenFactory(factory.django.DjangoModelFactory):
    """
    Token factory
    """

    class Meta:
        """
        Metaclass for TokenFactory
        """

        model = "accounts.Token"

    user = factory.SubFactory(UserFactory)
    key = secrets.token_hex(20)

    @classmethod
    def _create(cls, model_class, *args, **kwargs):
        token, created = Token.objects.get_or_create(
            key=kwargs["key"], user=kwargs["user"]
        )
        return token
