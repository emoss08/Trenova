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
from typing import Any, Generator

import pytest
from django.urls import reverse

from accounting.models import GeneralLedgerAccount
from accounting.tests.factories import (
    DivisionCodeFactory,
    GeneralLedgerAccountFactory,
    RevenueCodeFactory,
)

pytestmark = pytest.mark.django_db


@pytest.fixture
def revenue_code() -> Generator[Any, Any, None]:
    """
    Revenue Code Fixture
    """
    yield RevenueCodeFactory()


@pytest.fixture
def general_ledger_account() -> Generator[Any, Any, None]:
    """
    Expense Account Fixture
    """
    yield GeneralLedgerAccountFactory()


@pytest.fixture
def revenue_code() -> Generator[Any, Any, None]:
    """
    Revenue Code Fixture
    """
    yield RevenueCodeFactory()


@pytest.fixture
def expense_account() -> Generator[Any, Any, None]:
    """
    Expense Code General Ledger Account Fixture
    """
    yield GeneralLedgerAccountFactory(
        account_type=GeneralLedgerAccount.AccountTypeChoices.EXPENSE
    )


@pytest.fixture
def revenue_account() -> Generator[Any, Any, None]:
    """
    Revenue Code General Ledger Account Fixture
    """
    yield GeneralLedgerAccountFactory(
        account_type=GeneralLedgerAccount.AccountTypeChoices.REVENUE
    )


@pytest.fixture
def division_code() -> Generator[Any, Any, None]:
    """
    Division Code Factory
    """
    yield DivisionCodeFactory()


@pytest.fixture
def cash_account() -> Generator[Any, Any, None]:
    """
    Cash Account from GL Account Factory
    """
    yield GeneralLedgerAccountFactory(
        account_classification=GeneralLedgerAccount.AccountClassificationChoices.CASH
    )


@pytest.fixture
def ap_account() -> Generator[Any, Any, None]:
    """
    AP Account from GL Account Factory
    """
    yield GeneralLedgerAccountFactory(
        account_classification=GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE
    )


@pytest.fixture
def gl_account_api(api_client, organization) -> Generator[Any, Any, None]:
    """
    GL account fixture for API
    """
    yield api_client.post(
        reverse("gl-accounts-list"),
        {
            "organization": str(organization),
            "account_number": "1234-1234-1234-1234",
            "account_type": GeneralLedgerAccount.AccountTypeChoices.REVENUE,
            "description": "Test General Ledger Account",
        },
        format="json",
    )

@pytest.fixture
def division_code_api(api_client, organization) -> Generator[Any, Any, None]:
    """
    Division Code API Fixture
    """
    yield api_client.post(
        reverse("division-codes-list"),
        {
            "organization": str(organization),
            "is_active": True,
            "code": "Test",
            "description": "Test Description",
        },
        format="json",
    )

@pytest.fixture
def revenue_code_api(api_client, organization) -> Generator[Any, Any, None]:
    """
    Revenue Code API Fixture
    """
    yield api_client.post(
        reverse("revenue-codes-list"),
        {
            "organization": str(organization),
            "code": "Test",
            "description": "Test Description",
        },
        format="json",
    )