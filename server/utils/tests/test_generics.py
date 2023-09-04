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
from rest_framework.test import APIClient

from customer.api import ValidateCustomerInfoView
from utils import views

pytestmark = pytest.mark.django_db


class DummyViewWithoutSerializer(views.ValidateView):
    step_to_fields = {"dummy_step": ["dummy_field"]}


class DummyViewWithoutStepToFields(views.ValidateView):
    serializer_class = ValidateCustomerInfoView.serializer_class


def test_missing_serializer_class_raises_not_implemented_error() -> None:
    """Test that a NotImplementedError is raised when a view does not define a serializer_class.

    Returns:
        None: This function does not return anything.
    """
    data = {"step": "dummy_step", "dummy_field": "dummy_data"}

    with pytest.raises(NotImplementedError, match=r".*must define serializer_class"):
        DummyViewWithoutSerializer().post(APIClient().post("/", data))


def test_missing_step_to_fields_raises_not_implemented_error() -> None:
    """Test that a NotImplementedError is raised when a view does not define step_to_fields.

    Returns:
        None: This function does not return anything.
    """
    data = {"step": "dummy_step", "dummy_field": "dummy_data"}

    with pytest.raises(NotImplementedError, match=r".*must define step_to_fields"):
        DummyViewWithoutStepToFields().post(APIClient().post("/", data))
