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

pytestmark = pytest.mark.django_db


class TestWorkerValidation:
    """
    Test for Validating Worker Clean() Method
    """

    def test_date_of_birth(self, worker):
        """
        Test when adding a worker with date of birth that worker is over
        18 years old.
        """
        with pytest.raises(ValidationError) as excinfo:
            worker.profile.update_worker_profile(date_of_birth="2022-01-01")

        assert excinfo.value.message_dict["date_of_birth"] == [
            "Worker must be at least 18 years old to be entered. Please try again."
        ]

    def test_hazmat_endorsement(self, worker):
        """
        Test when adding a worker with hazmat endorsement and no hazmat_expiration_date,
        that validation error is thrown for date not being entered.
        """
        with pytest.raises(ValidationError) as excinfo:
            worker.profile.update_worker_profile(endorsements="H")

        assert excinfo.value.message_dict["hazmat_expiration_date"] == [
            "Hazmat expiration date is required for this endorsement. Please try again."
        ]

    def test_x_endorsement(self, worker):
        """
        Test when adding a worker with X endorsement and no hazmat_expiration_date,
        that validation error is thrown for date not being entered.
        """
        with pytest.raises(ValidationError) as excinfo:
            worker.profile.update_worker_profile(endorsements="H")

        assert excinfo.value.message_dict["hazmat_expiration_date"] == [
            "Hazmat expiration date is required for this endorsement. Please try again."
        ]

    def test_license_state(self, worker):
        """
        Test when adding a worker with a license_number, but no license_state,
        validation is thrown.
        """
        with pytest.raises(ValidationError) as excinfo:
            worker.profile.update_worker_profile(
                license_number="1234567890", license_expiration_date="2022-01-01"
            )

        assert excinfo.value.message_dict["license_state"] == [
            "You must provide license state. Please try again."
        ]

    def test_license_expiration_date(self, worker):
        """
        Test when adding a worker with a license_number,
        but no license_state, validation is thrown.
        """
        with pytest.raises(ValidationError) as excinfo:
            worker.profile.update_worker_profile(
                license_number="1234567890", license_state="CA"
            )

        assert excinfo.value.message_dict["license_expiration_date"] == [
            "You must provide license expiration date. Please try again."
        ]

    def test_worker_endorsement_choices(self, worker):
        """
        Test Worker Endorsement choices throws ValidationError
        when the passed choice is not valid.
        """
        with pytest.raises(ValidationError) as excinfo:
            worker.profile.endorsements = "invalid"
            worker.profile.full_clean()

        assert excinfo.value.message_dict["endorsements"] == [
            "Value 'invalid' is not a valid choice."
        ]

    def test_worker_sex_choices(self, worker):
        """
        Test Worker Sex choices throws ValidationError
        when the passed choice is not valid.
        """
        with pytest.raises(ValidationError) as excinfo:
            worker.profile.sex = "invalid"
            worker.profile.full_clean()

        assert excinfo.value.message_dict["sex"] == [
            "Value 'invalid' is not a valid choice."
        ]
