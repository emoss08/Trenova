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

import pytest
from django.contrib.auth import get_user_model
from rest_framework.exceptions import ValidationError

from accounts.serializers import UserSerializer
from accounts.tests.factories import JobTitleFactory

pytestmark = pytest.mark.django_db


class TestUserAPI:
    def test_get(self, api_client):
        """
        Test get users
        """
        response = api_client.get("/api/users/")
        assert response.status_code == 200

    def test_get_by_id(self, api_client, user_api):
        """
        Test get user by ID
        """
        response = api_client.get(f"/api/users/{user_api.data['id']}/")
        assert response.status_code == 200

    def test_create_success(self, api_client):
        """
        Test Create user
        """
        job_title = JobTitleFactory()

        payload = {
            "username": "test_user",
            "email": "test_user@example.com",
            "password": "test_password1234%",
            "profile": {
                "first_name": "test",
                "last_name": "user",
                "address_line_1": "test",
                "city": "test",
                "state": "NC",
                "zip_code": "12345",
                "title": job_title.id,
            },
        }

        response = api_client.post("/api/users/", payload, format="json")

        assert response.status_code == 201
        user = get_user_model().objects.get(username=payload["username"])
        assert user.check_password(payload["password"])
        assert "password" not in response.data
        assert response.data["username"] == payload["username"]
        assert response.data["email"] == payload["email"]

    def test_user_with_email_exists_error(self, api_client, organization):
        """
        Test Create user with email exists
        """
        payload = {
            "username": "test_user2",
            "email": "test_user@example.com",
            "password": "test_password1234%",
            "profile": {
                "first_name": "test",
                "last_name": "user",
                "address_line_1": "test",
                "city": "test",
                "state": "NC",
                "zip_code": "12345",
            },
        }
        get_user_model().objects.create_user(
            organization=organization,
            username=payload["username"],
            email=payload["email"],
            password=payload["password"],
        )
        response = api_client.post("/api/users/", payload, format="json")
        assert response.status_code == 400

    def test_put(self, user_api, api_client):
        """
        Test Put request
        """
        response = api_client.put(
            f"/api/users/{user_api.data['id']}/",
            {
                "username": "test2342",
                "email": "test@test.com",
                "profile": {
                    "first_name": "test",
                    "last_name": "user",
                    "address_line_1": "test",
                    "city": "test",
                    "state": "NC",
                    "zip_code": "12345",
                },
            },
            format="json",
        )

        assert response.status_code == 200
        assert response.data["username"] == "test2342"
        assert response.data["email"] == "test@test.com"
        assert response.data["profile"]["first_name"] == "Test"
        assert response.data["profile"]["last_name"] == "User"
        assert response.data["profile"]["address_line_1"] == "test"
        assert response.data["profile"]["city"] == "test"
        assert response.data["profile"]["state"] == "NC"
        assert response.data["profile"]["zip_code"] == "12345"
        assert "password" not in response.data

    def test_delete(self, user_api, token, api_client):
        """
        Test delete user
        """
        response = api_client.delete(f"/api/users/{user_api.data['id']}/")
        assert response.status_code == 204
        assert response.data is None

    def test_user_cannot_change_password_on_update(self, user):
        """
        Test ValidationError is thrown when posting to update user endpoint
        with password.
        """
        payload = {
            "username": "test_user",
            "email": "test_user@example.com",
            "password": "test_password1234%",
            "profile": {
                "first_name": "test",
                "last_name": "user",
                "address_line_1": "test",
                "city": "test",
                "state": "NC",
                "zip_code": "12345",
            },
        }

        with pytest.raises(ValidationError) as excinfo:
            serializer = UserSerializer.update(
                self=UserSerializer, instance=user, validated_data=payload
            )
            serializer.is_valid(raise_exception=True)

        assert (
            "Password cannot be changed using this endpoint. Please use the change password endpoint."
            in str(excinfo.value.detail)
        )
        assert "code='invalid'" in str(excinfo.value.detail)
        assert excinfo.value.default_code == "invalid"
