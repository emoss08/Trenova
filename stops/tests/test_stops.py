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
import datetime
from datetime import timedelta

import pytest
from django.core.exceptions import ValidationError
from django.urls import reverse
from django.utils import timezone
from rest_framework.response import Response
from rest_framework.test import APIClient

from accounting.models import RevenueCode
from accounts.models import User
from customer.models import Customer
from dispatch.factories import FleetCodeFactory
from dispatch.models import DispatchControl
from equipment.models import EquipmentType
from equipment.tests.factories import TractorFactory
from location.models import Location
from movements.models import Movement
from movements.tests.factories import MovementFactory
from order.models import Order, OrderType
from organization.models import BusinessUnit, Organization
from stops import models
from stops.models import ServiceIncident
from stops.tests.factories import StopFactory
from worker.factories import WorkerFactory

pytestmark = pytest.mark.django_db

STOP_LIST_URL = reverse("stops-list")


def test_list(stop: models.Stop) -> None:
    """
    Test for Stop List
    """
    assert stop is not None


def test_create(
    movement: Movement,
    organization: Organization,
    location: Location,
    business_unit: BusinessUnit,
) -> None:
    """
    Test Stop Create
    """
    stop = models.Stop.objects.create(
        organization=organization,
        business_unit=business_unit,
        movement=movement,
        location=location,
        appointment_time_window_start=timezone.now(),
        appointment_time_window_end=timezone.now(),
        stop_type=models.StopChoices.PICKUP,
    )

    assert stop is not None
    assert stop.organization == organization
    assert stop.location == location
    assert stop.stop_type == models.StopChoices.PICKUP


def test_update(stop: models.Stop, movement: Movement, location: Location) -> None:
    """
    Test Stop Update
    """

    stop.location = location
    stop.stop_type = models.StopChoices.DELIVERY

    assert stop.location == location
    assert stop.stop_type == models.StopChoices.DELIVERY


def test_location_address(
    location: Location,
    organization: Organization,
    movement: Movement,
    business_unit: BusinessUnit,
) -> None:
    """
    Test when adding location code to a stop, that the address_line
    is filled on save
    """

    stop = models.Stop.objects.create(
        business_unit=business_unit,
        organization=organization,
        movement=movement,
        location=location,
        appointment_time_window_start=timezone.now(),
        appointment_time_window_end=timezone.now(),
        stop_type=models.StopChoices.PICKUP,
    )

    assert stop.address_line == location.get_address_combination


def test_get(api_client: APIClient) -> None:
    """
    Test get Stop
    """
    response = api_client.get(STOP_LIST_URL)
    assert response.status_code == 200


def get_by_id(api_client: APIClient, stop_api: Response) -> None:
    """
    Test get Stop by ID
    """

    response = api_client.get(f"/api/stops/{stop_api.data['id']}")

    assert response.status_code == 200
    assert response.data is not None
    assert response.data["stop_type"] == models.StopChoices.PICKUP


def test_post(
    api_client: APIClient,
    location: Location,
    movement: Movement,
    organization: Organization,
) -> None:
    """
    Test post Stop
    """
    response = api_client.post(
        STOP_LIST_URL,
        {
            "organization": organization.id,
            "location": location.id,
            "movement": movement.id,
            "sequence": 1,
            "appointment_time_window_start": timezone.now(),
            "appointment_time_window_end": timezone.now(),
            "stop_type": "SP",
        },
    )

    assert response.status_code == 201
    assert response.data is not None
    assert response.data["location"] == location.id
    assert response.data["movement"] == movement.id
    assert response.data["stop_type"] == "SP"


def test_put(
    api_client: APIClient,
    stop_api: Response,
    location: Location,
    movement: Movement,
    organization: Organization,
) -> None:
    """
    Test put Stop
    """

    response = api_client.put(
        reverse("stops-detail", kwargs={"pk": stop_api.data["id"]}),
        {
            "organization": organization.id,
            "location": location.id,
            "movement": movement.id,
            "appointment_time_window_start": stop_api.data[
                "appointment_time_window_start"
            ],
            "appointment_time_window_end": stop_api.data["appointment_time_window_end"],
            "stop_type": "SD",
        },
    )

    assert response.status_code == 200
    assert response.data is not None
    assert response.data["location"] == location.id
    assert response.data["movement"] == movement.id
    assert response.data["stop_type"] == "SD"


def test_patch(api_client: APIClient, location: Location, stop_api: Response) -> None:
    """
    Test Patch stop
    """
    response = api_client.patch(
        reverse("stops-detail", kwargs={"pk": stop_api.data["id"]}),
        {
            "location": location.id,
        },
    )

    assert response.status_code == 200
    assert response.data is not None
    assert response.data["location"] == location.id


def test_worker_or_equipment_in_movement_before_arrival_time() -> None:
    """
    Test ValidationError is thrown when the `arrival_time` is set in the stop, but
    Movement does not have a worker or equipment assigned to it.
    """
    movement = MovementFactory(tractor=None, primary_worker=None)

    with pytest.raises(ValidationError) as excinfo:
        StopFactory(movement=movement, arrival_time=timezone.now())

    assert excinfo.value.message_dict["arrival_time"] == [
        "Must assign worker or tractor to movement before setting arrival time. Please try again."
    ]


def test_movement_has_worker_and_equipment_if_stop_in_progress_or_completed() -> None:
    """
    Test validationError is thrown if the stop status is `IN_PROGRESS` or `COMPLETED`
    and movement does not have a `primary_worker` and `equipment` assigned to it.
    """

    movement = MovementFactory(primary_worker=None, tractor=None)
    stop = StopFactory(movement=movement)

    with pytest.raises(ValidationError) as excinfo:
        stop.status = "P"
        stop.clean()

    assert excinfo.value.message_dict["status"] == [
        "Cannot change status to in progress or completed if there is no tractor or primary worker. Please try again."
    ]


def test_stop_has_location_or_address_line():
    """
    Test ValidationError is thrown if the stop `location` or `address_line`
    is `None`
    """
    with pytest.raises(ValidationError) as excinfo:
        StopFactory(location=None, address_line=None)

    assert excinfo.value.message_dict["location"] == [
        "Must enter a location or address line. Please try again."
    ]


def test_cannot_change_status_to_in_progress_or_completed_if_first_stop_is_not_completed(
    order: Order,
) -> None:
    """
    Test ValidationError is thrown when the status of the stop is changed to `IN_PROGRESS`
    or `COMPLETED` if the previous stop in the movement is not `COMPLETED`.
    """

    movement = Movement.objects.filter(order=order).first()

    with pytest.raises(ValidationError) as excinfo:
        StopFactory(
            arrival_time=timezone.now(), status="P", sequence=2, movement=movement
        )

    assert excinfo.value.message_dict["status"] == [
        "Cannot change status to in progress or completed if previous stop is not completed. Please try again."
    ]


def test_arrival_time_set_before_departure_time() -> None:
    with pytest.raises(ValidationError) as excinfo:
        StopFactory(
            departure_time=timezone.now() + timedelta(hours=1),
        )

    assert excinfo.value.message_dict["arrival_time"] == [
        "Must set arrival time before setting departure time. Please try again."
    ]


def test_departure_is_after_arrival_time() -> None:
    """
    Test ValidationError is thrown if the Departure time is after the arrival time.
    """
    with pytest.raises(ValidationError) as excinfo:
        StopFactory(
            arrival_time=timezone.now() + timedelta(hours=1),
            departure_time=timezone.now(),
        )

    assert excinfo.value.message_dict["departure_time"] == [
        "Departure time must be after arrival time. Please try again."
    ]


def test_ensure_location() -> None:
    """
    Test ValidationError is thrown if the stop does not have a location or address
    line.
    """
    with pytest.raises(ValidationError) as excinfo:
        StopFactory(location=None, address_line=None)

    assert excinfo.value.message_dict["location"] == [
        "Must enter a location or address line. Please try again."
    ]


SERVICE_INCIDENT_PARAMS = [
    (
        DispatchControl.ServiceIncidentControlChoices.PICKUP,
        models.StopChoices.PICKUP,
        True,
    ),
    (
        DispatchControl.ServiceIncidentControlChoices.PICKUP,
        models.StopChoices.DELIVERY,
        False,
    ),
    (
        DispatchControl.ServiceIncidentControlChoices.DELIVERY,
        models.StopChoices.PICKUP,
        False,
    ),
    (
        DispatchControl.ServiceIncidentControlChoices.DELIVERY,
        models.StopChoices.DELIVERY,
        True,
    ),
    (
        DispatchControl.ServiceIncidentControlChoices.PICKUP_DELIVERY,
        models.StopChoices.PICKUP,
        True,
    ),
    (
        DispatchControl.ServiceIncidentControlChoices.PICKUP_DELIVERY,
        models.StopChoices.DELIVERY,
        True,
    ),
    (
        DispatchControl.ServiceIncidentControlChoices.ALL_EX_SHIPPER,
        models.StopChoices.PICKUP,
        False,
    ),
    (
        DispatchControl.ServiceIncidentControlChoices.ALL_EX_SHIPPER,
        models.StopChoices.DELIVERY,
        True,
    ),
    (
        DispatchControl.ServiceIncidentControlChoices.NEVER,
        models.StopChoices.PICKUP,
        False,
    ),
    (
        DispatchControl.ServiceIncidentControlChoices.NEVER,
        models.StopChoices.DELIVERY,
        False,
    ),
]


@pytest.mark.parametrize(
    "dispatch_control_choice, stop_choice, expected", SERVICE_INCIDENT_PARAMS
)
def test_service_incident_created(
    dispatch_control_choice,
    stop_choice,
    expected,
    organization: Organization,
    order_type: OrderType,
    revenue_code: RevenueCode,
    origin_location: Location,
    destination_location: Location,
    customer: Customer,
    equipment_type: EquipmentType,
    user: User,
    business_unit: BusinessUnit,
) -> None:
    """Test create a service incident if the stop is late.

    Args:
        organization (Organization): An organization instance.
        order_type (OrderType): An order type instance.
        revenue_code (RevenueCode): A revenue code instance.
        origin_location (Location): A location instance.
        destination_location (Location): A location instance.
        customer (Customer): A customer instance.
        equipment_type (EquipmentType): An equipment type instance.
        user (User): A user instance.
        business_unit (BusinessUnit): A business unit instance.

    Returns:
        None: This function does not return anything.
    """

    order = Order.objects.create(
        organization=organization,
        business_unit=business_unit,
        order_type=order_type,
        revenue_code=revenue_code,
        origin_location=origin_location,
        destination_location=destination_location,
        origin_appointment_window_start=timezone.now(),
        origin_appointment_window_end=timezone.now(),
        destination_appointment_window_start=timezone.now()
        + datetime.timedelta(days=2),
        destination_appointment_window_end=timezone.now() + datetime.timedelta(days=2),
        customer=customer,
        freight_charge_amount=100.00,
        equipment_type=equipment_type,
        entered_by=user,
        bol_number="1234567890",
    )
    dispatch_control: DispatchControl = order.organization.dispatch_control
    dispatch_control.record_service_incident = dispatch_control_choice
    dispatch_control.save()

    fleet_code = FleetCodeFactory()
    worker = WorkerFactory(fleet=fleet_code)
    tractor = TractorFactory(primary_worker=worker, fleet=fleet_code)
    Movement.objects.filter(order=order).update(tractor=tractor, primary_worker=worker)
    order_movement = Movement.objects.get(order=order)

    # Act: Set arrival time past the appointment window on pickup
    stop_1: models.Stop = models.Stop.objects.get(movement=order_movement, sequence=1)
    stop_1.stop_type = stop_choice
    stop_1.appointment_time_window_start = timezone.now() - datetime.timedelta(hours=1)
    stop_1.appointment_time_window_end = timezone.now() + datetime.timedelta(hours=1)
    stop_1.arrival_time = timezone.now() + datetime.timedelta(hours=3)
    stop_1.departure_time = timezone.now() + datetime.timedelta(hours=4)
    stop_1.save()

    # Act: Set arrival time past the appointment window
    stop_2: models.Stop = models.Stop.objects.get(movement=order_movement, sequence=2)
    stop_2.stop_type = models.StopChoices.DELIVERY
    stop_2.appointment_time_window_start = timezone.now() - datetime.timedelta(hours=1)
    stop_2.appointment_time_window_end = timezone.now() + datetime.timedelta(hours=1)
    stop_2.arrival_time = timezone.now() + datetime.timedelta(hours=3)
    stop_2.departure_time = timezone.now() + datetime.timedelta(hours=4)
    stop_2.save()

    assert (
        ServiceIncident.objects.filter(stop=stop_1, movement=order_movement).exists()
        == expected
    )
