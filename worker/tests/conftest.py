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

import os
import shutil
from collections.abc import Generator
from pathlib import Path
from typing import Any

import pytest

from accounts.models import User
from dispatch.factories import CommentTypeFactory, FleetCodeFactory
from organization.models import Organization
from worker import models
from worker.factories import WorkerFactory

pytestmark = pytest.mark.django_db


@pytest.fixture
def worker() -> Generator[Any, Any, None]:
    """
    Worker Fixture
    """

    yield WorkerFactory()


@pytest.fixture
def comment_type() -> Generator[Any, Any, None]:
    """
    Comment Type Fixture
    """

    yield CommentTypeFactory()


@pytest.fixture
def worker_api(organization: Organization, user: User) -> Generator[Any, Any, None]:
    """
    Worker Fixture
    """
    fleet = FleetCodeFactory()
    yield models.Worker.objects.create(
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
        manager=user,
        entered_by=user,
        fleet=fleet,
    )


def remove_media_directory(file_path: str) -> None:
    """Remove Media Directory after test tear down.

    Primary usage is when tests are performing file uplaods.
    This method deletes the media directory after the test.
    This is to prevent the media directory from filling up
    with test files.

    Args:
        file_path (str): path to directory in media folder.

    Returns:
        None
    """

    base_dir = Path(__file__).resolve().parent.parent
    media_dir = os.path.join(base_dir, f"media/{file_path}")

    if os.path.exists(media_dir):
        shutil.rmtree(media_dir, ignore_errors=True, onerror=None)


def remove_file(file_path: str) -> None:
    """Remove File after test tear down.

    Primary usage is when tests are performing file uplaods.
    This method deletes the file after the test.
    This is to prevent the media directory from filling up
    with test files.

    Args:
        file_path (str): path to file in media folder.

    Returns:
        None
    """

    base_dir = Path(__file__).resolve().parent.parent
    file = os.path.join(base_dir, f"media/{file_path}")

    if os.path.exists(file):
        os.remove(file)
