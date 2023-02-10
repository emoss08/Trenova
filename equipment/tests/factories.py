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

from equipment.models import EquipmentTypeDetail


class EquipmentTypeFactory(factory.django.DjangoModelFactory):
    """
    EquipmentType factory
    """

    class Meta:
        """
        Metaclass for EquipmentTypeFactory
        """

        model = "equipment.EquipmentType"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    id = factory.Faker("pystr", max_chars=50)
    description = "Test Equipment Type Description"


class EquipmentTypeDetailFactory(factory.django.DjangoModelFactory):
    """
    Factory for EquipmentTypeDetail model.
    """

    class Meta:
        """
        Metaclass for EquipmentTypeDetailFactory
        """

        model = "equipment.EquipmentTypeDetail"
        django_get_or_create = (
            "organization",
            "equipment_type",
        )

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    equipment_type = factory.SubFactory(EquipmentTypeFactory)
    equipment_class = EquipmentTypeDetail.EquipmentClassChoices.TRAILER


class EquipmentManufacturerFactory(factory.django.DjangoModelFactory):
    """
    EquipmentManufacturer factory
    """

    class Meta:
        """
        Metaclass for EquipmentManufacturerFactory
        """

        model = "equipment.EquipmentManufacturer"
        django_get_or_create = ("organization",)

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    id = factory.Faker("pystr", max_chars=50)
    description = factory.Faker("text")


class EquipmentFactory(factory.django.DjangoModelFactory):
    """
    Equipment factory
    """

    class Meta:
        """
        Metaclass for EquipmentFactory
        """

        model = "equipment.Equipment"
        django_get_or_create = (
            "organization",
            "equipment_type",
            "manufacturer",
        )

    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    id = factory.Faker("pystr", max_chars=50)
    equipment_type = factory.SubFactory(EquipmentTypeFactory)
    manufacturer = factory.SubFactory(EquipmentManufacturerFactory)
