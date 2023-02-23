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

from accounts import models

pytestmark = pytest.mark.django_db


def test_list(job_title):
    """
    Test Job Title list
    """
    assert job_title is not None


def test_create(job_title):
    """
    Test job title creation
    """
    new_job_title = models.JobTitle.objects.create(
        organization=job_title.organization,
        is_active=True,
        name="TEST",
        description="Another Description",
        job_function="SYS_ADMIN",
    )

    assert new_job_title is not None
    assert new_job_title.name == "TEST"
    assert new_job_title.description == "Another Description"


def test_update_job_title(job_title):
    """
    Test job title update
    """
    job_title.name = "test_update_job_title"
    job_title.save()

    assert job_title.name == "test_update_job_title"
