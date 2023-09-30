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
from django.core.management import call_command
from django_celery_beat.models import IntervalSchedule

from organization import models

pytestmark = pytest.mark.django_db


def test_organization_creation(organization: models.Organization) -> None:
    """
    Test organization creation
    """
    assert organization is not None


def test_organization_update(organization: models.Organization) -> None:
    """
    Test organization update
    """
    organization.name = "New Name"
    organization.scac_code = "NEW"
    organization.save()
    assert organization.name == "New Name"
    assert organization.scac_code == "NEW"


def test_shipment_control_creation(organization: models.Organization) -> None:
    """
    Test dispatch control is created from create_shipment_control post_save signal
    """
    assert organization.shipment_control.auto_rate_shipments is True
    assert organization.shipment_control.organization == organization


def test_billing_control_hook(organization: models.Organization) -> None:
    """
    Test that the billing control hook is created when a new organization is
    created.
    """
    assert organization.billing_control is not None


def test_shipment_control_hook(organization: models.Organization) -> None:
    """
    Test that the Shipment Control hook is created when a new organization is
    created.
    """
    assert organization.shipment_control is not None


def test_dispatch_control_hook(organization: models.Organization) -> None:
    """
    Test that the dispatch control hook is created when a new organization is
    created.
    """
    assert organization.dispatch_control is not None


def test_depot_creation(depot: models.Depot) -> None:
    """
    Test depot creation
    """
    assert depot is not None


def test_depot_update(depot: models.Depot) -> None:
    """
    Test depot update
    """
    depot.name = "New Name"
    depot.save()
    assert depot.name == "New Name"


def test_depot_organization(depot: models.Depot) -> None:
    """
    Test dispatch control is created from create_depot_detail post_save signal
    """
    assert depot.details.organization == depot.organization


def test_depot_details_hook(depot: models.Depot) -> None:
    """
    Test that the depot details hook is created when a new depot is
    created.
    """
    assert depot.details is not None


def test_create_celery_beat_configurations_command() -> None:
    """
    Test that the create_celery_beat_configurations command creates
    configurations.
    """
    # Ensure there are no initial configurations
    assert IntervalSchedule.objects.count() == 0

    # Call the command to create configurations
    call_command("setupcelerybeat")  # TODO(WOLFRED): Add call count to this test.

    # Check that configurations have been created
    assert IntervalSchedule.objects.count() > 0
