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


class ChargeTypeFactory(factory.django.DjangoModelFactory):
    """
    ChargeType factory
    """

    class Meta:
        """
        Metaclass for ChargeTypeFactory
        """

        model = "billing.ChargeType"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("word", locale="en_US")


class AccessorialChargeFactory(factory.django.DjangoModelFactory):
    """
    AccessorialCharge factory
    """

    class Meta:
        """
        Metaclass for AccessorialChargeFactory
        """

        model = "billing.AccessorialCharge"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("word", locale="en_US")
    charge_amount = 100.0


class DocumentClassificationFactory(factory.django.DjangoModelFactory):
    """
    DocumentClassification factory
    """

    class Meta:
        """
        Metaclass for DocumentClassificationFactory
        """

        model = "billing.DocumentClassification"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("word", locale="en_US")
