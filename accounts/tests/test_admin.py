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
from django.contrib.auth import get_user_model
from django.test import Client
from django.urls import reverse

from accounts.tests.factories import UserFactory

pytestmark = pytest.mark.django_db


@pytest.fixture
def client():
    """
    Fixture to create a client.
    """
    yield Client()


@pytest.fixture
def user_test():
    """
    Fixture to create a user.
    """
    yield UserFactory(
        email="user@example.com",
        username="regularjoe",
        password="testpass123",
    )


@pytest.fixture
def admin_user(organization):
    """
    Fixture to create a superuser.
    """
    yield get_user_model().objects.create_superuser(
        organization=organization,
        email="admin@example.com",
        username="bigboss",
        password="anotherpassword1234%",
    )


def test_admin_users_list(client, admin_user, user_test):
    client.force_login(admin_user)
    url = reverse("admin:accounts_user_changelist")
    res = client.get(url)

    assert b"regularjoe" in res.content
    assert b"user@example.com" in res.content


def test_admin_edit_user_page(client, admin_user, user):
    client.force_login(admin_user)
    url = reverse("admin:accounts_user_change", args=[user.id])
    res = client.get(url)

    assert res.status_code == 200


def test_admin_create_user_page(client, admin_user):
    client.force_login(admin_user)
    url = reverse("admin:accounts_user_add")
    res = client.get(url)

    assert res.status_code == 200


def test_create_superuser_is_superuser_error(organization):
    """
    Test creating superuser throws
    value error
    """

    with pytest.raises(ValueError) as excinfo:
        get_user_model().objects.create_superuser(
            organization=organization,
            username="test_admin",
            email="test@admin.com",
            password="test_admin",
            is_superuser=False,
            is_staff=True,
        )

    assert excinfo.value.__str__() == "Superuser must have is_superuser=True."


def test_create_superuser_is_staff_error(organization):
    """
    Test creating superuser throws
    value error
    """

    with pytest.raises(ValueError) as excinfo:
        get_user_model().objects.create_superuser(
            organization=organization,
            username="test_admin",
            email="test@admin.com",
            password="test_admin",
            is_superuser=True,
            is_staff=False,
        )

    assert excinfo.value.__str__() == "Superuser must have is_staff=True."
