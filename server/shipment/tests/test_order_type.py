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
from django.urls import reverse
from rest_framework.response import Response
from rest_framework.test import APIClient

from organization.models import BusinessUnit, Organization
from shipment import models

pytestmark = pytest.mark.django_db


def test_list(shipment_type: models.ShipmentType) -> None:
    """
    Test shipment type list
    """
    assert shipment_type


def test_create(organization: Organization, business_unit: BusinessUnit) -> None:
    """
    Test shipment type Create
    """

    ord_type = models.ShipmentType.objects.create(
        organization=organization,
        business_unit=business_unit,
        is_active=True,
        name="foo bar",
        description="foo bar",
    )

    assert ord_type
    assert ord_type.is_active
    assert ord_type.name == "foo bar"
    assert ord_type.description == "foo bar"


def test_update(shipment_type: models.ShipmentType) -> None:
    """
    Test shipment type update
    """

    ord_type = models.ShipmentType.objects.get(id=shipment_type.id)

    ord_type.name = "Foo Bart"

    ord_type.save()

    assert ord_type
    assert ord_type.name == "Foo Bart"


def test_get(api_client: APIClient) -> None:
    """
    Test get shipment type
    """
    response = api_client.get("/api/shipment_types/")
    assert response.status_code == 200


def test_get_by_id(api_client: APIClient, shipment_type_api: Response) -> None:
    """
    Test get shipment type by id
    """
    response = api_client.get(
        reverse("shipment-types-detail", kwargs={"pk": shipment_type_api.data["id"]})
    )

    assert response.status_code == 200
    assert response.data["name"] == "Foo Bar"
    assert response.data["description"] == "Foo Bar"
    assert response.data["is_active"] is True


def test_put(
    api_client: APIClient, shipment_type_api: Response, organization: Organization
) -> None:
    """
    Test put shipment type
    """
    response = api_client.put(
        reverse("shipment-types-detail", kwargs={"pk": shipment_type_api.data["id"]}),
        {
            "organization": organization.id,
            "name": "New Name",
            "description": "New Description",
            "is_active": False,
        },
    )

    assert response.status_code == 200
    assert response.data["name"] == "New Name"
    assert response.data["description"] == "New Description"
    assert response.data["is_active"] is False


def test_delete(api_client: APIClient, shipment_type_api: Response) -> None:
    """
    Test Delete shipment type
    """
    response = api_client.delete(
        reverse("shipment-types-detail", kwargs={"pk": shipment_type_api.data["id"]}),
    )

    assert response.status_code == 204
    assert not response.data
