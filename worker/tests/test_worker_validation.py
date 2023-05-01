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
from django.core.exceptions import ValidationError

from worker.models import Worker

pytestmark = pytest.mark.django_db


def test_date_of_birth(worker: Worker) -> None:
    """
    Test when adding a worker with date of birth that worker is over
    18 years old.
    """
    with pytest.raises(ValidationError) as excinfo:
        worker.profile.update_worker_profile(date_of_birth="2022-01-01")

    assert excinfo.value.message_dict["date_of_birth"] == [
        "Worker must be at least 18 years old to be entered. Please try again."
    ]


def test_hazmat_endorsement(worker: Worker) -> None:
    """
    Test when adding a worker with hazmat endorsement and no hazmat_expiration_date,
    that validation error is thrown for date not being entered.
    """
    with pytest.raises(ValidationError) as excinfo:
        worker.profile.update_worker_profile(endorsements="H")

    assert excinfo.value.message_dict["hazmat_expiration_date"] == [
        "Hazmat expiration date is required for this endorsement. Please try again."
    ]


def test_x_endorsement(worker: Worker) -> None:
    """
    Test when adding a worker with X endorsement and no hazmat_expiration_date,
    that validation error is thrown for date not being entered.
    """
    with pytest.raises(ValidationError) as excinfo:
        worker.profile.update_worker_profile(endorsements="H")

    assert excinfo.value.message_dict["hazmat_expiration_date"] == [
        "Hazmat expiration date is required for this endorsement. Please try again."
    ]


def test_license_state(worker: Worker) -> None:
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


def test_license_expiration_date(worker: Worker) -> None:
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


def test_worker_endorsement_choices(worker: Worker) -> None:
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


def test_worker_sex_choices(worker: Worker) -> None:
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
