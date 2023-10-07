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

from collections.abc import Generator
from typing import Any

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
        account_type=GeneralLedgerAccount.AccountTypeChoices.REVENUE,
        account_classification=GeneralLedgerAccount.AccountClassificationChoices.CASH,
    )


@pytest.fixture
def gl_account_api(api_client, organization) -> Generator[Any, Any, None]:
    """
    GL account fixture for API
    """
    yield api_client.post(
        reverse("gl-accounts-list"),
        {
            "organization": organization.id,
            "account_number": "1234-12",
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
            "organization": organization.id,
            "status": "A",
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
            "organization": organization.id,
            "code": "Test",
            "description": "Test Description",
        },
        format="json",
    )
