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
from unittest.mock import Mock, patch

import pytest
from celery.exceptions import Retry
from django.core.files.uploadedfile import SimpleUploadedFile
from rest_framework.test import APIClient

from accounts.models import User
from core.exceptions import ServiceException
from reports import models, tasks, utils

pytestmark = pytest.mark.django_db


@pytest.mark.parametrize("file_format", ["csv", "xlsx", "pdf"])
def test_generate_report(file_format, user: User) -> None:
    """Test report is generated in various formats and stored in ``UserReport`` model.

    Args:
        file_format (str): The format of the file to generate.
        user (User): User object

    Returns:
        None: This function does not return anything.
    """

    # List of columns available on User model
    columns = [
        "username",
        "email",
        "date_joined",
        "is_staff",
        "profiles__first_name",
        "profiles__last_name",
        "profiles__address_line_1",
        "profiles__address_line_2",
        "profiles__city",
        "profiles__state",
        "profiles__zip_code",
        "profiles__phone_number",
        "profiles__is_phone_verified",
        "profiles__job_title__name",
        "profiles__job_title__description",
        "department__name",
        "department__description",
        "organization__name",
    ]

    utils.generate_report(
        model_name="User", columns=columns, user_id=user.id, file_format=file_format
    )
    reports = models.UserReport.objects.all()

    assert reports.count() == 1


@patch("reports.tasks.utils.generate_report")
def test_generate_report_task(mock_generate_report: Mock, user: User) -> None:
    """Test that the generate_report_task calls the generate_report function with the correct arguments.

    Args:
        mock_generate_report (Mock): Mocked function of the actual generate_report in reports.tasks.utils.

    Returns:
        None: This function does not return anything.
    """

    mock_generate_report.return_value = None

    model_name = "User"
    columns = [
        "username",
        "email",
        "date_joined",
        "is_staff",
    ]
    user_id = user.id
    file_format = "csv"

    tasks.generate_report_task(
        model_name=model_name,
        columns=columns,
        user_id=user_id,
        file_format=file_format,
    )

    mock_generate_report.assert_called_with(
        model_name=model_name,
        columns=columns,
        user_id=user_id,
        file_format=file_format,
    )
    mock_generate_report.assert_called_once()


@patch("reports.tasks.utils.generate_report")
def test_generate_report_task_failure(generate_report: Mock, user: User) -> None:
    """Test that a Retry exception is raised when the generate_report_task encounters an OperationalError.

    Args:
        generate_report: Mocked function of the actual generate_report in reports.tasks.utils.
        user: The User object for the test.

    Returns:
        None: This function does not return anything.
    """

    # Mock generate_report to throw an OperationalError
    generate_report.side_effect = ServiceException()

    with patch(
        "reports.tasks.generate_report_task.retry", side_effect=Retry()
    ) as generate_report_retry, pytest.raises(Retry):
        tasks.generate_report_task(
            model_name="InvalidModel",
            columns=[
                "username",
                "email",
                "date_joined",
                "is_staff",
            ],
            user_id=user.id,
            file_format="csv",
        )

    # Ensure that the retry method was called
    generate_report_retry.assert_called_once()


def test_user_report_get(api_client: APIClient) -> None:
    """Test get ``user report`` get request

    Args:
        api_client (APIClient): APIClient object

    Returns:
        None: This function does not return anything.
    """
    response = api_client.get("/api/custom_reports/")
    assert response.status_code == 200


def test_user_report_post(api_client: APIClient, user: User) -> None:
    """Test get ``user report`` post request

    Args:
        api_client (APIClient): APIClient object
        user (User): User object

    Returns:
        None: This function does not return anything.
    """

    response = api_client.post(
        "/api/user_reports/",
        {
            "organization": user.organization.id,
            "user": user.id,
            "report": SimpleUploadedFile(
                "report.csv", b"file_content", content_type="text/csv"
            ),
        },
    )
    print(response.data)
    assert response.status_code == 201
    assert response.data["user"] == user.id
