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

import uuid

import pytest
from django.core.exceptions import ValidationError
from pydantic import BaseModel
from rest_framework.response import Response
from rest_framework.test import APIClient

from accounting import models
from accounting.models import DivisionCode, GeneralLedgerAccount
from accounting.tests.factories import GeneralLedgerAccountFactory
from organization.models import Organization, BusinessUnit

pytestmark = pytest.mark.django_db


class DivisionCodeBase(BaseModel):
    """
    Division Code Base Schema
    """

    organization_id: uuid.UUID
    status: str
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
        status="A",
        code="NEW",
        description="Test Description",
        cash_account_id=uuid.uuid4(),
        ap_account_id=uuid.uuid4(),
        expense_account_id=uuid.uuid4(),
    )

    div_code = div_code_create.model_dump()

    assert div_code is not None
    assert div_code["status"] == "A"
    assert div_code["code"] == "NEW"
    assert div_code["description"] == "Test Description"


def test_update_schema() -> None:
    """
    Test Division Code update Schema
    """

    div_code_update = DivisionCodeUpdate(
        id=uuid.uuid4(),
        organization_id=uuid.uuid4(),
        status="A",
        code="FOOB",
        description="Test Description",
        cash_account_id=uuid.uuid4(),
        ap_account_id=uuid.uuid4(),
        expense_account_id=uuid.uuid4(),
    )

    div_code = div_code_update.model_dump()

    assert div_code is not None
    assert div_code["code"] == "FOOB"
    assert div_code["status"] == "A"
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
            status="A",
            code="NEW1",
            description="Test Description 1",
            cash_account_id=uuid.uuid4(),
            ap_account_id=uuid.uuid4(),
            expense_account_id=uuid.uuid4(),
        ),
        DivisionCodeBase(
            organization_id=uuid.uuid4(),
            status="A",
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


def test_division_code_str_representation(division_code: DivisionCode) -> None:
    """
    Test Division Code String Representation
    """
    assert str(division_code) == division_code.code


def test_division_code_get_absolute_url(division_code: DivisionCode) -> None:
    """
    Test Division Code Get Absolute URL
    """
    assert (
        division_code.get_absolute_url() == f"/api/division_codes/{division_code.id}/"
    )


def test_division_code_clean_method_with_valid_data(
    division_code: DivisionCode,
) -> None:
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


def test_division_code_clean_method_with_invalid_expense_account(
    division_code: DivisionCode,
) -> None:
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
    division_code: DivisionCode, expense_account: GeneralLedgerAccount
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


def test_list(division_code: DivisionCode) -> None:
    """
    Test Division Code List
    """
    assert division_code is not None


def test_create(
    business_unit: BusinessUnit,
    organization: Organization,
    expense_account: GeneralLedgerAccount,
    cash_account: GeneralLedgerAccount,
) -> None:
    """
    Test Division Code Creation
    """
    random_account = GeneralLedgerAccountFactory(
        account_type=GeneralLedgerAccount.AccountTypeChoices.REVENUE,
        account_classification=GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE,
    )

    div_code = models.DivisionCode.objects.create(
        business_unit=business_unit,
        organization=organization,
        status="A",
        code="NEW",
        description="Test Description",
        cash_account=cash_account,
        ap_account=random_account,
        expense_account=expense_account,
    )

    assert div_code is not None
    assert div_code.status == "A"
    assert div_code.code == "NEW"
    assert div_code.description == "Test Description"


def test_update(division_code: DivisionCode) -> None:
    """
    Test Division Code update
    """

    div_code = models.DivisionCode.objects.filter(id=division_code.id).get()
    div_code.code = "FOOB"
    div_code.save()

    assert div_code is not None
    assert div_code.code == "FOOB"


def test_api_get(api_client: APIClient) -> None:
    """
    Test get Division Code
    """

    response = api_client.get("/api/division_codes/")
    assert response.status_code == 200


def test_api_get_by_id(
    api_client: APIClient, organization: Organization, division_code_api: Response
) -> None:
    """
    Test get Division Code by ID
    """

    response = api_client.get(f"/api/division_codes/{division_code_api.data['id']}/")

    assert response.status_code == 200
    assert response.data["status"] == "A"
    assert response.data["code"] == "Test"
    assert response.data["description"] == "Test Description"


def test_api_put(
    api_client: APIClient, organization: Organization, division_code_api: Response
) -> None:
    """
    Test put Division Code
    """

    cash_account_data = api_client.post(
        "/api/gl_accounts/",
        {
            "organization": organization.id,
            "status": "A",
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
            "organization": organization.id,
            "code": "foob",
            "status": "A",
            "description": "Another Description",
            "cash_account": f"{cash_account_data.data['id']}",
        },
        format="json",
    )

    assert response.status_code == 200
    assert response.data["code"] == "foob"
    assert response.data["status"] == "A"
    assert response.data["description"] == "Another Description"


def test_api_delete(
    api_client: APIClient, organization: Organization, division_code_api: Response
) -> None:
    """
    Test Delete Division Code
    """

    response = api_client.delete(f"/api/division_codes/{division_code_api.data['id']}/")
    assert response.status_code == 204
    assert response.data is None
