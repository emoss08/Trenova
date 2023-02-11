"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""

import pytest

from order import models

pytestmark = pytest.mark.django_db


def test_list(additional_charge):
    """
    Test Order Type list
    """
    assert additional_charge is not None


def test_create(organization, order, accessorial_charge, user):
    """
    Test Order Type Create
    """

    add_charge = models.AdditionalCharge.objects.create(
        organization=organization,
        order=order,
        charge=accessorial_charge,
        unit=1,
        entered_by=user,
    )

    assert add_charge is not None
    assert add_charge.order == order
    assert add_charge.charge == accessorial_charge
    assert add_charge.unit == 1
    assert add_charge.entered_by == user


def test_update(accessorial_charge, additional_charge):
    """
    Test order type update
    """

    add_charge = models.AdditionalCharge.objects.get(id=additional_charge.id)

    add_charge.charge = accessorial_charge
    add_charge.save()

    assert add_charge is not None
    assert add_charge.charge == accessorial_charge
    assert (
        add_charge.sub_total
        == accessorial_charge.charge_amount * additional_charge.unit
    )


def test_api_get(api_client):
    """
    Test get Additional Charge
    """
    response = api_client.get("/api/additional_charges/")
    assert response.status_code == 200


def test_api_get_by_id(
    api_client, additional_charge_api, order, user, accessorial_charge
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


def test_api_patch(api_client, additional_charge_api, order):
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


def test_api_delete(api_client, additional_charge_api):
    """
    Test Delete Additional Charge
    """
    response = api_client.delete(
        f"/api/additional_charges/{additional_charge_api.data['id']}/"
    )

    assert response.status_code == 204
    assert response.data is None
