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
from rest_framework.test import APIClient
import pytest

from accounts.models import Token
from organization.models import Organization

pytestmark = pytest.mark.django_db


def test_idempotency_middleware_fails_with_missing_header(
    token: Token, organization: Organization
) -> None:
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
