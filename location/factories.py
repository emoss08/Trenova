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


class LocationCategoryFactory(factory.django.DjangoModelFactory):
    """
    LocationCategory factory
    """

    class Meta:
        """
        Metaclass for LocationCategoryFactory
        """

        model = "location.LocationCategory"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("word", locale="en_US")


class LocationFactory(factory.django.DjangoModelFactory):
    """
    Location factory
    """

    class Meta:
        """
        Metaclass for LocationFactory
        """

        model = "location.Location"
        django_get_or_create = (
            "organization",
            "location_category",
        )

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("word", locale="en_US")
    location_category = factory.SubFactory("location.factories.LocationCategoryFactory")
    address_line_1 = factory.Faker("address", locale="en_US")
    city = factory.Faker("city", locale="en_US")
    state = "NC"
    zip_code = factory.Faker("zipcode", locale="en_US")


class LocationContactFactory(factory.django.DjangoModelFactory):
    """
    LocationContact factory
    """

    class Meta:
        """
        Metaclass for LocationContactFactory
        """

        model = "location.LocationContact"
        django_get_or_create = (
            "organization",
            "location",
        )

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    location = factory.SubFactory("location.factories.LocationFactory")
    name = factory.Faker("name", locale="en_US")
    email = factory.Faker("email", locale="en_US")


class LocationCommentFactory(factory.django.DjangoModelFactory):
    """
    LocationComment factory
    """

    class Meta:
        """
        Metaclass for LocationCommentFactory
        """

        model = "location.LocationComment"
        django_get_or_create = (
            "organization",
            "location",
            "comment_type",
            "entered_by",
        )

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    location = factory.SubFactory("location.factories.LocationFactory")
    comment_type = factory.SubFactory("dispatch.factories.CommentTypeFactory")
    comment = factory.Faker("text", locale="en_US")
    entered_by = factory.SubFactory("accounts.tests.factories.UserFactory")
