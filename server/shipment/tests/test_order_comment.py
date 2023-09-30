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

from accounts.models import User
from dispatch.models import CommentType
from organization.models import BusinessUnit, Organization
from shipment import models

pytestmark = pytest.mark.django_db


def test_list(shipment_comment: models.ShipmentComment) -> None:
    """
    Test shipment list
    """
    assert shipment_comment is not None


def test_create(
    organization: Organization,
    user: User,
    shipment: models.Shipment,
    comment_type: CommentType,
    business_unit: BusinessUnit,
) -> None:
    """
    Test shipment Create
    """

    shipment_comment = models.ShipmentComment.objects.create(
        organization=organization,
        business_unit=business_unit,
        shipment=shipment,
        comment_type=comment_type,
        comment="DONT BE SAD",
        entered_by=user,
    )
    assert shipment_comment is not None
    assert shipment_comment.shipment == order
    assert shipment_comment.comment_type == comment_type
    assert shipment_comment.comment == "DONT BE SAD"
    assert shipment_comment.entered_by == user


def test_update(shipment_comment: models.ShipmentComment) -> None:
    """
    Test shipment update
    """

    ord_comment = models.ShipmentComment.objects.get(id=shipment_comment.id)
    ord_comment.comment = "GET GLAD"
    ord_comment.save()

    assert ord_comment is not None
    assert ord_comment.comment == "GET GLAD"


def test_get_by_id(
    shipment_comment_api: Response,
    shipment_api: Response,
    comment_type: CommentType,
    user: User,
    api_client: APIClient,
) -> None:
    """
    Test get shipment Comment by ID
    """
    response = api_client.get(
        f"/api/shipment_comments/{shipment_comment_api.data['id']}/"
    )
    assert response.status_code == 200
    assert response.data is not None
    assert (
        f"{response.data['order']}" == shipment_api.data["id"]
    )  # returns UUID <UUID>, convert to F-string
    assert response.data["comment_type"] == comment_type.id
    assert response.data["comment"] == "IM HAPPY YOU'RE HERE"
    assert response.data["entered_by"] == user.id


def test_put(
    api_client: APIClient,
    shipment_api: Response,
    shipment_comment_api: Response,
    comment_type: CommentType,
    user: User,
) -> None:
    """
    Test put shipment Comment
    """
    response = api_client.put(
        f"/api/shipment_comments/{shipment_comment_api.data['id']}/",
        {
            "shipment": f"{shipment_api.data['id']}",
            "comment_type": f"{comment_type.id}",
            "comment": "BE GLAD IM HERE",
            "entered_by": f"{user.id}",
        },
        format="json",
    )

    assert response.status_code == 200
    assert response.data is not None
    assert (
        f"{response.data['order']}" == shipment_api.data["id"]
    )  # returns UUID <UUID>, convert to F-string
    assert response.data["comment_type"] == comment_type.id
    assert response.data["comment"] == "BE GLAD IM HERE"
    assert response.data["entered_by"] == user.id


def test_patch(api_client: APIClient, shipment_comment_api: Response) -> None:
    """
    Test patch shipment Comment
    """
    response = api_client.patch(
        f"/api/shipment_comments/{shipment_comment_api.data['id']}/",
        {
            "comment": "DONT BE SAD GET GLAD",
        },
        format="json",
    )

    assert response.status_code == 200
    assert response.data is not None
    assert response.data["comment"] == "DONT BE SAD GET GLAD"


def test_delete(api_client: APIClient, shipment_comment_api: Response) -> None:
    """
    Test delete shipment Comment
    """
    response = api_client.delete(
        f"/api/shipment_comments/{shipment_comment_api.data['id']}/"
    )

    assert response.status_code == 204
    assert response.data is None
