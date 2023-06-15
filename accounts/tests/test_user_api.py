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
from django.core import mail
from rest_framework.exceptions import ValidationError
from rest_framework.response import Response
from rest_framework.test import APIClient

from accounts.models import User
from accounts.serializers import UserSerializer
from accounts.tests.factories import JobTitleFactory, UserFactory
from organization.models import Organization

pytestmark = pytest.mark.django_db


def test_get(api_client: APIClient) -> None:
    """
    Test get users
    """
    response = api_client.get("/api/users/")
    assert response.status_code == 200


def test_get_by_id(api_client: APIClient, user_api: Response) -> None:
    """
    Test get user by ID
    """
    response = api_client.get(f"/api/users/{user_api.data['id']}/")

    assert response.status_code == 200


def test_create_success(api_client: APIClient, organization: Organization) -> None:
    """
    Test Create user
    """
    job_title = JobTitleFactory()

    payload = {
        "organization": organization.id,
        "username": "test_user",
        "email": "test_user@example.com",
        "profile": {
            "organization": organization.id,
            "first_name": "test",
            "last_name": "user",
            "address_line_1": "test",
            "city": "test",
            "state": "NC",
            "zip_code": "12345",
            "job_title": job_title.id,
        },
    }

    response = api_client.post("/api/users/", payload, format="json")

    assert response.status_code == 201
    assert "password" not in response.data
    assert response.data["username"] == payload["username"]
    assert response.data["email"] == payload["email"]
    assert len(mail.outbox) == 1
    assert "You have been added to " in mail.outbox[0].subject


def test_user_with_email_exists_error(
    api_client: APIClient, organization: Organization
) -> None:
    """
    Test Create user with email exists
    """
    payload = {
        "username": "test_user2",
        "email": "test_user@example.com",
        "profile": {
            "first_name": "test",
            "last_name": "user",
            "address_line_1": "test",
            "city": "test",
            "state": "NC",
            "zip_code": "12345",
        },
    }
    User.objects.create_user(
        organization=organization,
        username=payload["username"],
        email=payload["email"],
    )
    response = api_client.post("/api/users/", payload, format="json")
    assert response.status_code == 400


def test_put(
    user_api: Response, api_client: APIClient, organization: Organization
) -> None:
    """
    Test Put request
    """
    response = api_client.put(
        f"/api/users/{user_api.data['id']}/",
        {
            "organization": organization.id,
            "username": "test2342",
            "email": "test@test.com",
            "profile": {
                "organization": organization.id,
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


def test_delete(user_api: Response, api_client: APIClient) -> None:
    """
    Test delete user
    """
    response = api_client.delete(f"/api/users/{user_api.data['id']}/")
    assert response.status_code == 204
    assert response.data is None


def test_user_cannot_change_password_on_update(user: User) -> None:
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


def test_inactive_user_cannot_login(api_client: APIClient, user_api: Response) -> None:
    """
    Test inactive user cannot log in

    Args:
        api_client (APIClient): API Client
        user_api (Response): User API Response

    Returns:
        None: This function does not return anything.
    """
    user = User.objects.get(id=user_api.data["id"])
    user.is_active = False
    user.save()

    response = api_client.post(
        "/api/login/",
        {"username": user_api.data["username"], "password": "trashuser12345%"},
    )
    assert response.status_code == 400


def test_login_user(unauthenticated_api_client: APIClient, user_api: Response) -> None:
    """
    Test login user

    Args:
        unauthenticated_api_client (APIClient): API Client
        user_api (Response): User API Response

    Returns:
        None: This function does not return anything.

    """

    user = User.objects.get(id=user_api.data["id"])

    user.set_password("trashuser12345%")
    user.save()

    response = unauthenticated_api_client.post(
        "/api/login/",
        {"username": user_api.data["username"], "password": "trashuser12345%"},
    )
    assert response.status_code == 200
    assert response.data["token"]

    user.refresh_from_db()
    assert user.online is True
    assert user.last_login


def test_logout_user(api_client: APIClient, user_api: Response) -> None:
    """
    Test logout user

    Args:
        api_client (APIClient): API Client
        user_api (Response): User API Response

    Returns:
        None: This function does not return anything.
    """
    response = api_client.post("/api/logout/")
    assert response.status_code == 204

    user = User.objects.get(id=user_api.data["id"])
    assert user.online is False


def test_reset_password(unauthenticated_api_client: APIClient, user: User) -> None:
    """Test ``reset_password`` endpoint successfully resets password and sends email

    Args:
        unauthenticated_api_client (APIClient): Api Client
        user (): User

    Returns:
        None: This function does not return anything.
    """

    response = unauthenticated_api_client.post(
        "/api/reset_password/",
        {"email": user.email},
    )
    assert response.status_code == 200
    assert (
        response.data["message"]
        == "Password reset successful. Please check your email for the new password."
    )
    # Assert email was sent
    assert len(mail.outbox) == 1
    assert mail.outbox[0].subject == "Your password has been reset"


def test_validate_reset_password(unauthenticated_api_client: APIClient) -> None:
    """Test ``reset_password`` endpoint throws ValidationError if email is not found

    Args:
        unauthenticated_api_client (APIClient): API Client

    Returns:
        None: This function does not return anything.
    """

    response = unauthenticated_api_client.post(
        "/api/reset_password/",
        {"email": "random@monta.io"},
    )
    assert response.status_code == 400
    assert (
        response.data["email"][0]
        == "No user found with the given email exists. Please try again."
    )


def test_validate_reset_password_with_invalid_email(
    unauthenticated_api_client: APIClient,
) -> None:
    """Test ``reset_password`` endpoint throws ValidationError if email is not found

    Args:
        unauthenticated_api_client (APIClient): API Client

    Returns:
        None: This function does not return anything.
    """

    response = unauthenticated_api_client.post(
        "/api/reset_password/",
        {"email": "random"},
    )
    assert response.status_code == 400
    assert response.data["email"][0] == "Enter a valid email address."


def test_validate_reset_password_with_inactive_user(
    unauthenticated_api_client: APIClient, user: User
) -> None:
    """Test ``reset_password`` endpoint throws ValidationError if email is not found

    Args:
        unauthenticated_api_client (APIClient): API Client
        user (User): User

    Returns:
        None: This function does not return anything.
    """

    user.is_active = False
    user.save()

    response = unauthenticated_api_client.post(
        "/api/reset_password/",
        {"email": user.email},
    )
    assert response.status_code == 400
    assert (
        response.data["email"][0]
        == "This user is not active. Please contact support for assistance."
    )


def test_change_email(user: User) -> None:
    """Test ``reset_password`` endpoint successfully resets password and sends email

    Args:
        user (User): User

    Returns:
        None: This function does not return anything.
    """

    new_password = "new_password1234%"
    user.set_password(new_password)
    user.save()
    user.refresh_from_db()

    client = APIClient()
    client.force_authenticate(user=user)

    response = client.post(
        "/api/change_email/",
        {"email": "anothertest@monta.io", "current_password": "new_password1234%"},
    )
    assert response.status_code == 200
    assert response.data["message"] == "Email successfully changed."


def test_change_email_with_invalid_password(user: User) -> None:
    """Test ``reset_password`` endpoint throws ValidationError if email is not found

    Args:
        user (User): User

    Returns:
        None: This function does not return anything.
    """

    new_password = "new_password1234%"
    user.set_password(new_password)
    user.save()
    user.refresh_from_db()

    client = APIClient()
    client.force_authenticate(user=user)

    response = client.post(
        "/api/change_email/",
        {"email": "test_email@monta.io", "current_password": "wrong_password"},
    )

    assert response.status_code == 400
    assert (
        response.data["current_password"][0]
        == "Current password is incorrect. Please try again."
    )


def test_change_email_with_same_email(user: User) -> None:
    """Test ``reset_password`` endpoint throws ValidationError if email is not found

    Args:
        user (User): User

    Returns:
        None: This function does not return anything.

    """

    new_password = "new_password1234%"
    user.set_password(new_password)
    user.save()
    user.refresh_from_db()

    client = APIClient()
    client.force_authenticate(user=user)

    response = client.post(
        "/api/change_email/",
        {"email": user.email, "current_password": "new_password1234%"},
    )

    assert response.status_code == 400
    assert (
        response.data["email"][0]
        == "New email cannot be the same as the current email."
    )


def test_change_email_with_other_users_email(user: User) -> None:
    """Test ``reset_password`` endpoint throws ValidationError if email is not found

    Args:
        user (User): User

    Returns:
        None: This function does not return anything.
    """

    new_password = "new_password1234%"
    user.set_password(new_password)
    user.save()
    user.refresh_from_db()

    user_2 = UserFactory()
    user_2.email = "test@monta.io"
    user_2.save()

    client = APIClient()
    client.force_authenticate(user=user)

    response = client.post(
        "/api/change_email/",
        {"email": user_2.email, "current_password": "new_password1234%"},
    )

    assert response.status_code == 400
    assert response.data["email"][0] == "A user with the given email already exists."


def test_validate_password_not_allowed_on_post(
    api_client: APIClient, organization: Organization
) -> None:
    """
    Test Create user
    """
    job_title = JobTitleFactory()

    payload = {
        "organization": organization.id,
        "username": "test_user",
        "email": "test_user@example.com",
        "password": "test_password",
        "profile": {
            "organization": organization.id,
            "first_name": "test",
            "last_name": "user",
            "address_line_1": "test",
            "city": "test",
            "state": "NC",
            "zip_code": "12345",
            "job_title": job_title.id,
        },
    }

    response = api_client.post("/api/users/", payload, format="json")

    assert response.status_code == 400
    assert response.data["errors"][0]["attr"] == "password"
    assert (
        response.data["errors"][0]["detail"]
        == "Password cannot be added directly to a user. Please use the password reset endpoint."
    )
