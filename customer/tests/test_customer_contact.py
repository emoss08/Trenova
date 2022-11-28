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

from customer.factories import CustomerContactFactory


@pytest.fixture()
def customer_contact():
    """
    Customer contact fixture
    """
    return CustomerContactFactory()


@pytest.mark.django_db
def test_customer_contact_creation(customer_contact):
    """
    Test customer contact creation
    """
    assert customer_contact is not None


@pytest.mark.django_db
def test_customer_contact_update(customer_contact):
    """
    Test customer contact update
    """
    customer_contact.name = "New name"
    customer_contact.save()
    assert customer_contact.name == "New name"


@pytest.mark.django_db
def test_customer_contact_payable_has_no_email(customer_contact):
    """
    Test customer contact payable has no email
    """
    customer_contact.email = ""
    customer_contact.save()

    with pytest.raises(
        ValidationError, match="Payable contact must have an email address"
    ):
        customer_contact.full_clean()
