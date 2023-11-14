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

from accounting import models, services
from accounting.tests.factories import GeneralLedgerAccountFactory, RevenueCodeFactory
from organization.models import Organization
from shipment.tests.factories import ShipmentFactory

pytestmark = pytest.mark.django_db


def test_create_transaction_from_shipment(organization: Organization) -> None:
    """Test creating Financial Transaction for shipment without any issues that would cause reconciliation.

    Args:
        organization (Organization): Organization instance

    Returns:
        None: This function does not return anything.
    """
    gl_account = GeneralLedgerAccountFactory(account_type="REVENUE")
    revenue_code = RevenueCodeFactory(revenue_account=gl_account)

    organization.accounting_control.default_revenue_account = gl_account
    organization.accounting_control.save()

    # Add user to

    organization.billing_control.shipment_transfer_criteria = "READY_TO_BILL"
    organization.billing_control.save()

    shipment = ShipmentFactory(
        organization=organization, revenue_code=revenue_code, ready_to_bill=True
    )

    services.TransactionService.create_transaction_from_shipment(shipment=shipment)

    transaction: models.FinancialTransaction = models.FinancialTransaction.objects.get(
        shipment=shipment
    )

    transaction.refresh_from_db()

    assert transaction
    assert transaction.amount == round(shipment.sub_total, 2)
    assert transaction.general_ledger_account == gl_account
    assert transaction.status == models.TransactionStatusChoices.PENDING
    assert transaction.transaction_type == models.TransactionTypeChoices.DEBIT
