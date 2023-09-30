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

from organization.models import Organization
from shipment import models

pytestmark = pytest.mark.django_db


def test_list(organization: Organization) -> None:
    """
    Test Shipment Control is created when the organization is
    created. This process happens via signals.
    """

    assert organization.shipment_control is not None


def test_update(organization: Organization) -> None:
    """
    Test shipment type update
    """

    shipment_control = models.ShipmentControl.objects.get(organization=organization)

    shipment_control.auto_rate_shipments = False
    shipment_control.calculate_distance = True
    shipment_control.enforce_customer = True
    shipment_control.enforce_rev_code = True

    shipment_control.save()

    assert shipment_control is not None
    assert not shipment_control.auto_rate_shipments
    assert shipment_control.calculate_distance
    assert shipment_control.enforce_customer
    assert shipment_control.enforce_rev_code
