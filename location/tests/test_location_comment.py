# Create your tests here.
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

from location.factories import LocationCommentFactory


@pytest.fixture()
def location_comment():
    """
    Location comment fixture
    """
    return LocationCommentFactory()


@pytest.mark.django_db
def test_location_comment_creation(location_comment):
    """
    Test location comment creation
    """
    assert location_comment is not None


@pytest.mark.django_db
def test_location_comment_update(location_comment):
    """
    Test location comment update
    """
    location_comment.comment = "New comment"
    location_comment.save()
    assert location_comment.comment == "New comment"


@pytest.mark.django_db
def test_add_comment_to_location(location_comment):
    """
    Test add comment to location
    """
    location = LocationCommentFactory()
    location.location_comment = location_comment
    location.save()
    assert location.location_comment == location_comment
