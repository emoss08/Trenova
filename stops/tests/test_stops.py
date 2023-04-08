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

from datetime import timedelta

import pytest
from django.core.exceptions import ValidationError
from django.urls import reverse
from django.utils import timezone
from rest_framework.response import Response
from rest_framework.test import APIClient

from location.models import Location
from movements.models import Movement
from movements.tests.factories import MovementFactory
from organization.models import Organization
from stops import models
from stops.tests.factories import StopFactory
from utils.models import StatusChoices

pytestmark = pytest.mark.django_db

STOP_LIST_URL = reverse("stops-list")


def test_list(stop: models.Stop) -> None:
    """
    Test for Stop List
    """
    assert stop is not None


def test_create(
    movement: Movement, organization: Organization, location: Location
) -> None:
    """
    Test Stop Create
    """
    stop = models.Stop.objects.create(
        organization=organization,
        movement=movement,
        location=location,
        appointment_time=timezone.now(),
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
    location: Location, organization: Organization, movement: Movement
) -> None:
    """
    Test when adding location code to a stop, that the address_line
    is filled on save
    """

    stop = models.Stop.objects.create(
        organization=organization,
        movement=movement,
        location=location,
        appointment_time=timezone.now(),
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


def test_put(
    api_client: APIClient, stop_api: Response, location: Location, movement: Movement
) -> None:
    """
    Test put Stop
    """

    response = api_client.put(
        reverse("stops-detail", kwargs={"pk": stop_api.data["id"]}),
        {
            "location": location.id,
            "movement": movement.id,
            "appointment_time": stop_api.data["appointment_time"],
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


def test_delete(api_client: APIClient, stop_api: Response) -> None:
    """
    Test Delete Stop
    """
    response = api_client.delete(
        reverse("stops-detail", kwargs={"pk": stop_api.data["id"]})
    )

    assert response.status_code == 204
    assert not response.data


def test_worker_or_equipment_in_movement_before_arrival_time() -> None:
    """
    Test ValidationError is thrown when the `arrival_time` is set in the stop, but
    Movement does not have a worker or equipment assigned to it.
    """
    movement = MovementFactory(tractor=None, primary_worker=None)

    with pytest.raises(ValidationError) as excinfo:
        StopFactory(movement=movement, arrival_time=timezone.now())

    assert excinfo.value.message_dict["arrival_time"] == [
        "Must assign worker or equipment to movement before setting arrival time. Please try again."
    ]


def test_movement_has_worker_and_equipment_if_stop_in_progress_or_completed() -> None:
    """
    Test validationError is thrown if the stop status is `IN_PROGRESS` or `COMPLETED`
    and movement does not have a `primary_worker` and `equipment` assigned to it.
    """

    movement = MovementFactory(primary_worker=None, tractor=None)

    with pytest.raises(ValidationError) as excinfo:
        StopFactory(status="P", movement=movement)

    assert excinfo.value.message_dict["status"] == [
        "Cannot change status to in progress or completed if there is no equipment or primary worker. Please try again."
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
    movement: Movement,
) -> None:
    """
    Test ValidationError is thrown when the status of the stop is changed to `IN_PROGRESS`
    or `COMPLETED` if the previous stop in the movement is not `COMPLETED`.
    """

    StopFactory(movement=movement, arrival_time=timezone.now(), status="N", sequence=1)

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
