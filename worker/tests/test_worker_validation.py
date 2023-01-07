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

from worker.factories import WorkerFactory

pytestmark = pytest.mark.django_db


class TestWorkerValidation:
    """
    Test for Validating Worker Clean() Method
    """

    @pytest.fixture()
    def worker(self):
        """
        Worker Fixture
        """
        return WorkerFactory()

    def test_date_of_birth(self, worker):
        """
        Test when adding a worker with date of birth
        that worker is over 18 years old.
        """
        with pytest.raises(
                ValidationError, match="Worker must be at least 18 years old to be entered."
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
                match="Hazmat expiration date is required for this endorsement.",
        ):
            worker.profile.update_worker_profile(
                endorsements="H"
            )

    def test_x_endorsement(self, worker):
        """
        Test when adding a worker with X
        endorsement and no hazmat_expiration_date,
        that validation error is thrown for date
        not being entered.
        """
        with pytest.raises(
                ValidationError,
                match="Hazmat expiration date is required for this endorsement.",
        ):
            worker.profile.update_worker_profile(
                endorsements="X"
            )
