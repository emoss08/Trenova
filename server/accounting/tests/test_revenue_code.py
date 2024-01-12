# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
from django.core.exceptions import ValidationError
from rest_framework import status
from rest_framework.response import Response
from rest_framework.test import APIClient

from accounting import models
from accounting.models import GeneralLedgerAccount, RevenueCode
from organization.models import BusinessUnit, Organization

pytestmark = pytest.mark.django_db


def test_list(revenue_code: RevenueCode) -> None:
    """
    Test Revenue code list
    """
    assert revenue_code is not None


def test_create(
    revenue_code: RevenueCode,
    organization: Organization,
    business_unit: BusinessUnit,
    expense_account: GeneralLedgerAccount,
    revenue_account: GeneralLedgerAccount,
) -> None:
    """
    Test Revenue code creation
    """

    rev_code = models.RevenueCode.objects.create(
        organization=organization,
        business_unit=business_unit,
        code="TEST",
        description="Another Description",
        expense_account=expense_account,
        revenue_account=revenue_account,
    )

    assert rev_code is not None
    assert rev_code.code == "TEST"
    assert rev_code.description == "Another Description"


def test_update(revenue_code: RevenueCode) -> None:
    """
    Test Revenue code update
    """

    rev_code = models.RevenueCode.objects.get(id=revenue_code.id)
    rev_code.code = "FOOB"
    rev_code.save()

    assert rev_code is not None
    assert rev_code.code == "FOOB"


def test_expense_account(
    revenue_code: RevenueCode, revenue_account: GeneralLedgerAccount
) -> None:
    """
    Test Whether the validation error is thrown if an account other than an expense account
    is passed.
    """

    with pytest.raises(ValidationError) as excinfo:
        revenue_code.expense_account = revenue_account
        revenue_code.full_clean()

    assert (
        excinfo.value.message_dict["expense_account"][0]
        == "Entered account is a REVENUE account, not a expense account. Please try again."
    )


def test_revenue_account(
    revenue_code: RevenueCode, expense_account: GeneralLedgerAccount
) -> None:
    """
    Test Whether the validation error is thrown if an account other than an expense account
    is passed.
    """

    with pytest.raises(ValidationError) as excinfo:
        revenue_code.revenue_account = expense_account
        revenue_code.full_clean()

    assert (
        excinfo.value.message_dict["revenue_account"][0]
        == "Entered account is a EXPENSE account, not a revenue account. Please try again."
    )


def test_api_get(api_client: APIClient) -> None:
    """
    Test Revenue code API get
    """
    response = api_client.get("/api/revenue_codes/")
    assert response.status_code == 200


def test_api_get_by_id(api_client: APIClient, revenue_code_api: Response) -> None:
    """
    Test Revenue code API get by id
    """
    response = api_client.get(f"/api/revenue_codes/{revenue_code_api.data['id']}/")
    assert response.status_code == 200
    assert response.data["code"] == "TEST"
    assert response.data["description"] == "Test Description"
    assert response.data["expense_account"] is None
    assert response.data["revenue_account"] is None


def test_post_with_unique_code(api_client, revenue_code: models.RevenueCode) -> None:
    """
    Test posting a revenue code with the same code throws serializer.ValidationError.
    """
    revenue_code.code = "test"
    revenue_code.save()

    revenue_code.refresh_from_db()

    response = api_client.post(
        "/api/revenue_codes/",
        {
            "code": "test",
            "description": "Test Description",
            "organization": revenue_code.organization.id,
            "business_unit": revenue_code.business_unit.id,
        },
        format="json",
    )

    assert response.status_code == 400
    assert response.data["type"] == "validationError"
    assert (
        response.data["errors"][0]["detail"]
        == "Revenue Code with this `code` already exists. Please try again."
    )


def test_validate_code(api_client: APIClient, revenue_code: models.RevenueCode) -> None:
    """Test serializer Validation is thrown if the code given is not unique.

    Args:
        revenue_code (models.RevenueCode): Revenue Code Object

    Return:
        None: This function does not return anything.
    """

    response = api_client.post(
        "/api/revenue_codes/",
        {
            "code": revenue_code.code,
            "description": "Test Description",
            "organization": revenue_code.organization.id,
            "business_unit": revenue_code.business_unit.id,
        },
        format="json",
    )

    assert response.status_code == status.HTTP_400_BAD_REQUEST
    assert (
        response.data["errors"][0]["detail"]
        == "Revenue Code with this `code` already exists. Please try again."
    )
    assert response.data["errors"][0]["attr"] == "code"
