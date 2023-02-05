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

pytestmark = pytest.mark.django_db


class TestOrganization:
    """
    Class to test Organization
    """

    def test_organization_creation(self, organization):
        """
        Test organization creation
        """
        assert organization is not None

    def test_organization_update(self, organization):
        """
        Test organization update
        """
        organization.name = "New Name"
        organization.scac_code = "NEW"
        organization.save()
        assert organization.name == "New Name"
        assert organization.scac_code == "NEW"

    def test_order_control_creation(self, organization):
        """
        Test dispatch control is created from create_order_control post_save signal
        """
        assert organization.order_control.auto_rate_orders is True
        assert organization.order_control.organization == organization

    def test_billing_control_hook(self, organization) -> None:
        """
        Test that the billing control hook is created when a new organization is
        created.
        """
        assert organization.billing_control is not None

    def test_order_control_hook(self, organization) -> None:
        """
        Test that the order control hook is created when a new organization is
        created.
        """
        assert organization.order_control is not None

    def test_dispatch_control_hook(self, organization) -> None:
        """
        Test that the dispatch control hook is created when a new organization is
        created.
        """
        assert organization.dispatch_control is not None


class TestDepot:
    """
    Class to test Depot
    """

    def test_depot_creation(self, depot) -> None:
        """
        Test depot creation
        """
        assert depot is not None

    def test_depot_update(self, depot) -> None:
        """
        Test depot update
        """
        depot.name = "New Name"
        depot.save()
        assert depot.name == "New Name"

    def test_depot_organization(self, depot) -> None:
        """
        Test dispatch control is created from create_depot_detail post_save signal
        """
        assert depot.details.organization == depot.organization

    def test_depot_details_hook(self, depot) -> None:
        """
        Test that the depot details hook is created when a new depot is
        created.
        """
        assert depot.details is not None
