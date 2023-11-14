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


class EquipmentTypeFactory(factory.django.DjangoModelFactory):
    """
    EquipmentType factory
    """

    class Meta:
        """
        Metaclass for EquipmentTypeFactory
        """

        model = "equipment.EquipmentType"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("name")


class EquipmentManufacturerFactory(factory.django.DjangoModelFactory):
    """
    EquipmentManufacturer factory
    """

    class Meta:
        """
        Metaclass for EquipmentManufacturerFactory
        """

        model = "equipment.EquipmentManufacturer"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    name = factory.Faker("pystr", max_chars=50)
    description = factory.Faker("text")


class TractorFactory(factory.django.DjangoModelFactory):
    """
    Tractor factory
    """

    class Meta:
        """
        Metaclass for TractorFactory
        """

        model = "equipment.Tractor"

    business_unit = factory.SubFactory("organization.factories.BusinessUnitFactory")
    organization = factory.SubFactory("organization.factories.OrganizationFactory")
    code = factory.Faker("pystr", max_chars=50)
    equipment_type = factory.SubFactory(EquipmentTypeFactory)
    manufacturer = factory.SubFactory(EquipmentManufacturerFactory)
    fleet_code = factory.SubFactory("dispatch.factories.FleetCodeFactory")
