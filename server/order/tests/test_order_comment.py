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
from order import models
from organization.models import BusinessUnit, Organization

pytestmark = pytest.mark.django_db


def test_list(order_comment: models.OrderComment) -> None:
    """
    Test Order list
    """
    assert order_comment is not None


def test_create(
    organization: Organization,
    user: User,
    order: models.Order,
    comment_type: CommentType,
    business_unit: BusinessUnit,
) -> None:
    """
    Test Order Create
    """

    order_comment = models.OrderComment.objects.create(
        organization=organization,
        business_unit=business_unit,
        order=order,
        comment_type=comment_type,
        comment="DONT BE SAD",
        entered_by=user,
    )
    assert order_comment is not None
    assert order_comment.order == order
    assert order_comment.comment_type == comment_type
    assert order_comment.comment == "DONT BE SAD"
    assert order_comment.entered_by == user


def test_update(order_comment: models.OrderComment) -> None:
    """
    Test Order update
    """

    ord_comment = models.OrderComment.objects.get(id=order_comment.id)
    ord_comment.comment = "GET GLAD"
    ord_comment.save()

    assert ord_comment is not None
    assert ord_comment.comment == "GET GLAD"


def test_get_by_id(
    order_comment_api: Response,
    order_api: Response,
    comment_type: CommentType,
    user: User,
    api_client: APIClient,
) -> None:
    """
    Test get Order Comment by ID
    """
    response = api_client.get(f"/api/order_comments/{order_comment_api.data['id']}/")
    assert response.status_code == 200
    assert response.data is not None
    assert (
        f"{response.data['order']}" == order_api.data["id"]
    )  # returns UUID <UUID>, convert to F-string
    assert response.data["comment_type"] == comment_type.id
    assert response.data["comment"] == "IM HAPPY YOU'RE HERE"
    assert response.data["entered_by"] == user.id


def test_put(
    api_client: APIClient,
    order_api: Response,
    order_comment_api: Response,
    comment_type: CommentType,
    user: User,
) -> None:
    """
    Test put Order Comment
    """
    response = api_client.put(
        f"/api/order_comments/{order_comment_api.data['id']}/",
        {
            "order": f"{order_api.data['id']}",
            "comment_type": f"{comment_type.id}",
            "comment": "BE GLAD IM HERE",
            "entered_by": f"{user.id}",
        },
        format="json",
    )

    assert response.status_code == 200
    assert response.data is not None
    assert (
        f"{response.data['order']}" == order_api.data["id"]
    )  # returns UUID <UUID>, convert to F-string
    assert response.data["comment_type"] == comment_type.id
    assert response.data["comment"] == "BE GLAD IM HERE"
    assert response.data["entered_by"] == user.id


def test_patch(api_client: APIClient, order_comment_api: Response) -> None:
    """
    Test patch Order Comment
    """
    response = api_client.patch(
        f"/api/order_comments/{order_comment_api.data['id']}/",
        {
            "comment": "DONT BE SAD GET GLAD",
        },
        format="json",
    )

    assert response.status_code == 200
    assert response.data is not None
    assert response.data["comment"] == "DONT BE SAD GET GLAD"


def test_delete(api_client: APIClient, order_comment_api: Response) -> None:
    """
    Test delete Order Comment
    """
    response = api_client.delete(f"/api/order_comments/{order_comment_api.data['id']}/")

    assert response.status_code == 204
    assert response.data is None
