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
from accounts import models
from organization.models import BusinessUnit, Organization
from rest_framework.test import APIClient

pytestmark = pytest.mark.django_db


def test_list(job_title: models.JobTitle) -> None:
    """
    Test Job Title list
    """
    assert job_title is not None


def test_create(organization: Organization, business_unit: BusinessUnit) -> None:
    """
    Test job title creation
    """
    new_job_title = models.JobTitle.objects.create(
        organization=organization,
        business_unit=business_unit,
        status="A",
        name="TEST",
        description="Another Description",
        job_function="SYS_ADMIN",
    )

    assert new_job_title is not None
    assert new_job_title.name == "TEST"
    assert new_job_title.description == "Another Description"


def test_update_job_title(job_title: models.JobTitle) -> None:
    """
    Test job title update
    """
    job_title.name = "test_update_job_title"
    job_title.save()

    assert job_title.name == "test_update_job_title"


def test_get_job_titles(api_client: APIClient) -> None:
    """Test get ``job titles`` request

    Args:
        api_client (APIClient): APIClient object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.get("/api/job_titles/")
    assert response.status_code == 200


def test_post_job_title(api_client: APIClient) -> None:
    """Test post ``job titles`` request

    Args:
        api_client (APIClient): APIClient object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.post(
        "/api/job_titles/", {"name": "test_job_title", "job_function": "MANAGER"}
    )
    assert response.status_code == 201
    assert response.data["name"] == "test_job_title"
    assert response.data["job_function"] == "MANAGER"


def test_put_job_title(api_client: APIClient, job_title: models.JobTitle) -> None:
    """Test put ``job titles`` request

    Args:
        api_client (APIClient): APIClient object
        job_title (models.JobTitle): JobTitle object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.put(
        f"/api/job_titles/{job_title.id}/",
        {"name": "test_job_title", "job_function": "MANAGER"},
    )
    assert response.status_code == 200
    assert response.data["name"] == "test_job_title"
    assert response.data["job_function"] == "MANAGER"


def test_delete_job_title(api_client: APIClient, job_title: models.JobTitle) -> None:
    """Test delete ``job titles`` request

    Args:
        api_client (APIClient): APIClient object
        job_title (models.JobTitle): JobTitle object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.delete(f"/api/job_titles/{job_title.id}/")
    assert response.status_code == 204
    assert response.data is None


def test_get_job_title(api_client: APIClient, job_title: models.JobTitle) -> None:
    """Test get ``job title`` request

    Args:
        api_client (APIClient): APIClient object
        job_title (models.JobTitle): JobTitle object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.get(f"/api/job_titles/{job_title.id}/")
    assert response.status_code == 200
    assert response.data["name"] == job_title.name
    assert response.data["job_function"] == job_title.job_function


def test_expand_users_is_true(
    api_client: APIClient, job_title: models.JobTitle
) -> None:
    """Test ``expand_users`` query param on job_titles endpoint

    Args:
        api_client (APIClient): APIClient object
        job_title (models.JobTitle): JobTitle object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.get("/api/job_titles/?expand_users=true")

    assert response.status_code == 200
    assert response.data["results"][0]["users"] is not None


def test_expand_users_is_false(
    api_client: APIClient, job_title: models.JobTitle
) -> None:
    """Test ``expand_users`` query param on job_titles endpoint

    Args:
        api_client (APIClient): APIClient object
        job_title (models.JobTitle): JobTitle object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.get("/api/job_titles/?expand_users=false")

    assert response.status_code == 200
    assert "users" not in response.data["results"][0]
