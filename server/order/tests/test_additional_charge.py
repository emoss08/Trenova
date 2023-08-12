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
from billing.models import AccessorialCharge
from order import models
from organization.models import BusinessUnit, Organization
from rest_framework.response import Response
from rest_framework.test import APIClient

pytestmark = pytest.mark.django_db


def test_list(additional_charge: models.AdditionalCharge) -> None:
    """
    Test Order Type list
    """
    assert additional_charge is not None


def test_create(
    organization: Organization,
    business_unit: BusinessUnit,
    order: models.Order,
    accessorial_charge: AccessorialCharge,
    user: User,
) -> None:
    """
    Test Order Type Create
    """

    add_charge = models.AdditionalCharge.objects.create(
        organization=organization,
        business_unit=business_unit,
        order=order,
        accessorial_charge=accessorial_charge,
        unit=1,
        entered_by=user,
    )

    assert add_charge is not None
    assert add_charge.order == order
    assert add_charge.accessorial_charge == accessorial_charge
    assert add_charge.unit == 1
    assert add_charge.entered_by == user


def test_update(
    accessorial_charge: AccessorialCharge, additional_charge: models.AdditionalCharge
) -> None:
    """
    Test order type update
    """

    add_charge = models.AdditionalCharge.objects.get(id=additional_charge.id)

    add_charge.accessorial_charge = accessorial_charge
    add_charge.save()

    assert add_charge is not None
    assert add_charge.accessorial_charge == accessorial_charge
    assert (
        add_charge.sub_total
        == accessorial_charge.charge_amount * additional_charge.unit
    )


def test_api_get(api_client: APIClient) -> None:
    """
    Test get Additional Charge
    """
    response = api_client.get("/api/additional_charges/")
    assert response.status_code == 200


def test_api_get_by_id(
    api_client: APIClient,
    additional_charge_api: Response,
    order: models.Order,
    user: User,
):
    """
    Test get Additional Charge by id
    """

    response = api_client.get(
        f"/api/additional_charges/{additional_charge_api.data['id']}/"
    )

    assert response.status_code == 200
    assert response.data is not None
    assert response.data["order"] == order.id
    assert response.data["unit"] == 2
    assert response.data["entered_by"] == user.id


def test_api_patch(
    api_client: APIClient, additional_charge_api: Response, order: models.Order
) -> None:
    """
    Test put Order Type
    """
    response = api_client.patch(
        f"/api/additional_charges/{additional_charge_api.data['id']}/",
        {"order": f"{order.id}"},
    )

    assert response.status_code == 200
    assert response.data is not None
    assert response.data["order"] == order.id


def test_api_delete(api_client: APIClient, additional_charge_api: Response) -> None:
    """
    Test Delete Additional Charge
    """
    response = api_client.delete(
        f"/api/additional_charges/{additional_charge_api.data['id']}/"
    )

    assert response.status_code == 204
    assert response.data is None
