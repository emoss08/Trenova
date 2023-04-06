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

from order import models
from organization.models import Organization

pytestmark = pytest.mark.django_db


def test_list(order_type: models.OrderType) -> None:
    """
    Test Order Type list
    """
    assert order_type


def test_create(organization: Organization) -> None:
    """
    Test Order Type Create
    """

    ord_type = models.OrderType.objects.create(
        organization=organization,
        is_active=True,
        name="foo bar",
        description="foo bar",
    )

    assert ord_type
    assert ord_type.is_active
    assert ord_type.name == "foo bar"
    assert ord_type.description == "foo bar"


def test_update(order_type: models.OrderType) -> None:
    """
    Test order type update
    """

    ord_type = models.OrderType.objects.get(id=order_type.id)

    ord_type.name = "Foo Bart"

    ord_type.save()

    assert ord_type
    assert ord_type.name == "Foo Bart"


def test_get(api_client: APIClient) -> None:
    """
    Test get Order Type
    """
    response = api_client.get("/api/order_types/")
    assert response.status_code == 200


def test_get_by_id(api_client: APIClient, order_type_api: Response) -> None:
    """
    Test get Order Type by id
    """
    response = api_client.get(
        reverse("order-types-detail", kwargs={"pk": order_type_api.data["id"]})
    )

    assert response.status_code == 200
    assert response.data["name"] == "Foo Bar"
    assert response.data["description"] == "Foo Bar"
    assert response.data["is_active"] is True


def test_put(api_client: APIClient, order_type_api: Response) -> None:
    """
    Test put Order Type
    """
    response = api_client.put(
        reverse("order-types-detail", kwargs={"pk": order_type_api.data["id"]}),
        {"name": "New Name", "description": "New Description", "is_active": False},
    )

    assert response.status_code == 200
    assert response.data["name"] == "New Name"
    assert response.data["description"] == "New Description"
    assert response.data["is_active"] is False


def test_delete(api_client: APIClient, order_type_api: Response) -> None:
    """
    Test Delete order type
    """
    response = api_client.delete(
        reverse("order-types-detail", kwargs={"pk": order_type_api.data["id"]}),
    )

    assert response.status_code == 204
    assert not response.data
