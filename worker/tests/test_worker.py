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
import threading

import pytest
from django.core.exceptions import ValidationError
from django.db import transaction

from dispatch.factories import CommentTypeFactory
from worker.models import WorkerComment

pytestmark = pytest.mark.django_db


def test_worker_creation(worker):
    """
    Test worker creation
    """
    assert worker is not None


def test_worker_code_hook(worker):
    """
    Test worker code is generated from create_worker_code_before_save BEFORE_SAVE hook
    """
    assert worker.code is not None


def test_worker_type_choices(worker):
    """
    Test Worker Type choices throws ValidationError when the passed choice is not valid.
    """
    with pytest.raises(ValidationError) as excinfo:
        worker.worker_type = "invalid"
        worker.full_clean()

    assert excinfo.value.message_dict["worker_type"] == [
        "Value 'invalid' is not a valid choice."
    ]


def test_worker_profile_hook(worker):
    """
    Test worker code is generated from create_worker_profile_after_create AFTER_CREATE hook
    """
    assert worker.profile is not None


def test_worker_contact_creation(worker):
    """
    Test worker contact creation
    """
    assert worker.worker_contact is not None


def test_worker_contact_update(worker):
    """
    Test worker contact update
    """
    worker.worker_contact.phone_number = "1234567890"
    worker.worker_contact.save()
    assert worker.worker_contact.phone_number == "1234567890"


def test_worker_comment_creation(worker):
    """
    Test worker comment creation
    """
    assert worker.worker_comment is not None


def test_worker_comment_update(worker):
    """
    Test worker comment update
    """
    worker.worker_comment.comment = "Test comment"
    worker.worker_comment.save()
    assert worker.worker_comment.comment == "Test comment"