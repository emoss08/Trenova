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

import pytest
from django.utils import timezone

from accounting.tests.factories import RevenueCodeFactory
from customer.factories import CustomerFactory
from equipment.tests.factories import EquipmentTypeFactory
from location.factories import LocationFactory
from order import models
from order.tests.factories import OrderFactory, OrderTypeFactory
from utils.tests import ApiTest, UnitTest


class TestOrder(UnitTest):
    """
    Class to test Order
    """

    @pytest.fixture()
    def order(self):
        """
        Pytest Fixture for Reason Code
        """
        return OrderFactory()

    @pytest.fixture()
    def order_type(self):
        """
        Pytest Fixture for Order Type
        """
        return OrderTypeFactory()

    @pytest.fixture()
    def origin_location(self):
        """
        Pytest Fixture for Origin Location
        """
        return LocationFactory()

    @pytest.fixture()
    def destination_location(self):
        """
        Pytest Fixture for Destination Location
        """
        return LocationFactory()

    @pytest.fixture()
    def revenue_code(self):
        """
        Pytest Fixture for Revenue Code
        """
        return RevenueCodeFactory()

    @pytest.fixture()
    def customer(self):
        """
        Pytest Fixture for Customer
        """
        return CustomerFactory()

    @pytest.fixture()
    def equipment_type(self):
        """
        Pytest Fixture for Equipment Type
        """
        return EquipmentTypeFactory()

    def test_list(self, order):
        """
        Test Order list
        """
        assert order is not None

    def test_create(
        self,
        organization,
        order_type,
        revenue_code,
        origin_location,
        destination_location,
        customer,
        equipment_type,
        user,
    ):
        """
        Test Order Create
        """

        order = models.Order.objects.create(
            organization=organization,
            order_type=order_type,
            revenue_code=revenue_code,
            origin_location=origin_location,
            origin_appointment=timezone.now(),
            destination_location=destination_location,
            destination_appointment=timezone.now(),
            customer=customer,
            equipment_type=equipment_type,
            entered_by=user,
            bol_number="1234567890",
        )

        assert order is not None
        assert order.order_type == order_type
        assert order.revenue_code == revenue_code
        assert order.origin_location == origin_location
        assert order.destination_location == destination_location
        assert order.customer == customer
        assert order.equipment_type == equipment_type
        assert order.entered_by == user
        assert order.bol_number == "1234567890"

    def test_update(self, order):
        """
        Test Order update
        """

        n_order = models.Order.objects.get(id=order.id)

        n_order.weight = 20_000
        n_order.pieces = 12
        n_order.bol_number = "newbolnumber"

        n_order.save()

        assert n_order is not None
        assert n_order.bol_number == "newbolnumber"
        assert n_order.pieces == 12
        assert n_order.weight == 20_000


class TestOrderAPI(ApiTest):
    """
    Test for Reason Code API
    """

    @pytest.fixture()
    def order_type(self):
        """
        Pytest Fixture for Order Type
        """
        return OrderTypeFactory()

    @pytest.fixture()
    def origin_location(self):
        """
        Pytest Fixture for Origin Location
        """
        return LocationFactory()

    @pytest.fixture()
    def destination_location(self):
        """
        Pytest Fixture for Destination Location
        """
        return LocationFactory()

    @pytest.fixture()
    def revenue_code(self):
        """
        Pytest Fixture for Revenue Code
        """
        return RevenueCodeFactory()

    @pytest.fixture()
    def customer(self):
        """
        Pytest Fixture for Customer
        """
        return CustomerFactory()

    @pytest.fixture()
    def equipment_type(self):
        """
        Pytest Fixture for Equipment Type
        """
        return EquipmentTypeFactory()

    @pytest.fixture()
    def order(
        self,
        api_client,
        organization,
        order_type,
        revenue_code,
        origin_location,
        destination_location,
        customer,
        equipment_type,
        user,
    ):
        """
        Pytest Fixture for Reason Code
        """
        return api_client.post(
            "/api/orders/",
            {
                "organization": f"{organization.id}",
                "order_type": f"{order_type.id}",
                "revenue_code": f"{revenue_code.id}",
                "origin_location": f"{origin_location.id}",
                "origin_appointment": f"{timezone.now()}",
                "destination_location": f"{destination_location.id}",
                "destination_appointment": f"{timezone.now()}",
                "customer": f"{customer.id}",
                "equipment_type": f"{equipment_type.id}",
                "entered_by": f"{user.id}",
                "bol_number": "newbol",
            },
            format="json",
        )

    def test_get(self, api_client):
        """
        Test get Reason Code
        """
        response = api_client.get("/api/orders/")
        assert response.status_code == 200

    def test_get_by_id(
        self,
        api_client,
        order,
        order_type,
        revenue_code,
        origin_location,
        destination_location,
        customer,
        equipment_type,
        user,
    ):
        """
        Test get Order by id
        """
        response = api_client.get(f"/api/orders/{order.data['id']}/")
        assert response.status_code == 200
        assert response.data["order_type"] == order_type.id
        assert response.data["revenue_code"] == revenue_code.id
        assert response.data["origin_location"] == origin_location.id
        assert (
            response.data["origin_address"] == origin_location.get_address_combination
        )
        assert response.data["destination_location"] == destination_location.id
        assert (
            response.data["destination_address"]
            == destination_location.get_address_combination
        )
        assert response.data["customer"] == customer.id
        assert response.data["equipment_type"] == equipment_type.id
        assert response.data["entered_by"] == user.id
        assert response.data["bol_number"] == "newbol"

    def test_put(
        self,
        api_client,
        order,
        origin_location,
        destination_location,
        order_type,
        revenue_code,
        customer,
        equipment_type,
        user,
    ):
        """
        Test put Order
        """
        response = api_client.put(
            f"/api/orders/{order.data['id']}/",
            {
                "origin_location": f"{origin_location.id}",
                "destination_location": f"{destination_location.id}",
                "order_type": f"{order_type.id}",
                "revenue_code": f"{revenue_code.id}",
                "origin_appointment": f"{timezone.now()}",
                "destination_appointment": f"{timezone.now()}",
                "customer": f"{customer.id}",
                "equipment_type": f"{equipment_type.id}",
                "entered_by": f"{user.id}",
                "bol_number": "anotherbol",
            },
        )

        assert response.status_code == 200
        assert response.data["origin_location"] == origin_location.id
        assert (
            response.data["origin_address"] == origin_location.get_address_combination
        )
        assert response.data["destination_location"] == destination_location.id
        assert (
            response.data["destination_address"]
            == destination_location.get_address_combination
        )
        assert response.data["order_type"] == order_type.id
        assert response.data["revenue_code"] == revenue_code.id
        assert response.data["customer"] == customer.id
        assert response.data["equipment_type"] == equipment_type.id
        assert response.data["entered_by"] == user.id
        assert response.data["bol_number"] == "anotherbol"

    def test_patch(
        self,
        api_client,
        order,
    ):
        """
        Test patch Order
        """
        response = api_client.patch(
            f"/api/orders/{order.data['id']}/",
            {
                "bol_number": "patchedbol",
            },
        )

        assert response.status_code == 200
        assert response.data["bol_number"] == "patchedbol"
