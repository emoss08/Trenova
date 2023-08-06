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

from organization.models import BusinessUnit, Organization


class BusinessUnitFactory(factory.django.DjangoModelFactory):
    """
    Business Unit factory class
    """

    class Meta:
        model = "organization.BusinessUnit"

    @classmethod
    def _create(cls, model_class, *args, **kwargs):
        business_unit, created = BusinessUnit.objects.get_or_create(
            name="RNDM",
        )
        return business_unit


class OrganizationFactory(factory.django.DjangoModelFactory):
    """
    Organization factory class
    """

    class Meta:
        """
        Metaclass for OrganizationFactory
        """

        model = "organization.Organization"

    @classmethod
    def _create(cls, model_class, *args, **kwargs):
        business_unit, b_created = BusinessUnit.objects.get_or_create(name="RNDM")
        organization, o_created = Organization.objects.get_or_create(
            name="Random Company",
            scac_code="RNDM",
            business_unit=business_unit,
        )

        return organization


class DepotFactory(factory.django.DjangoModelFactory):
    """
    Depot factory class
    """

    class Meta:
        """
        Metaclass for DepotFactory
        """

        model = "organization.Depot"
        django_get_or_create = ("name",)

    name = factory.Faker("company", locale="en_US")
    organization = factory.SubFactory(OrganizationFactory)
    business_unit = factory.SubFactory(BusinessUnitFactory)


class EmailProfileFactory(factory.django.DjangoModelFactory):
    """
    Email Profile factory class
    """

    class Meta:
        """
        Metaclass for EmailProfileFactory
        """

        model = "organization.EmailProfile"
        django_get_or_create = ("name",)

    business_unit = factory.SubFactory(BusinessUnitFactory)
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

    business_unit = factory.SubFactory(BusinessUnitFactory)
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
            "organization",
            "depot",
            "email_profile",
        ),
    )
