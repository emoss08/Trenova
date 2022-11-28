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

from organization.factories import OrganizationFactory


@pytest.fixture()
def organization():
    """
    Organization fixture
    """
    return OrganizationFactory()


@pytest.mark.django_db
def test_organization_creation(organization):
    """
    Test organization creation
    """
    assert organization is not None


@pytest.mark.django_db
def test_organization_update(organization):
    """
    Test organization update
    """
    organization.name = "New Name"
    organization.scac_code = "NEW"
    organization.save()
    assert organization.name == "New Name"
    assert organization.scac_code == "NEW"


@pytest.mark.django_db
def test_dispatch_control_creation(organization):
    """
    Test dispatch control is created from
    create_dispatch_control post_save signal
    """
    assert organization.dispatch_control.driver_assign is True
    assert organization.dispatch_control.organization == organization


@pytest.mark.django_db
def test_order_control_creation(organization):
    """
    Test dispatch control is created from
    create_order_control post_save signal
    """
    assert organization.order_control.auto_rate_orders is True
    assert organization.order_control.organization == organization
