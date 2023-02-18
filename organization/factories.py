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


class OrganizationFactory(factory.django.DjangoModelFactory):
    """
    Organization factory class
    """

    class Meta:
        """
        Metaclass for OrganizationFactory
        """

        model = "organization.Organization"

    name = factory.Faker("company", locale="en_US")
    scac_code = "RNDM"


class DepotFactory(factory.django.DjangoModelFactory):
    """
    Depot factory class
    """

    class Meta:
        """
        Metaclass for DepotFactory
        """

        model = "organization.Depot"
        django_get_or_create = ("organization",)

    name = factory.Faker("company", locale="en_US")
    organization = factory.SubFactory(OrganizationFactory)


class EmailProfileFactory(factory.django.DjangoModelFactory):
    """
    Email Profile factory class
    """

    class Meta:
        """
        Metaclass for EmailProfileFactory
        """

        model = "organization.EmailProfile"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory(OrganizationFactory)
    name = factory.Faker("name", locale="en_US")
    email = factory.Faker("email", locale="en_US")
    protocol = factory.Faker(
        "random_element",
        elements=("SMTP", "UNENCRYPTED", "STARTTLS"),
    )
    host = "127.0.0.1"
    port = 20
    username = factory.Faker("name", locale="en_US")
    password = factory.Faker("password")


class TableChangeAlertFactory(factory.django.DjangoModelFactory):
    """
    Table Change Alert factory class
    """

    class Meta:
        """
        Metaclass for the TableChangeAlertFactory
        """

        model = "organization.TableChangeAlert"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory(OrganizationFactory)
    is_active = True
    name = factory.Faker("name", locale="en_US")
    database_action = factory.Faker(
        "random_element",
        elements=("INSERT", "UPDATE", "BOTH"),
    )
    table = factory.Faker(
        "random_element",
        elements=(
            "organization_organization",
            "organization_depot",
            "organization_emailprofile",
        ),
    )
