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
from datetime import timedelta

import pytest
from django.core.exceptions import ValidationError
from django.urls import reverse
from django.utils import timezone

from movements.tests.factories import MovementFactory
from stops.tests.factories import StopFactory
from stops import models
from utils.models import StatusChoices

pytestmark = pytest.mark.django_db


class TestStop:
    """
    Class to test Stop
    """

    def test_list(self, stop) -> None:
        """
        Test for Stop List
        """
        assert stop is not None

    def test_create(self, movement, organization, location) -> None:
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

    def test_update(self, stop, movement, location) -> None:
        """
        Test Stop Update
        """

        new_stop = models.Stop.objects.get(id=stop.id)

        new_stop.location = location
        new_stop.stop_type = models.StopChoices.DELIVERY

        assert new_stop.location == location
        assert new_stop.stop_type == models.StopChoices.DELIVERY

    def test_location_address(self, location, organization, movement) -> None:
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

    def test_stop_put_in_progress_when_arrival_time(
        self, location, organization, movement
    ) -> None:
        """
        Test stop `status` field is changed to `IN_PROGRESS` when adding an `arrival_time`
        """

        stop = models.Stop.objects.create(
            organization=organization,
            movement=movement,
            location=location,
            appointment_time=timezone.now(),
            arrival_time=timezone.now(),
            stop_type=models.StopChoices.PICKUP,
        )

        assert stop.status == StatusChoices.IN_PROGRESS

    def test_stop_completed_when_arrival_and_departure_time(
        self, location, organization, movement
    ) -> None:
        """
        Test stop `status` field is changed to `COMPLETED` when `arrival_time` and `departure_time`
        """
        stop = models.Stop.objects.create(
            organization=organization,
            movement=movement,
            location=location,
            appointment_time=timezone.now(),
            arrival_time=timezone.now(),
            departure_time=timezone.now() + timedelta(minutes=1),
            stop_type=models.StopChoices.PICKUP,
        )

        assert stop.status == StatusChoices.COMPLETED


class TestStopAPI:
    """
    Class to Test Stops API.
    """

    STOP_LIST_URL = reverse("stops-list")

    def test_get(self, api_client) -> None:
        """
        Test get Stop
        """
        response = api_client.get(self.STOP_LIST_URL)
        assert response.status_code == 200

    def get_by_id(self, api_client, stop_api) -> None:
        """
        Test get Stop by ID
        """

        response = api_client.get(f"/api/stops/{stop_api.data['id']}")

        assert response.status_code == 200
        assert response.data is not None
        assert response.data["stop_type"] == models.StopChoices.PICKUP

    def test_put(self, api_client, stop_api, location, movement) -> None:
        """
        Test put Stop
        """

        response = api_client.put(
            reverse("stops-detail", kwargs={"pk": stop_api.data["id"]}),
            {
                "location": location.id,
                "movement": movement.id,
                "appointment_time": stop_api.data["appointment_time"],
                "stop_type": models.StopChoices.SPLIT_DROP,
            },
        )

        assert response.status_code == 200
        assert response.data is not None
        assert response.data["location"] == location.id
        assert response.data["movement"] == movement.id
        assert response.data["stop_type"] == models.StopChoices.SPLIT_DROP

    def test_patch(self, api_client, location, stop_api) -> None:
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

    def test_delete(self, api_client, stop_api) -> None:
        """
        Test Delete Stop
        """
        response = api_client.delete(
            reverse("stops-detail", kwargs={"pk": stop_api.data["id"]})
        )

        assert response.status_code == 200
        assert not response.data


class TestStopValidation:
    """
    Test for Stop Validation
    """

    def test_worker_or_equipment_in_movement_before_arrival_time(self):
        """
        Test ValidationError is thrown when the `arrival_time` is set in the stop, but
        Movement does not have a worker or equipment assigned to it.
        """
        movement = MovementFactory(equipment=None, primary_worker=None)

        with pytest.raises(ValidationError) as excinfo:
            StopFactory(movement=movement, arrival_time=timezone.now())

        assert excinfo.value.message_dict["arrival_time"] == [
            "Must assign worker or equipment to movement before setting arrival time. Please try again."
        ]

    def test_movement_has_worker_and_equipment_if_stop_in_progress_or_completed(self):
        """
        Test validationError is thrown if the stop status is `IN_PROGRESS` or `COMPLETED`
        and movement does not have a `primary_worker` and `equipment` assigned to it.
        """

        movement = MovementFactory(primary_worker=None, equipment=None)

        with pytest.raises(ValidationError) as excinfo:
            StopFactory(status="P", movement=movement)

        assert excinfo.value.message_dict["status"] == [
            "Cannot change status to in progress or completed if there is no equipment or primary worker. Please try again."
        ]

    def test_stop_has_location_or_address_line(self):
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
        self, movement
    ):
        """
        Test ValidationError is thrown when the status of the stop is changed to `IN_PROGRESS`
        or `COMPLETED` if the previous stop in the movement is not `COMPLETED`.
        """

        StopFactory(
            movement=movement, arrival_time=timezone.now(), status="N", sequence=1
        )

        with pytest.raises(ValidationError) as excinfo:
            StopFactory(
                arrival_time=timezone.now(), status="P", sequence=2, movement=movement
            )

        assert excinfo.value.message_dict["status"] == [
            "Cannot change status to in progress or completed if previous stop is not completed. Please try again."
        ]

    def test_arrival_time_set_before_departure_time(self) -> None:
        with pytest.raises(ValidationError) as excinfo:
            StopFactory(
                departure_time=timezone.now() + timedelta(hours=1),
            )

        assert excinfo.value.message_dict["arrival_time"] == [
            "Must set arrival time before setting departure time. Please try again."
        ]

    def test_departure_is_after_arrival_time(self) -> None:
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

    def test_ensure_location(self) -> None:
        """
        Test ValidationError is thrown if the stop does not have a location or address
        line.
        """
        with pytest.raises(ValidationError) as excinfo:
            StopFactory(location=None, address_line=None)

        assert excinfo.value.message_dict["location"] == [
            "Must enter a location or address line. Please try again."
        ]
