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
from rest_framework.response import Response
from rest_framework.test import APIClient

from order import models
from organization.models import BusinessUnit, Organization

pytestmark = pytest.mark.django_db


def test_list(reason_code: models.ReasonCode) -> None:
    """
    Test Reason Code list
    """
    assert reason_code is not None


def test_create(organization: Organization, business_unit: BusinessUnit) -> None:
    """
    Test Reason Code Create
    """
    r_code = models.ReasonCode.objects.create(
        organization=organization,
        business_unit=business_unit,
        is_active=True,
        code="foobo",
        description="foo bar",
        code_type="VOIDED",
    )

    assert r_code is not None
    assert r_code.is_active is True
    assert r_code.code == "foobo"
    assert r_code.description == "foo bar"
    assert r_code.code_type == "VOIDED"


def test_update(reason_code: models.ReasonCode) -> None:
    """
    Test order type update
    """

    r_code = models.ReasonCode.objects.get(id=reason_code.id)

    r_code.code = "NEWTY"

    r_code.save()

    assert r_code is not None
    assert r_code.code == "NEWTY"


def test_get(api_client: APIClient) -> None:
    """
    Test get Reason Code
    """
    response = api_client.get("/api/reason_codes/")
    assert response.status_code == 200


def test_get_by_id(api_client: APIClient, reason_code_api: Response) -> None:
    """
    Test get Reason Code by id
    """
    response = api_client.get(f"/api/reason_codes/{reason_code_api.data['id']}/")

    assert response.status_code == 200
    assert response.data["code"] == "NEWT"
    assert response.data["description"] == "Foo Bar"
    assert response.data["is_active"] is True
    assert response.data["code_type"] == "VOIDED"


def test_put(
    api_client: APIClient, reason_code_api: Response, organization: Organization
) -> None:
    """
    Test put Reason Code
    """
    response = api_client.put(
        f"/api/reason_codes/{reason_code_api.data['id']}/",
        {
            "organization": organization.id,
            "code": "FOBO",
            "description": "New Description",
            "is_active": False,
            "code_type": "VOIDED",
        },
    )

    assert response.status_code == 200
    assert response.data["code"] == "FOBO"
    assert response.data["description"] == "New Description"
    assert response.data["is_active"] is False


def test_delete(api_client: APIClient, reason_code_api: Response) -> None:
    """
    Test Delete Reason Code
    """
    response = api_client.delete(f"/api/reason_codes/{reason_code_api.data['id']}/")

    assert response.status_code == 204
    assert response.data is None
