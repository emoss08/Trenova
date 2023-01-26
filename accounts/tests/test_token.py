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
from django.urls import reverse
from rest_framework.exceptions import ValidationError
from rest_framework.test import APIClient

from accounts.serializers import TokenProvisionSerializer

pytestmark = pytest.mark.django_db

TOKEN_URL = reverse("provision-token")

client = APIClient()

def test_obtain_token(organization):
    """
    Test obtain token successfully.
    """
    user = get_user_model().objects.create_user(
        organization=organization,
        username="test_user",
        email="test_user@example",
        password="test_password1234%",
    )

    payload = {
        "username": user.username,
        "password": "test_password1234%",
    }

    response = client.post(TOKEN_URL, payload, format="json")

    assert response.status_code == 200
    assert "api_token" in response.data
    assert "user_id" in response.data

def test_obtain_token_with_bad_credentials(user):
    """
    Test obtain token throws ValidationError when the credentials
    given are invalid.
    """

    payload = {
        "username": user.username,
        "password": "test_password1234%",
    }

    with pytest.raises(ValidationError) as excinfo:
        serializer = TokenProvisionSerializer(data=payload)
        serializer.is_valid(raise_exception=True)

    assert "User with the given credentials does not exist. Please try again." in str(excinfo.value.detail)
    assert excinfo.value.default_code == "invalid"
    assert "code='authorization'" in str(excinfo.value.detail["non_field_errors"])
def test_obtain_token_without_username():
    """
    Test obtain token throws ValidationError when the username
    is not given.
    """

    payload = {
        "password": "test_password1234%",
    }

    with pytest.raises(ValidationError) as excinfo:
        serializer = TokenProvisionSerializer(data=payload)
        serializer.is_valid(raise_exception=True)

    assert "This field is required." in str(excinfo.value.detail["username"])
    assert "code='required'" in str(excinfo.value.detail["username"])
    assert excinfo.value.default_code == "invalid"