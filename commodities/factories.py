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

class CommodityFactory(factory.django.DjangoModelFactory):
    """
    Commodity factory
    """
    class Meta:
        """
        Metaclass for CommodityFactory
        """

        model = "commodities.Commodity"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("word", locale="en_US")
    hazmat = factory.SubFactory("commodities.factories.HazardousMaterialFactory")


class HazardousMaterialFactory(factory.django.DjangoModelFactory):
    """
    HazardousMaterial Factory
    """
    class Meta:
        """
        Metaclass for HazardousMaterialFactory
        """
        model = "commodities.HazardousMaterial"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("word", locale="en_US")
    hazard_class = "4.1"
