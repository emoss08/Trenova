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


def test_worker_creation(worker: Worker) -> None:
    """
    Test worker creation
    """
    assert worker is not None


def test_worker_code_hook(worker: Worker) -> None:
    """
    Test worker code is generated from create_worker_code_before_save BEFORE_SAVE hook
    """
    assert worker.code is not None


def test_worker_type_choices(worker: Worker) -> None:
    """
    Test Worker Type choices throws ValidationError when the passed choice is not valid.
    """
    with pytest.raises(ValidationError) as excinfo:
        worker.worker_type = "invalid"
        worker.full_clean()

    assert excinfo.value.message_dict["worker_type"] == [
        "Value 'invalid' is not a valid choice."
    ]


def test_worker_profile_hook(worker: Worker) -> None:
    """
    Test worker code is generated from create_worker_profile_after_create AFTER_CREATE hook
    """
    assert worker.profile is not None


def test_worker_contact_creation(worker: Worker) -> None:
    """
    Test worker contact creation
    """
    assert worker.worker_contact is not None


def test_worker_contact_update(worker: Worker) -> None:
    """
    Test worker contact update
    """
    worker.worker_contact.phone_number = "1234567890"
    worker.worker_contact.save()
    assert worker.worker_contact.phone_number == "1234567890"


def test_worker_comment_creation(worker: Worker) -> None:
    """
    Test worker comment creation
    """
    assert worker.worker_comment is not None


def test_worker_comment_update(worker) -> None:
    """
    Test worker comment update
    """
    worker.worker_comment.comment = "Test comment"
    worker.worker_comment.save()
    assert worker.worker_comment.comment == "Test comment"
