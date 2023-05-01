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
from django.core.exceptions import ValidationError

from organization.models import Organization

pytestmark = pytest.mark.django_db


def test_dispatch_control_creation(organization: Organization) -> None:
    """
    Test dispatch control is created from
    create_dispatch_control post_save signal
    """
    assert organization.dispatch_control.driver_assign is True
    assert organization.dispatch_control.organization == organization


def test_service_incident_control_choices(organization: Organization) -> None:
    """
    Test Service incident control choices throws ValidationError
    when the passed choice is not valid.
    """
    with pytest.raises(ValidationError) as excinfo:
        organization.dispatch_control.record_service_incident = "invalid"
        organization.dispatch_control.full_clean()

    assert excinfo.value.message_dict["record_service_incident"] == [
        "Value 'invalid' is not a valid choice."
    ]
