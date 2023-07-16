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
from django.urls import reverse
from pydantic import BaseModel
from rest_framework.response import Response
from rest_framework.test import APIClient

from accounting import models
from accounting.models import GeneralLedgerAccount
from accounting.tests.factories import GeneralLedgerAccountFactory
from organization.models import BusinessUnit, Organization

pytestmark = pytest.mark.django_db


class GeneralLedgerAccountBase(BaseModel):
    """
    Division Code Base Schema
    """

    organization_id: uuid.UUID
    status: str
    account_number: str
    description: str
    account_type: str
    cash_flow_type: str
    account_sub_type: str
    account_classification: str


class GeneralAccountLedgerAccountCreate(GeneralLedgerAccountBase):
    """
    Division Code Create Schema
    """

    pass


class GeneralAccountLedgerAccountUpdate(GeneralLedgerAccountBase):
    """
    Division Code Update Schema
    """

    id: uuid.UUID


def test_create_schema() -> None:
    """
    Test create schema
    """
    gl_account_create = GeneralAccountLedgerAccountCreate(
        organization_id=uuid.uuid4(),
        status="A",
        account_number="1234-1234-1234-1234",
        description="Description",
        account_type=models.GeneralLedgerAccount.AccountTypeChoices.ASSET,
        cash_flow_type=models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING,
        account_sub_type=models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET,
        account_classification=models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE,
    )

    gl_account = gl_account_create.model_dump()

    assert gl_account is not None
    assert gl_account["status"] == "A"
    assert gl_account["account_number"] == "1234-1234-1234-1234"
    assert gl_account["description"] == "Description"
    assert (
        gl_account["account_type"]
        == models.GeneralLedgerAccount.AccountTypeChoices.ASSET
    )
    assert (
        gl_account["cash_flow_type"]
        == models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING
    )
    assert (
        gl_account["account_sub_type"]
        == models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET
    )
    assert (
        gl_account["account_classification"]
        == models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE
    )


def test_update_schema() -> None:
    """
    Test General Ledger Account update Schema
    """

    gl_account_update = GeneralAccountLedgerAccountUpdate(
        id=uuid.uuid4(),
        organization_id=uuid.uuid4(),
        status="A",
        account_number="1234-1234-1234-1234",
        description="Description",
        account_type=models.GeneralLedgerAccount.AccountTypeChoices.ASSET,
        cash_flow_type=models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING,
        account_sub_type=models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET,
        account_classification=models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE,
    )

    gl_account = gl_account_update.model_dump()

    assert gl_account is not None
    assert gl_account["status"] == "A"
    assert gl_account["account_number"] == "1234-1234-1234-1234"
    assert gl_account["description"] == "Description"
    assert (
        gl_account["account_type"]
        == models.GeneralLedgerAccount.AccountTypeChoices.ASSET
    )
    assert (
        gl_account["cash_flow_type"]
        == models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING
    )
    assert (
        gl_account["account_sub_type"]
        == models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET
    )
    assert (
        gl_account["account_classification"]
        == models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE
    )


def test_delete_schema() -> None:
    """
    Test GL Account Delete Schema
    """

    gl_accounts = [
        GeneralLedgerAccountBase(
            organization_id=uuid.uuid4(),
            status="A",
            account_number="1234-1234-1234-1234",
            description="Description 1",
            account_type=models.GeneralLedgerAccount.AccountTypeChoices.ASSET,
            cash_flow_type=models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING,
            account_sub_type=models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET,
            account_classification=models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE,
        ),
        GeneralLedgerAccountBase(
            organization_id=uuid.uuid4(),
            status="A",
            account_number="1234-1234-1234-1235",
            description="Description 2",
            account_type=models.GeneralLedgerAccount.AccountTypeChoices.ASSET,
            cash_flow_type=models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING,
            account_sub_type=models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET,
            account_classification=models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE,
        ),
    ]

    # Store the GL Accounts in a list
    gl_account_store = gl_accounts.copy()

    # Delete the first GL Account
    gl_account_store.pop(0)

    assert len(gl_accounts) == 2
    assert len(gl_account_store) == 1
    assert gl_accounts[0].account_number == "1234-1234-1234-1234"
    assert gl_account_store[0].account_number == "1234-1234-1234-1235"


def test_gl_account_get_absolute_url(
    general_ledger_account: GeneralLedgerAccount,
) -> None:
    """
    Test GL Account Get Absolute URL
    """
    assert (
        general_ledger_account.get_absolute_url()
        == f"/api/gl_accounts/{general_ledger_account.id}/"
    )


def test_gl_account_clean_method_with_valid_data(
    general_ledger_account: GeneralLedgerAccount,
) -> None:
    """
    Test GL Account Clean Method with valid data
    """
    try:
        general_ledger_account.clean()
    except ValidationError:
        pytest.fail("clean method raised ValidationError unexpectedly")


def test_list(general_ledger_account: GeneralLedgerAccount) -> None:
    """
    Test general ledger account list
    """
    assert general_ledger_account is not None


def test_create(organization: Organization, business_unit: BusinessUnit) -> None:
    """
    Test general ledger account creation
    """
    gl_account = models.GeneralLedgerAccount.objects.create(
        organization=organization,
        business_unit=business_unit,
        account_number="1234-1234-1234-1234",
        account_type=models.GeneralLedgerAccount.AccountTypeChoices.ASSET,
        description="Another Description",
        cash_flow_type=models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING,
        account_sub_type=models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET,
        account_classification=models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE,
    )

    assert gl_account is not None
    assert gl_account.account_number == "1234-1234-1234-1234"


def test_update(general_ledger_account: GeneralLedgerAccount) -> None:
    """
    Test general ledger account update
    """
    general_ledger_account.account_number = "1234-1234-1234-1234"
    general_ledger_account.save()
    assert general_ledger_account.account_number == "1234-1234-1234-1234"


def test_api_get(api_client: APIClient) -> None:
    """
    Test get General Ledger accounts
    """
    response = api_client.get(reverse("gl-accounts-list"))
    assert response.status_code == 200


def test_api_get_by_id(api_client: APIClient, gl_account_api: Response) -> None:
    """
    Test get General Ledger account by ID
    """
    response = api_client.get(
        reverse(
            "gl-accounts-detail",
            kwargs={"pk": gl_account_api.data["id"]},
        )
    )
    assert response.status_code == 200
    assert response.data["account_number"] == gl_account_api.data["account_number"]
    assert response.data["account_type"] == gl_account_api.data["account_type"]
    assert response.data["description"] == gl_account_api.data["description"]


def test_api_put(
    api_client: APIClient, gl_account_api: Response, organization: Organization
) -> None:
    """
    Test put General Ledger Account
    """
    response = api_client.put(
        reverse(
            "gl-accounts-detail",
            kwargs={"pk": gl_account_api.data["id"]},
        ),
        {
            "organization": organization.id,
            "account_number": "2345-2345-2345-2345",
            "description": "Another Test Description",
            "account_type": f"{models.GeneralLedgerAccount.AccountTypeChoices.EXPENSE}",
            "cash_flow_type": f"{models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING}",
            "account_sub_type": f"{models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET}",
            "account_classification": f"{models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_RECEIVABLE}",
        },
    )
    assert response.status_code == 200
    assert response.data["account_number"] == "2345-2345-2345-2345"
    assert (
        response.data["account_type"]
        == models.GeneralLedgerAccount.AccountTypeChoices.EXPENSE
    )
    assert response.data["description"] == "Another Test Description"
    assert (
        response.data["cash_flow_type"]
        == models.GeneralLedgerAccount.CashFlowTypeChoices.FINANCING
    )
    assert (
        response.data["account_sub_type"]
        == models.GeneralLedgerAccount.AccountSubTypeChoices.CURRENT_ASSET
    )
    assert (
        response.data["account_classification"]
        == models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_RECEIVABLE
    )


def test_api_delete(api_client: APIClient, gl_account_api: Response) -> None:
    """
    Test delete general Ledger account
    """
    response = api_client.delete(
        reverse(
            "gl-accounts-detail",
            kwargs={"pk": gl_account_api.data["id"]},
        )
    )

    assert response.status_code == 204
    assert not response.data


def test_account_number(general_ledger_account: GeneralLedgerAccount) -> None:
    """
    Test Whether the validation error is thrown if the entered account_number value is not a
    regex match.
    """

    with pytest.raises(ValidationError) as excinfo:
        general_ledger_account.account_number = "00000-2323411-124141"
        general_ledger_account.full_clean()

    assert excinfo.value.message_dict["account_number"] == [
        "Account number must be in the format 0000-0000-0000-0000."
    ]


def test_unique_account_numer(general_ledger_account: GeneralLedgerAccount) -> None:
    """
    Test creating a General Ledger account with the same account number
    throws ValidationError.
    """
    general_ledger_account.account_number = "1234-1234-1234-1234"
    general_ledger_account.save()

    with pytest.raises(ValidationError) as excinfo:
        GeneralLedgerAccountFactory(account_number="1234-1234-1234-1234")

    assert excinfo.value.message_dict["account_number"] == [
        "An account with this account number already exists. Please try again."
    ]
