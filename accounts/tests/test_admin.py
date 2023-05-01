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

from collections.abc import Generator
from typing import Any

import pytest
from django.test import Client
from django.urls import reverse

from accounts.models import User
from accounts.tests.factories import UserFactory
from organization.models import Organization

pytestmark = pytest.mark.django_db


@pytest.fixture
def client() -> Generator[Any, Any, None]:
    """
    Fixture to create a client.
    """
    yield Client()


@pytest.fixture
def user_test() -> Generator[Any, Any, None]:
    """
    Fixture to create a user.
    """
    yield UserFactory(
        email="user@example.com",
        username="regularjoe",
        password="testpass123",
    )


@pytest.fixture
def admin_user(organization: Organization) -> Generator[Any, Any, None]:
    """
    Fixture to create a superuser.
    """
    yield User.objects.create_superuser(
        organization=organization,
        email="admin@example.com",
        username="bigboss",
        password="anotherpassword1234%",
    )


def test_admin_users_list(client: Client, admin_user: User, user_test: User) -> None:
    client.force_login(admin_user)
    url = reverse("admin:accounts_user_changelist")
    res = client.get(url)

    assert b"regularjoe" in res.content
    assert b"user@example.com" in res.content


def test_admin_edit_user_page(client: Client, admin_user: User, user: User) -> None:
    client.force_login(admin_user)
    url = reverse("admin:accounts_user_change", args=[user.id])
    res = client.get(url)

    assert res.status_code == 200


def test_admin_create_user_page(client: Client, admin_user: User) -> None:
    client.force_login(admin_user)
    url = reverse("admin:accounts_user_add")
    res = client.get(url)

    assert res.status_code == 200


def test_create_superuser_is_superuser_error(organization: Organization) -> None:
    """
    Test creating superuser throws
    value error
    """

    with pytest.raises(ValueError) as excinfo:
        User.objects.create_superuser(
            organization=organization,
            username="test_admin",
            email="test@admin.com",
            password="test_admin",
            is_superuser=False,
            is_staff=True,
        )

    assert excinfo.value.__str__() == "Superuser must have is_superuser=True."


def test_create_superuser_is_staff_error(organization: Organization) -> None:
    """
    Test creating superuser throws
    value error
    """

    with pytest.raises(ValueError) as excinfo:
        User.objects.create_superuser(
            organization=organization,
            username="test_admin",
            email="test@admin.com",
            password="test_admin",
            is_superuser=True,
            is_staff=False,
        )

    assert excinfo.value.__str__() == "Superuser must have is_staff=True."
