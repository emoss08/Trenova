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

from customer.factories import CustomerFactory

pytestmark = pytest.mark.django_db

# TODO(Wolfred): I didn't realize I had literally no tests for this.... Let's do that add some point.


@pytest.fixture
def customer() -> Generator[Any, Any, None]:
    """
    Customer fixture
    """
    yield CustomerFactory()


def test_customer_creation(customer) -> None:
    """
    Test customer creation
    """
    assert customer is not None


def test_customer_update(customer) -> None:
    """
    Test customer update
    """
    customer.name = "New name"
    customer.save()
    assert customer.name == "New name"


def test_customer_code_exists(customer) -> None:
    """
    Test customer code is added from set_code_before_create BEFORE_CREATE hook
    """
    assert customer.code is not None
