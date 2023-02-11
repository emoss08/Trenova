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

import uuid

import pytest
from django.core.exceptions import ValidationError
from pydantic import BaseModel

from accounting import models
from accounting.models import GeneralLedgerAccount
from accounting.tests.factories import GeneralLedgerAccountFactory

pytestmark = pytest.mark.django_db


class DivisionCodeBase(BaseModel):
    """
    Division Code Base Schema
    """

    organization_id: uuid.UUID
    is_active: bool
    code: str
    description: str
    cash_account_id: uuid.UUID | None
    ap_account_id: uuid.UUID | None
    expense_account_id: uuid.UUID | None


class DivisionCodeCreate(DivisionCodeBase):
    """
    Division Code Create Schema
    """

    pass


class DivisionCodeUpdate(DivisionCodeBase):
    """
    Division Code Update Schema
    """

    id: uuid.UUID


def test_create_schema() -> None:
    """
    Test Division Code Creation Schema
    """
    div_code_create = DivisionCodeCreate(
        organization_id=uuid.uuid4(),
        is_active=True,
        code="NEW",
        description="Test Description",
        cash_account_id=uuid.uuid4(),
        ap_account_id=uuid.uuid4(),
        expense_account_id=uuid.uuid4(),
    )

    div_code = div_code_create.dict()

    assert div_code is not None
    assert div_code["is_active"] is True
    assert div_code["code"] == "NEW"
    assert div_code["description"] == "Test Description"


def test_update_schema() -> None:
    """
    Test Division Code update Schema
    """

    div_code_update = DivisionCodeUpdate(
        id=uuid.uuid4(),
        organization_id=uuid.uuid4(),
        is_active=True,
        code="FOOB",
        description="Test Description",
        cash_account_id=uuid.uuid4(),
        ap_account_id=uuid.uuid4(),
        expense_account_id=uuid.uuid4(),
    )

    div_code = div_code_update.dict()

    assert div_code is not None
    assert div_code["code"] == "FOOB"
    assert div_code["is_active"] is True
    assert div_code["description"] == "Test Description"
    assert div_code["id"] is not None
    assert div_code["organization_id"] is not None
    assert div_code["cash_account_id"] is not None
    assert div_code["ap_account_id"] is not None
    assert div_code["expense_account_id"] is not None


def test_delete_schema() -> None:
    """
    Test Division Code delete Schema
    """
    division_codes = [
        DivisionCodeBase(
            organization_id=uuid.uuid4(),
            is_active=True,
            code="NEW1",
            description="Test Description 1",
            cash_account_id=uuid.uuid4(),
            ap_account_id=uuid.uuid4(),
            expense_account_id=uuid.uuid4(),
        ),
        DivisionCodeBase(
            organization_id=uuid.uuid4(),
            is_active=True,
            code="NEW2",
            description="Test Description 2",
            cash_account_id=uuid.uuid4(),
            ap_account_id=uuid.uuid4(),
            expense_account_id=uuid.uuid4(),
        ),
    ]

    # Store the division codes in a list
    division_codes_store = division_codes.copy()

    # Delete a division code
    division_codes_store.pop(0)

    assert len(division_codes) == 2
    assert len(division_codes_store) == 1
    assert division_codes[0].code == "NEW1"
    assert division_codes_store[0].code == "NEW2"


def test_division_code_str_representation(division_code) -> None:
    """
    Test Division Code String Representation
    """
    assert str(division_code) == division_code.code


def test_division_code_get_absolute_url(division_code) -> None:
    """
    Test Division Code Get Absolute URL
    """
    assert (
        division_code.get_absolute_url() == f"/api/division_codes/{division_code.id}/"
    )


def test_division_code_clean_method_with_valid_data(division_code) -> None:
    """
    Test Division Code Clean Method with valid data
    """
    try:
        division_code.clean()
    except ValidationError:
        pytest.fail("clean method raised ValidationError unexpectedly")


def test_division_code_clean_method_with_invalid_cash_account(division_code) -> None:
    """
    Test Division Code Clean Method with invalid cash account
    """

    random_account = GeneralLedgerAccountFactory(
        account_type=GeneralLedgerAccount.AccountTypeChoices.REVENUE,
        account_classification=GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE,
    )

    division_code.cash_account = random_account
    with pytest.raises(ValidationError) as excinfo:
        division_code.clean()

    assert excinfo.value.message_dict == {
        "cash_account": ["Entered account is not an cash account. Please try again."]
    }


def test_division_code_clean_method_with_invalid_expense_account(division_code) -> None:
    """
    Test Division Code Clean Method with invalid expense account
    """
    random_account = GeneralLedgerAccountFactory(
        account_type=GeneralLedgerAccount.AccountTypeChoices.REVENUE,
        account_classification=GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE,
    )
    division_code.expense_account = random_account

    with pytest.raises(ValidationError) as excinfo:
        division_code.clean()

    assert excinfo.value.message_dict == {
        "expense_account": [
            "Entered account is not an expense account. Please try again."
        ]
    }


def test_division_code_clean_method_with_invalid_ap_account(
    division_code, expense_account
) -> None:
    """
    Test Division Code Clean Method with invalid ap account
    """
    division_code.ap_account = expense_account
    with pytest.raises(ValidationError) as excinfo:
        division_code.clean()

    assert excinfo.value.message_dict == {
        "ap_account": [
            "Entered account is not an accounts payable account. Please try again."
        ]
    }


def test_list(division_code) -> None:
    """
    Test Division Code List
    """
    assert division_code is not None


def test_create(organization, expense_account, cash_account) -> None:
    """
    Test Division Code Creation
    """
    random_account = GeneralLedgerAccountFactory(
        account_type=GeneralLedgerAccount.AccountTypeChoices.REVENUE,
        account_classification=GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE,
    )

    div_code = models.DivisionCode.objects.create(
        organization=organization,
        is_active=True,
        code="NEW",
        description="Test Description",
        cash_account=cash_account,
        ap_account=random_account,
        expense_account=expense_account,
    )

    assert div_code is not None
    assert div_code.is_active is True
    assert div_code.code == "NEW"
    assert div_code.description == "Test Description"


def test_update(division_code) -> None:
    """
    Test Division Code update
    """

    div_code = models.DivisionCode.objects.filter(id=division_code.id).get()
    div_code.code = "FOOB"
    div_code.save()

    assert div_code is not None
    assert div_code.code == "FOOB"


def test_api_get(api_client) -> None:
    """
    Test get Division Code
    """

    response = api_client.get("/api/division_codes/")
    assert response.status_code == 200


def test_api_get_by_id(api_client, organization, division_code_api) -> None:
    """
    Test get Division Code by ID
    """

    response = api_client.get(f"/api/division_codes/{division_code_api.data['id']}/")

    assert response.status_code == 200
    assert response.data["is_active"] is True
    assert response.data["code"] == "Test"
    assert response.data["description"] == "Test Description"


def test_api_put(api_client, organization, division_code_api) -> None:
    """
    Test put Division Code
    """

    cash_account_data = api_client.post(
        "/api/gl_accounts/",
        {
            "is_active": True,
            "account_number": "7000-0000-0000-0000",
            "description": "Foo bar",
            "account_type": "ASSET",
            "account_classification": "CASH",
        },
        format="json",
    )

    response = api_client.put(
        f"/api/division_codes/{division_code_api.data['id']}/",
        {
            "code": "foob",
            "is_active": False,
            "description": "Another Description",
            "cash_account": f"{cash_account_data.data['id']}",
        },
        format="json",
    )

    assert response.status_code == 200
    assert response.data["code"] == "foob"
    assert response.data["is_active"] is False
    assert response.data["description"] == "Another Description"


def test_api_delete(api_client, organization, division_code_api) -> None:
    """
    Test Delete Division Code
    """

    response = api_client.delete(f"/api/division_codes/{division_code_api.data['id']}/")
    assert response.status_code == 204
    assert response.data is None
