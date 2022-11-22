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

from django.test import TestCase

from equipment.factories import EquipmentFactory


class TestEquipmentType(TestCase):
    def setUp(self):
        self.equipment_type = EquipmentFactory.EquipmentTypeFactory()
        self.equipment_type_details = EquipmentFactory.EquipmentTypeDetailFactory(
            equipment_type=self.equipment_type
        )

    def test_equipment_type_creation(self):
        self.assertEqual(self.equipment_type.name, "Test Equipment Type")
        self.assertEqual(
            self.equipment_type.description, "Test Equipment Type Description"
        )

    def test_equipment_type_update(self):
        self.equipment_type.name = "Test Equipment Type Updated"
        self.equipment_type.description = "Test Equipment Type Description Updated"
        self.equipment_type.save()
        self.assertEqual(self.equipment_type.name, "Test Equipment Type Updated")
        self.assertEqual(
            self.equipment_type.description, "Test Equipment Type Description Updated"
        )
