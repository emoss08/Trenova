# """
# COPYRIGHT 2022 MONTA
#
# This file is part of Monta.
#
# Monta is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# Monta is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with Monta.  If not, see <https://www.gnu.org/licenses/>.
# """
#
# import pytest
#
# from equipment.tests.factories import EquipmentFactory
# from movements.tests.factories import MovementFactory
# from movements import models
# from order.tests.factories import OrderFactory
# from utils.tests import ApiTest, UnitTest
# from worker.factories import WorkerFactory
#
#
# class TestMovement(UnitTest):
#     """
#     Class to test Movement
#     """
#
#     @pytest.fixture()
#     def movement(self):
#         """
#         Pytest Fixture for Movement
#         """
#         return MovementFactory()
#
#     @pytest.fixture()
#     def worker(self):
#         """
#         Pytest Fixture for Worker
#         """
#         return WorkerFactory()
#
#     @pytest.fixture()
#     def equipment(self):
#         """
#         Pytest fixture for Equipment
#         """
#         return EquipmentFactory()
#
#     @pytest.fixture()
#     def order(self):
#         """
#         Pytest fixture for Order
#         """
#         return OrderFactory()
#
#     def test_list(self, movement):
#         """
#         Test for Movement List
#         """
#         print(movement)
#         assert movement is not None
#
#     def test_create(self, worker, equipment, organization, order):
#         """
#         Test Movement Create
#         """
#
#         movement = models.Movement.objects.create(
#             organization=organization,
#             order=order,
#             equipment=equipment,
#             primary_worker=worker,
#         )
#
#         assert movement is not None
#         assert movement.order == order
#         assert movement.equipment == equipment
#         assert movement.primary_worker == worker
#
#     def test_update(self, movement, equipment):
#         """
#         Test Movement Update
#         """
#
#         add_movement = models.Movement.objects.get(id=movement.id)
#
#         add_movement.equipment = equipment
#         add_movement.save()
#
#         assert add_movement is not None
#         assert add_movement.equipment == equipment
#
#
# class TestMovementAPI(ApiTest):
#     """
#     Class to test Movement API
#     """
#
#     @pytest.fixture()
#     def worker(self):
#         """
#         Pytest Fixture for Worker
#         """
#         return WorkerFactory()
#
#     @pytest.fixture()
#     def equipment(self):
#         """
#         Pytest fixture for Equipment
#         """
#         return EquipmentFactory()
#
#     @pytest.fixture()
#     def order(self):
#         """
#         Pytest fixture for Order
#         """
#         return OrderFactory()
#
#     @pytest.fixture()
#     def movement(self, api_client, organization, order, equipment, worker):
#         """
#         Movement Factory
#         """
#         return api_client.post(
#             "/api/movements/",
#             {
#                 "organization": f"{organization.id}",
#                 "order": f"{order.id}",
#                 "primary_worker": f"{worker.id}",
#                 "equipment": f"{equipment.id}"
#             }
#         )
#
#     def test_get(self, api_client):
#         """
#         Test get Movement
#         """
#         response = api_client.get("/api/movements/")
#         assert response.status_code == 200
#
#
#     def test_get_by_id(self, api_client, movement, order, worker, equipment):
#         """
#         Test get Movement by ID
#         """
#
#         response = api_client.get(
#             f"/api/movements/{movement.data['id']}/"
#         )
#
#         assert response.status_code == 200
#         assert response.data is not None
#         assert response.data['order'] == order.id
#         assert response.data['primary_worker'] == worker.id
#         assert response.data['equipment'] == equipment.id
#
