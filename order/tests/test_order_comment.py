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
from dispatch.factories import CommentTypeFactory
from equipment.tests.factories import EquipmentTypeFactory
from location.factories import LocationFactory
from order import models
from order.tests.factories import OrderCommentFactory, OrderFactory, OrderTypeFactory
from utils.tests import ApiTest, UnitTest


class TestOrderComment(UnitTest):
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
    def order_comment(self):
        """
        Pytest Fixture for Order Comment
        """
        return OrderCommentFactory()

    @pytest.fixture()
    def comment_type(self):
        """
        Pytest Fixture for Comment Type
        """
        return CommentTypeFactory()

    def test_list(self, order_comment):
        """
        Test Order list
        """
        assert order_comment is not None

    def test_create(self, organization, user, order, comment_type):
        """
        Test Order Create
        """

        order_comment = models.OrderComment.objects.create(
            organization=organization,
            order=order,
            comment_type=comment_type,
            comment="DONT BE SAD",
            entered_by=user,
        )
        assert order_comment is not None
        assert order_comment.order == order
        assert order_comment.comment_type == comment_type
        assert order_comment.comment == "DONT BE SAD"
        assert order_comment.entered_by == user

    def test_update(self, order_comment):
        """
        Test Order update
        """

        ord_comment = models.OrderComment.objects.get(id=order_comment.id)
        ord_comment.comment = "GET GLAD"
        ord_comment.save()

        assert ord_comment is not None
        assert ord_comment.comment == "GET GLAD"


class TestOrderCommentAPI(ApiTest):
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
    def comment_type(self):
        """
        Pytest Fixture for Comment Type
        """
        return CommentTypeFactory()

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
                "freight_charge_amount": 100.00,
                "customer": f"{customer.id}",
                "equipment_type": f"{equipment_type.id}",
                "entered_by": f"{user.id}",
                "bol_number": "newbol",
            },
            format="json",
        )

    @pytest.fixture()
    def order_comment(self, order, user, comment_type, api_client):
        """
        Pytest Fixture for Order Comment
        """
        return api_client.post(
            "/api/order_comments/",
            {
                "order": f"{order.data['id']}",
                "comment_type": f"{comment_type.id}",
                "comment": "IM HAPPY YOU'RE HERE",
                "entered_by": f"{user.id}",
            },
            format="json",
        )

    def test_get_by_id(self, order_comment, order, comment_type, user, api_client):
        """
        Test get Order Comment by ID
        """
        response = api_client.get(f"/api/order_comments/{order_comment.data['id']}/")
        assert response.status_code == 200
        assert response.data is not None
        assert (
            f"{response.data['order']}" == order.data["id"]
        )  # returns UUID <UUID>, convert to F-string
        assert response.data["comment_type"] == comment_type.id
        assert response.data["comment"] == "IM HAPPY YOU'RE HERE"
        assert response.data["entered_by"] == user.id

    def test_put(self, api_client, order, order_comment, comment_type, user):
        """
        Test put Order Comment
        """
        response = api_client.put(
            f"/api/order_comments/{order_comment.data['id']}/",
            {
                "order": f"{order.data['id']}",
                "comment_type": f"{comment_type.id}",
                "comment": "BE GLAD IM HERE",
                "entered_by": f"{user.id}",
            },
            format="json",
        )

        assert response.status_code == 200
        assert response.data is not None
        assert (
            f"{response.data['order']}" == order.data["id"]
        )  # returns UUID <UUID>, convert to F-string
        assert response.data["comment_type"] == comment_type.id
        assert response.data["comment"] == "BE GLAD IM HERE"
        assert response.data["entered_by"] == user.id

    def test_patch(self, api_client, order_comment):
        """
        Test patch Order Comment
        """
        response = api_client.patch(
            f"/api/order_comments/{order_comment.data['id']}/",
            {
                "comment": "DONT BE SAD GET GLAD",
            },
            format="json",
        )

        assert response.status_code == 200
        assert response.data is not None
        assert response.data["comment"] == "DONT BE SAD GET GLAD"

    def test_delete(self, api_client, order_comment):
        """
        Test delete Order Comment
        """
        response = api_client.delete(f"/api/order_comments/{order_comment.data['id']}/")

        assert response.status_code == 204
        assert response.data is None
