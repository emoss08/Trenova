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

from equipment import models


class EquipmentTypeFactory(factory.django.DjangoModelFactory):
    """
    EquipmentType factory
    """

    class Meta:
        """
        Metaclass for EquipmentTypeFactory
        """

        model = models.EquipmentType

    organization = factory.SubFactory(
        "organization.factories.organization.OrganizationFactory"
    )
    name = "Test Equipment Type"
    description = "Test Equipment Type Description"


class EquipmentTypeDetailFactory(factory.django.DjangoModelFactory):
    """
    Factory for EquipmentTypeDetail model.
    """

    class Meta:
        """
        Metaclass for EquipmentTypeDetailFactory
        """

        model = models.EquipmentTypeDetail

    organization = factory.SubFactory(
        "organization.factories.organization.OrganizationFactory"
    )
    equipment_type = factory.SubFactory(EquipmentTypeFactory)
    equipment_class = models.EquipmentTypeDetail.EquipmentClassChoices.TRAILER


class EquipmentManufacturerFactory(factory.django.DjangoModelFactory):
    """
    EquipmentManufacturer factory
    """

    class Meta:
        """
        Metaclass for EquipmentManufacturerFactory
        """

        model = models.EquipmentManufacturer

    organization = factory.SubFactory(
        "organization.factories.organization.OrganizationFactory"
    )
    id = factory.Faker("name")
    description = factory.Faker("text")


class EquipmentFactory(factory.django.DjangoModelFactory):
    """
    Equipment factory
    """

    class Meta:
        """
        Metaclass for EquipmentFactory
        """

        model = models.Equipment

    organization = factory.SubFactory(
        "organization.factories.organization.OrganizationFactory"
    )
    equipment_type = factory.SubFactory(EquipmentTypeFactory)
    manufacturer = factory.SubFactory(EquipmentManufacturerFactory)
