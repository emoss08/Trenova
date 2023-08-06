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
from rest_framework.test import APIClient

from accounts.models import User
from reports import selectors

pytestmark = pytest.mark.django_db


def test_log_entries_get(api_client: APIClient) -> None:
    """Test get ``log entries`` get request

    Args:
        api_client (APIClient): APIClient object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.get("/api/log_entries/?model_name=user&app_label=accounts")
    assert response.status_code == 200


def test_log_entries_throws_validation_error_when_model_name_missing(
    api_client: APIClient,
) -> None:
    """Test ``log entries`` endpoint throws validation error when
    query param ``model_name`` is missing.

    Args:
        api_client (APIClient): APIClient object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.get("/api/log_entries/?app_label=accounts")
    assert response.status_code == 400
    assert response.data["errors"][0]["code"] == "invalid"
    assert (
        response.data["errors"][0]["detail"]
        == "Query parameter 'model_name' is required"
    )
    assert response.data["errors"][0]["attr"] is None


def test_log_entries_throws_validation_error_app_label_missing(
    api_client: APIClient,
) -> None:
    """Test ``log entries`` endpoint throws validation error when
    query param ``app_label`` is missing.

    Args:
        api_client (APIClient): APIClient object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.get("/api/log_entries/?model_name=user")
    assert response.status_code == 400
    assert response.data["errors"][0]["code"] == "invalid"
    assert (
        response.data["errors"][0]["detail"]
        == "Query parameter 'app_label' is required"
    )
    assert response.data["errors"][0]["attr"] is None


def test_log_entries_doesnt_allow_post_request(
    api_client: APIClient,
) -> None:
    """Test ``log entries`` endpoint does not allow ``post`` request.

    Args:
        api_client (APIClient): APIClient object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.post("/api/log_entries/")
    assert response.status_code == 405
    assert response.data["errors"][0]["code"] == "method_not_allowed"
    assert response.data["errors"][0]["detail"] == 'Method "POST" not allowed.'
    assert response.data["errors"][0]["attr"] is None


def test_log_entries_doesnt_allow_put_request(
    api_client: APIClient,
) -> None:
    """Test ``log entries`` endpoint does not allow ``put`` request.

    Args:
        api_client (APIClient): APIClient object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.put("/api/log_entries/")
    assert response.status_code == 405
    assert response.data["errors"][0]["code"] == "method_not_allowed"
    assert response.data["errors"][0]["detail"] == 'Method "PUT" not allowed.'
    assert response.data["errors"][0]["attr"] is None


def test_log_entries_doesnt_allow_patch_request(
    api_client: APIClient,
) -> None:
    """Test ``log entries`` endpoint does not allow ``patch`` request.

    Args:
        api_client (APIClient): APIClient object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.patch("/api/log_entries/")
    assert response.status_code == 405
    assert response.data["errors"][0]["code"] == "method_not_allowed"
    assert response.data["errors"][0]["detail"] == 'Method "PATCH" not allowed.'
    assert response.data["errors"][0]["attr"] is None


def test_log_entries_doesnt_allow_delete_request(
    api_client: APIClient,
) -> None:
    """Test ``log entries`` endpoint does not allow ``delete`` request.

    Args:
        api_client (APIClient): APIClient object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.delete("/api/log_entries/")
    assert response.status_code == 405
    assert response.data["errors"][0]["code"] == "method_not_allowed"
    assert response.data["errors"][0]["detail"] == 'Method "DELETE" not allowed.'
    assert response.data["errors"][0]["attr"] is None


def test_log_entries_selector(user: User) -> None:
    """Test ``log entries`` selector

    Args:
        user(User): User object

    Returns:
        None: This function does not return anything.
    """
    data = selectors.get_audit_logs_by_model_name(
        model_name="user", app_label="accounts", organization_id=user.organization_id
    )
    assert data
    assert data.count() > 0
