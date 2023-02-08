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
from django.core.exceptions import ValidationError

from accounting import models

pytestmark = pytest.mark.django_db


def test_list(revenue_code) -> None:
    """
    Test Revenue code list
    """
    assert revenue_code is not None


def test_create(revenue_code, expense_account, revenue_account) -> None:
    """
    Test Revenue code creation
    """

    rev_code = models.RevenueCode.objects.create(
        organization=expense_account.organization,
        code="TEST",
        description="Another Description",
        expense_account=expense_account,
        revenue_account=revenue_account,
    )

    assert rev_code is not None
    assert rev_code.code == "TEST"
    assert rev_code.description == "Another Description"


def test_update(revenue_code) -> None:
    """
    Test Revenue code update
    """

    rev_code = models.RevenueCode.objects.get(id=revenue_code.id)
    rev_code.code = "FOOB"
    rev_code.save()

    assert rev_code is not None
    assert rev_code.code == "FOOB"


def test_expense_account(revenue_code, revenue_account) -> None:
    """
    Test Whether the validation error is thrown if an account other than an expense account
    is passed.
    """

    with pytest.raises(
        ValidationError, match="Entered account is not an expense account."
    ):
        revenue_code.expense_account = revenue_account
        revenue_code.full_clean()


def test_revenue_account(revenue_code, expense_account) -> None:
    """
    Test Whether the validation error is thrown if an account other than an expense account
    is passed.
    """

    with pytest.raises(
        ValidationError, match="Entered account is not a revenue account."
    ):
        revenue_code.revenue_account = expense_account
        revenue_code.full_clean()


def test_api_get(api_client) -> None:
    """
    Test Revenue code API get
    """
    response = api_client.get("/api/revenue_codes/")
    assert response.status_code == 200


def test_api_get_by_id(api_client, revenue_code_api) -> None:
    """
    Test Revenue code API get by id
    """
    response = api_client.get(f"/api/revenue_codes/{revenue_code_api.data['id']}/")
    assert response.status_code == 200
    assert response.data["code"] == "TEST"
    assert response.data["description"] == "Test Description"
    assert response.data["expense_account"] is None
    assert response.data["revenue_account"] is None
