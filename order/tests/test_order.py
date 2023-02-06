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
from django.core.exceptions import ValidationError
from django.utils import timezone

from location.factories import LocationFactory
from movements.models import Movement
from order import models
from order.tests.factories import OrderFactory

pytestmark = pytest.mark.django_db


class TestOrder:
    """
    Class to test Order
    """

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
            freight_charge_amount=100.00,
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

    def test_create_initial_movement_hook(
        self,
        organization,
        order_type,
        revenue_code,
        origin_location,
        destination_location,
        customer,
        equipment_type,
        user,
    ) -> None:
        """
        Test create initial movement hook when order is created.
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
            freight_charge_amount=100.00,
            equipment_type=equipment_type,
            entered_by=user,
            bol_number="1234567890",
        )

        movement_count = Movement.objects.filter(order=order).count()

        assert movement_count == 1

class TestOrderAPI:
    """
    Test for Reason Code API
    """

    def test_get(self, api_client):
        """
        Test get Reason Code
        """
        response = api_client.get("/api/orders/")
        assert response.status_code == 200

    def test_get_by_id(
        self,
        api_client,
        order_api,
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
        response = api_client.get(f"/api/orders/{order_api.data['id']}/")
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
        order_api,
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
            f"/api/orders/{order_api.data['id']}/",
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
        order_api,
    ):
        """
        Test patch Order
        """
        response = api_client.patch(
            f"/api/orders/{order_api.data['id']}/",
            {
                "bol_number": "patchedbol",
            },
        )

        assert response.status_code == 200
        assert response.data["bol_number"] == "patchedbol"


class TestOrderValidation:
    """
    Test for Order Validation
    """

    def test_flat_method_requires_freight_charge_amount(self):
        """
        Test ValidationError is thrown when the order has `FLAT` rating method
        and the `freight_charge_amount` is None
        """
        with pytest.raises(ValidationError) as excinfo:
            OrderFactory(rate_method="F", freight_charge_amount=None)

        assert excinfo.value.message_dict["freight_charge_amount"] == [
            "Freight Rate Method is Flat but Freight Charge Amount is not set. Please try again."
        ]

    def test_per_mile_requires_mileage(self):
        """
        Test ValidationError is thrown when the order has `PER-MILE` rating method
        and the `mileage` is None
        """
        with pytest.raises(ValidationError) as excinfo:
            OrderFactory(rate_method="PM", mileage=None)

        assert excinfo.value.message_dict["mileage"] == [
            "Rating Method 'PER-MILE' requires Mileage to be set. Please try again."
        ]

    def test_order_origin_destination_location_cannot_be_the_same(self):
        """
        Test ValidationError is thrown when the order `origin_location and
        `destination_location` is the same.
        """
        order = OrderFactory()
        order.organization.order_control.enforce_origin_destination = True

        location = LocationFactory()

        with pytest.raises(ValidationError) as excinfo:
            order.origin_location = location
            order.destination_location = location
            order.save()

        assert excinfo.value.message_dict["origin_location"] == [
            "Origin and Destination locations cannot be the same. Please try again."
        ]

    def test_order_revenue_code_is_enforced(self):
        """
        Test ValidationError is thrown if the `order_control` has `enforce_rev_code`
        set as `TRUE`
        """
        order = OrderFactory()
        order.organization.order_control.enforce_rev_code = True

        with pytest.raises(ValidationError) as excinfo:
            order.revenue_code = None
            order.save()

        assert excinfo.value.message_dict["revenue_code"] == [
            "Revenue code is required. Please try again."
        ]

    def test_order_commodity_is_enforced(self):
        """
        Test ValidationError is thrown if the `order_control` has `enforce_commodity`
        set as `TRUE`
        """
        order = OrderFactory()
        order.organization.order_control.enforce_commodity = True

        with pytest.raises(ValidationError) as excinfo:
            order.revenue_code = None
            order.save()

        assert excinfo.value.message_dict["commodity"] == [
            "Commodity is required. Please try again."
        ]

    def test_order_must_be_completed_to_bill(self):
        """
        Test ValidationError is thrown if the order status is not `COMPLETED`
        and `ready_to_bill` is marked `TRUE`
        """
        with pytest.raises(ValidationError) as excinfo:
            OrderFactory(status="N", ready_to_bill=True)

        assert excinfo.value.message_dict["ready_to_bill"] == [
            "Cannot mark an order ready to bill if status is not 'COMPLETED'. Please try again."
        ]

    def test_order_origin_location_or_address_is_required(self):
        """
        Test ValidationError is thrown if the order `origin_location` and
        `origin_address` is blank
        """
        with pytest.raises(ValidationError) as excinfo:
            OrderFactory(
                origin_location=None,
                origin_address=None,
            )

        assert excinfo.value.message_dict["origin_address"] == [
            "Origin Location or Address is required. Please try again."
        ]

    def test_order_destination_location_or_address_is_required(self):
        """
        Test ValidationError is thrown if the order `destination_location` and
        `destination_address` is blank
        """
        with pytest.raises(ValidationError) as excinfo:
            OrderFactory(
                destination_location=None,
                destination_address=None,
            )

        assert excinfo.value.message_dict["destination_address"] == [
            "Destination Location or Address is required. Please try again."
        ]
