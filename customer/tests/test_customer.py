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

from customer.factories import CustomerFactory


@pytest.fixture()
def customer():
    """
    Customer fixture
    """
    return CustomerFactory()


@pytest.mark.django_db
def test_customer_creation(customer):
    """
    Test customer creation
    """
    assert customer is not None


@pytest.mark.django_db
def test_customer_update(customer):
    """
    Test customer update
    """
    customer.name = "New name"
    customer.save()
    assert customer.name == "New name"


@pytest.mark.django_db
def test_customer_code_exists(customer):
    """
    Test customer code is added from
    generate_customer_code pre_save signal
    """
    assert customer.code is not None


@pytest.mark.django_db
def test_customer_billing_profile_creation(customer):
    """
    Test customer billing profile is created from
    create_customer_billing_profile post_save signal
    """
    assert customer.billing_profile is not None
