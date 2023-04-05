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

import pytest
import datetime
from django.core.exceptions import ValidationError
from django.utils import timezone

from dispatch.factories import FleetCodeFactory
from equipment.tests.factories import TractorFactory
from location.factories import LocationFactory
from movements.models import Movement
from order import models
from order.selectors import get_order_stops
from order.tests.factories import OrderFactory
from stops.models import Stop
from utils.models import StatusChoices
from worker.factories import WorkerFactory

pytestmark = pytest.mark.django_db


def test_list(order):
    """
    Test Order list
    """
    assert order is not None


def test_create(
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


def test_update(order):
    """
    Test Order update
    """

    n_order = models.Order.objects.get(id=order.id)

    n_order.weight = 20_000
    n_order.pieces = 12
    n_order.bol_number = "newbolnumber"
    n_order.status = "N"
    n_order.save()

    assert n_order is not None
    assert n_order.bol_number == "newbolnumber"
    assert n_order.pieces == 12
    assert n_order.weight == 20_000


def test_first_stop_completion_puts_order_movement_in_progress(
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
    Test when the first stop in a movement is completed. The associated movement and order are both
    put in progress.
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

    fleet_code = FleetCodeFactory()
    worker = WorkerFactory(fleet=fleet_code)
    tractor = TractorFactory(primary_worker=worker, fleet=fleet_code)
    Movement.objects.filter(order=order).update(tractor=tractor, primary_worker=worker)
    order_movement = Movement.objects.get(order=order)

    # Act: Complete the first stop in the movement
    stop_1 = Stop.objects.get(movement=order_movement, sequence=1)
    stop_1.arrival_time = timezone.now() - datetime.timedelta(hours=1)
    stop_1.departure_time = timezone.now()
    stop_1.save()

    order_movement.refresh_from_db()

    # Assert: Check if the first stop is completed and the movement is in progress
    assert stop_1.status == StatusChoices.COMPLETED
    assert order_movement.status == StatusChoices.IN_PROGRESS

    # Act: Complete the second stop in the movement
    stop_2 = Stop.objects.get(movement=order_movement, sequence=2)
    stop_2.arrival_time = timezone.now() + datetime.timedelta(hours=1)
    stop_2.departure_time = timezone.now() + datetime.timedelta(hours=2)
    stop_2.save()

    order_movement.refresh_from_db()

    # Assert: Check if the second stop is completed and the movement is completed
    assert stop_2.status == StatusChoices.COMPLETED
    assert order_movement.status == StatusChoices.COMPLETED
    assert order_movement.order.status == StatusChoices.COMPLETED


def test_create_initial_movement_hook(
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


def test_get(api_client):
    """
    Test get Reason Code
    """
    response = api_client.get("/api/orders/")
    assert response.status_code == 200


def test_get_by_id(
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
    assert response.data["origin_address"] == origin_location.get_address_combination
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
    assert response.data["origin_address"] == origin_location.get_address_combination
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


def test_flat_method_requires_freight_charge_amount():
    """
    Test ValidationError is thrown when the order has `FLAT` rating method
    and the `freight_charge_amount` is None
    """
    with pytest.raises(ValidationError) as excinfo:
        OrderFactory(rate_method="F", freight_charge_amount=None)

    assert excinfo.value.message_dict["freight_charge_amount"] == [
        "Freight Rate Method is Flat but Freight Charge Amount is not set. Please try again."
    ]


def test_per_mile_requires_mileage():
    """
    Test ValidationError is thrown when the order has `PER-MILE` rating method
    and the `mileage` is None
    """
    with pytest.raises(ValidationError) as excinfo:
        OrderFactory(rate_method="PM", mileage=None)

    assert excinfo.value.message_dict["mileage"] == [
        "Rating Method 'PER-MILE' requires Mileage to be set. Please try again."
    ]


def test_order_origin_destination_location_cannot_be_the_same():
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


def test_order_revenue_code_is_enforced():
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


def test_order_commodity_is_enforced():
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


def test_order_must_be_completed_to_bill():
    """
    Test ValidationError is thrown if the order status is not `COMPLETED`
    and `ready_to_bill` is marked `TRUE`
    """
    with pytest.raises(ValidationError) as excinfo:
        OrderFactory(status="N", ready_to_bill=True)

    assert excinfo.value.message_dict["ready_to_bill"] == [
        "Cannot mark an order ready to bill if status is not 'COMPLETED'. Please try again."
    ]


def test_order_origin_location_or_address_is_required():
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


def test_order_destination_location_or_address_is_required():
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


def test_remove_orders_validation(order) -> None:
    """
    Test ValidationError is thrown if the stop in an order is being deleted,
    and order_control does not allow it.
    """

    with pytest.raises(ValidationError) as excinfo:
        for stop in get_order_stops(order=order):
            stop.delete()

    assert excinfo.value.message_dict["ref_num"] == [
        "Organization does not allow Stop removal. Please contact your administrator."
    ]
