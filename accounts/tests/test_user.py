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
from django.test import Client

from accounts.factories import JobTitleFactory, UserFactory
from accounts.models import User
from organization.factories import OrganizationFactory


@pytest.fixture()
def user():
    """
    User fixture
    """
    return UserFactory()


@pytest.fixture()
def job_title(user):
    """
    Job title fixture
    """
    return JobTitleFactory(organization=user.organization)


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
def test_job_title_not_active(user, job_title):
    """
    Test if the job title is not active,
    that validation error is raised
    """
    job_title.is_active = False
    job_title.save()
    user.profile.title = job_title
    with pytest.raises(ValidationError, match="Title is not active"):
        user.profile.save()


@pytest.mark.django_db
def test_create_superuser():
    """
    Test creating supe user
    """
    organization = OrganizationFactory()

    admin_user = User(
        organization=organization,
        username="test_admin",
        email="test_admin@admin.com",
        is_staff=True,
        is_superuser=True,
    )
    admin_user.set_password("test_admin")
    admin_user.save()

    client = Client()
    login_response = client.login(username="test_admin", password="test_admin")
    assert login_response is True


@pytest.mark.django_db
def test_create_superuser_is_superuser_error():
    """
    Test creating superuser throws
    value error
    """
    organization = OrganizationFactory()

    with pytest.raises(ValueError, match="Superuser must have is_superuser=True."):
        User.objects.create_superuser(
            organization=organization,
            username="test_admin",
            email="test@admin.com",
            password="test_admin",
            is_superuser=False,
            is_staff=True,
        )


@pytest.mark.django_db
def test_create_superuser_is_staff_error():
    """
    Test creating superuser throws
    value error
    """
    organization = OrganizationFactory()

    with pytest.raises(ValueError, match="Superuser must have is_staff=True."):
        User.objects.create_superuser(
            organization=organization,
            username="test_admin",
            email="test@admin.com",
            password="test_admin",
            is_superuser=True,
            is_staff=False,
        )
