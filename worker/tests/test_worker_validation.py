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

from utils.tests import UnitTest
from worker import models

pytestmark = pytest.mark.django_db


class TestWorkerValidation(UnitTest):
    """
    Test for Validating Worker Clean() Method
    """

    @pytest.fixture()
    def worker(self, organization):
        """
        Worker Fixture
        """
        worker = models.Worker.objects.create(
            organization=organization,
            code="Test",
            is_active=True,
            worker_type="EMPLOYEE",
            first_name="Test",
            last_name="Worker",
            address_line_1="Test Address Line 1",
            address_line_2="Unit C",
            city="Sacramento",
            state="CA",
            zip_code="12345",
        )
        return worker

    def test_date_of_birth(self, worker):
        """
        Test when adding a worker with date of birth
        that worker is over 18 years old.
        """
        with pytest.raises(
            ValidationError,
            match="Worker must be at least 18 years old to be entered. Please try again.",
        ):
            worker.profile.update_worker_profile(date_of_birth="2022-01-01")

    def test_hazmat_endorsement(self, worker):
        """
        Test when adding a worker with hazmat
        endorsement and no hazmat_expiration_date,
        that validation error is thrown for date
        not being entered.
        """
        with pytest.raises(
            ValidationError,
            match="Hazmat expiration date is required for this endorsement. Please try again.",
        ):
            worker.profile.update_worker_profile(endorsements="H")

    def test_x_endorsement(self, worker):
        """
        Test when adding a worker with X
        endorsement and no hazmat_expiration_date,
        that validation error is thrown for date
        not being entered.
        """
        with pytest.raises(
            ValidationError,
            match="Hazmat expiration date is required for this endorsement. Please try again.",
        ):
            worker.profile.update_worker_profile(endorsements="H")

    def test_license_state(self, worker):
        """
        Test when adding a worker with a license_number,
        but no license_state, validation is thrown.
        """
        with pytest.raises(
            ValidationError, match="You must provide license state. Please try again."
        ):
            worker.profile.update_worker_profile(
                license_number="1234567890", license_expiration_date="2022-01-01"
            )

    def test_license_expiration_date(self, worker):
        """
        Test when adding a worker with a license_number,
        but no license_state, validation is thrown.
        """
        with pytest.raises(
            ValidationError,
            match="You must provide license expiration date. Please try again.",
        ):
            worker.profile.update_worker_profile(
                license_number="1234567890", license_state="CA"
            )
