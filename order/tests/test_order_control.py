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

from order import models
from organization.models import Organization

pytestmark = pytest.mark.django_db

def test_list(organization: Organization) -> None:
    """
    Test Order Control is created when the organization is
    created. This process happens via signals.
    """

    assert organization.order_control is not None

def test_update(organization: Organization) -> None:
    """
    Test order type update
    """

    order_control = models.OrderControl.objects.get(organization=organization)

    order_control.auto_rate_orders = False
    order_control.calculate_distance = True
    order_control.enforce_customer = True
    order_control.enforce_rev_code = True

    order_control.save()

    assert order_control is not None
    assert not order_control.auto_rate_orders
    assert order_control.calculate_distance
    assert order_control.enforce_customer
    assert order_control.enforce_rev_code
