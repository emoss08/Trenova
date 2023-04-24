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
from typing import Any
from collections.abc import Generator

import pytest

from location import factories, models

pytestmark = pytest.mark.django_db


@pytest.fixture()
def location_comment() -> Generator[Any, Any, None]:
    """
    Location comment fixture
    """
    return factories.LocationCommentFactory()


def test_location_comment_creation(location_comment: models.LocationComment) -> None:
    """
    Test location comment creation
    """
    assert location_comment is not None


def test_location_comment_update(location_comment: models.LocationComment) -> None:
    """
    Test location comment update
    """
    location_comment.comment = "New comment"
    location_comment.save()
    assert location_comment.comment == "New comment"
