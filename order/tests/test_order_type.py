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
from django.urls import reverse

from order import models

pytestmark = pytest.mark.django_db


class TestOrderType:
    """
    Class to test order Type
    """

    def test_list(self, order_type):
        """
        Test Order Type list
        """
        assert order_type

    def test_create(self, organization):
        """
        Test Order Type Create
        """

        ord_type = models.OrderType.objects.create(
            organization=organization,
            is_active=True,
            name="foo bar",
            description="foo bar",
        )

        assert ord_type
        assert ord_type.is_active
        assert ord_type.name == "foo bar"
        assert ord_type.description == "foo bar"

    def test_update(self, order_type):
        """
        Test order type update
        """

        ord_type = models.OrderType.objects.get(id=order_type.id)

        ord_type.name = "Foo Bart"

        ord_type.save()

        assert ord_type
        assert ord_type.name == "Foo Bart"


class TestOrderTypeAPI:
    """
    Test for Order Type API
    """

    def test_get(self, api_client):
        """
        Test get Order Type
        """
        response = api_client.get("/api/order_types/")
        assert response.status_code == 200

    def test_get_by_id(self, api_client, order_type_api):
        """
        Test get Order Type by id
        """
        response = api_client.get(
            reverse("order-types-detail", kwargs={"pk": order_type_api.data["id"]})
        )
        assert response.status_code == 200
        assert response.data["name"] == "Foo Bar"
        assert response.data["description"] == "Foo Bar"
        assert response.data["is_active"] is True

    def test_put(self, api_client, order_type_api):
        """
        Test put Order Type
        """
        response = api_client.put(
            reverse("order-types-detail", kwargs={"pk": order_type_api.data["id"]}),
            {"name": "New Name", "description": "New Description", "is_active": False},
        )
        assert response.status_code == 200
        assert response.data["name"] == "New Name"
        assert response.data["description"] == "New Description"
        assert response.data["is_active"] is False

    def test_delete(self, api_client, order_type_api):
        """
        Test Delete order type
        """
        response = api_client.delete(
            reverse("order-types-detail", kwargs={"pk": order_type_api.data["id"]}),
        )

        assert response.status_code == 200
        assert not response.data
