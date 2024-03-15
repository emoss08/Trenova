# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
import threading
import typing
import uuid

import pytest
from rest_framework.test import APIClient

from accounts.models import Token
from organization.models import Organization

pytestmark = pytest.mark.django_db


def test_idempotency_middleware_fails_with_missing_header(
    token: Token, organization: Organization
) -> None:
    """Test that the idempotency middleware returns an error when the idempotency key header is missing.

    Args:
        token (Token): The token to use for authentication.
        organization (Organization): The organization to use for the request.

    Returns:
        None: This function does not return anything.
    """
    api_client = APIClient()
    api_client.credentials(HTTP_AUTHORIZATION=f"Bearer {token.key}")
    response = api_client.post(
        "/api/document_classifications/",
        {
            "organization": organization.id,
            "name": "test",
            "description": "Test Description",
        },
    )

    data = response.json()

    assert response.status_code == 400
    assert (
        data["message"]
        == "Idempotency key validation failed. Please provide a valid idempotency key in the request header."
    )
    assert data["code"] == "idempotency_key_validation_failed"


def send_request(
    api_client: APIClient,
    path: str,
    data: dict[str, typing.Any],
    idempotency_key: str,
    results: list[typing.Any],
    lock: threading.Lock,
) -> None:
    """Send a request to the API.

    Args:
        api_client (APIClient): The API client to use for the request.
        path (str): The path to send the request to.
        data (dict[str, typing.Any]): The data to send in the request.
        idempotency_key (str): The idempotency key to use for the request.
        results (list[typing.Any]): The list to append the response to.
        lock (typing.Lock): The lock to use when appending to the results list.

    Returns:
        None: This function does not return anything.
    """
    response = api_client.post(
        path,
        data,
        **{
            "HTTP_X_IDEMPOTENCY_KEY": idempotency_key,
        },
    )
    with lock:
        results.append(response)


def test_idempotency_basic(token: Token, organization: Organization) -> None:
    """Test that a request with the same idempotency key does not perform the operation more than once.

    Args:
        token (Token): The token to use for authentication.
        organization (Organization): The organization to use for the request.

    Returns:
        None: This function does not return anything.
    """
    api_client = APIClient()
    api_client.credentials(HTTP_AUTHORIZATION=f"Bearer {token.key}")
    path = "/api/document_classifications/"
    data = {
        "organization": organization.id,
        "name": "test",
        "description": "Test Description",
    }
    idempotency_key = str(uuid.uuid4())

    # Send the same request twice
    response1 = api_client.post(
        path, data, **{"HTTP_X_IDEMPOTENCY_KEY": idempotency_key}
    )
    response2 = api_client.post(
        path, data, **{"HTTP_X_IDEMPOTENCY_KEY": idempotency_key}
    )

    assert response1.status_code == 201, "The first request should have succeeded."

    assert (
        response2.status_code == 400
    ), "The second request should indicate a duplicate operation."
