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

import pytest
from django.core.exceptions import ValidationError

pytestmark = pytest.mark.django_db


def test_auto_bill_criteria_required_when_auto_bill_true(organization):
    """
    Test if `auto_bill_orders` is true & `auto_bill_criteria` is blank
    that a ValidationError is thrown.
    """
    billing_control = organization.billing_control

    with pytest.raises(ValidationError) as excinfo:
        billing_control.auto_bill_orders = True
        billing_control.auto_bill_criteria = None
        billing_control.full_clean()

    assert excinfo.value.message_dict["auto_bill_criteria"] == [
        "Auto Billing criteria is required when `Auto Bill Orders` is on. Please try again."
    ]


def test_auto_bill_criteria_choices_is_invalid(organization):
    """
    Test when passing invalid choice to `auto_bill_criteria`
    that a ValidationError is thrown.
    """

    billing_control = organization.billing_control

    with pytest.raises(ValidationError) as excinfo:
        billing_control.auto_bill_criteria = "invalid"
        billing_control.full_clean()

    assert excinfo.value.message_dict["auto_bill_criteria"] == [
        "Value 'invalid' is not a valid choice."
    ]
