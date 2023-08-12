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
from billing import models
from organization.models import BusinessUnit, Organization
from rest_framework.response import Response
from rest_framework.test import APIClient

pytestmark = pytest.mark.django_db


def test_list(charge_type: models.ChargeType) -> None:
    """
    Test Charge Type List
    """
    assert charge_type is not None


def test_create(organization: Organization, business_unit: BusinessUnit) -> None:
    """
    Test Create Charge Type
    """
    charge_type = models.ChargeType.objects.create(
        organization=organization,
        business_unit=business_unit,
        name="test",
        description="Test Description",
    )

    assert charge_type is not None
    assert charge_type.name == "test"
    assert charge_type.description == "Test Description"


def test_update(charge_type: models.ChargeType) -> None:
    """
    Test Charge Type update
    """

    char_type = models.ChargeType.objects.get(id=charge_type.id)

    char_type.name = "maybe"
    char_type.save()

    assert char_type is not None
    assert char_type.name == "maybe"


def test_get(api_client: APIClient) -> None:
    """
    Test get Charge Type
    """
    response = api_client.get("/api/charge_types/")
    assert response.status_code == 200


def test_get_by_id(api_client: APIClient, charge_type_api: Response) -> None:
    """
    Test get Charge Type by ID
    """

    response = api_client.get(f"/api/charge_types/{charge_type_api.data['id']}/")

    assert response.status_code == 200
    assert response.data["name"] == "foob"
    assert response.data["description"] == "Test Description"


def test_put(
    api_client: APIClient, charge_type_api: Response, organization: Organization
) -> None:
    """
    Test put Charge Type
    """

    response = api_client.put(
        f"/api/charge_types/{charge_type_api.data['id']}/",
        {"organization": organization.id, "name": "foo bar"},
        format="json",
    )

    assert response.status_code == 200
    assert response.data["name"] == "foo bar"


def test_delete(api_client: APIClient, charge_type_api: Response) -> None:
    """
    Test Delete Charge Type
    """

    response = api_client.delete(
        f"/api/charge_types/{charge_type_api.data['id']}/",
    )

    assert response.status_code == 204
    assert response.data is None
