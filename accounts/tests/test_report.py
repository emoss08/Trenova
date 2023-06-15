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

from accounts.models import User
from reports import models, utils

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
        model=User, columns=columns, user=user, file_format=file_format
    )
    reports = models.UserReport.objects.all()

    assert reports.count() == 1
