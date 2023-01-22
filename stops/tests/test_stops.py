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
from django.utils import timezone

from movements.tests.factories import MovementFactory
from stops.tests.factories import StopFactory

pytestmark = pytest.mark.django_db


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
        self,
    ):
        """
        Test ValidationError is thrown when the status of the stop is changed to `IN_PROGRESS`
        or `COMPLETED` if the previous stop in the movement is not `COMPLETED`.
        """

        movement = MovementFactory()
        stop_1 = StopFactory(
            movement=movement, arrival_time=None, status="N", sequence=1
        )

        with pytest.raises(ValidationError) as excinfo:
            stop_2 = StopFactory(
                arrival_time=timezone.now(), status="P", sequence=2, movement=movement
            )

        assert excinfo.value.message_dict["status"] == [
            "Cannot change status to in progress or completed if previous stop is not completed. Please try again."
        ]

    def test_stop_arrival_time_before_departure_time(self):
        """
        Test ValidationError is thrown when the appointment time of the stop is before the
        previous stop appointment time.
        """

        with pytest.raises(ValidationError) as excinfo:
            StopFactory(arrival_time=None, departure_time=timezone.now())

        assert excinfo.value.message_dict["arrival_time"] == [
            "Must set arrival time before setting departure time. Please try again."
        ]
