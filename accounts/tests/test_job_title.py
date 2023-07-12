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
from accounts.models import JobTitle
from organization.models import BusinessUnit, Organization

pytestmark = pytest.mark.django_db


def test_list(job_title: JobTitle) -> None:
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


def test_update_job_title(job_title: JobTitle) -> None:
    """
    Test job title update
    """
    job_title.name = "test_update_job_title"
    job_title.save()

    assert job_title.name == "test_update_job_title"
