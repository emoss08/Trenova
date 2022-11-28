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

from accounts.factories.user import JobTitleFactory, UserFactory


@pytest.fixture()
def user():
    """
    User fixture
    """
    return UserFactory()


@pytest.fixture()
def job_title():
    """
    Job title fixture
    """
    return JobTitleFactory()


@pytest.mark.django_db
def test_user_creation(user):
    """
    Test user creation
    """
    assert user is not None


@pytest.mark.django_db
def test_user_organization(user):
    """
    Test user organization
    """
    assert user.organization is not None


@pytest.mark.django_db
def test_user_profile_creation(user):
    """
    Test user profile creation
    """
    assert user.profile is not None


@pytest.mark.django_db
def test_user_profile_updated(user):
    """
    Test user profile updated
    """
    user.profile.first_name = "test_first_name"
    user.profile.save()
    assert user.profile.first_name == "test_first_name"


@pytest.mark.django_db
def test_user_updated(user):
    """
    Test user updated
    """
    user.username = "test_user_updated"
    user.save()
    assert user.username == "test_user_updated"


@pytest.mark.django_db
def test_create_job_title(job_title):
    """
    Test job title creation
    """
    assert job_title is not None


@pytest.mark.django_db
def test_update_job_title(job_title):
    """
    Test job title update
    """
    job_title.name = "test_update_job_title"
    job_title.save()
    assert job_title.name == "test_update_job_title"
