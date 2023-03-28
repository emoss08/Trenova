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

from django.core.exceptions import ValidationError
from django.utils.translation import gettext_lazy as _

from dispatch.models import DispatchControl
from worker.models import WorkerProfile


def validate_worker_regulatory_information(value: WorkerProfile) -> None:
    """Validates that if the dispatch control has enforced regulatory validation set to true
    then require the user to enter license number, state, expiration date
     and endorsement information in the worker.

    Args:
        value (WorkerProfile): Instance of worker profile

    Returns:
        None: This function does not return anything.

    Raises:
        ValidationError: Validate the worker regulatory information.
    """
    dispatch_control: DispatchControl | None = DispatchControl.objects.filter(
        organization=value.worker.organization
    ).first()
    fields = {
        "license_number": _(
            "Organization has regulatory check enabled. Please enter a license number."
        ),
        "license_state": _(
            "Organization has regulatory check enabled. Please enter a license state."
        ),
        "license_expiration_date": _(
            "Organization has regulatory check enabled."
            " Please enter a license expiration date."
        ),
        "endorsements": _(
            "Organization has regulatory check enabled. Please enter endorsements."
        ),
        "physical_due_date": _(
            "Organization has regulatory check enabled. Please enter a physical due date."
        ),
        "medical_cert_date": _(
            "Organization has regulatory check enabled. Please enter a medical"
            " certificate date."
        ),
        "mvr_due_date": _(
            "Organization has regulatory check enabled. Please enter a MVR due date."
        ),
    }
    if dispatch_control and dispatch_control.regulatory_check:
        if errors := {
            field: error for field, error in fields.items() if not getattr(value, field)
        }:
            raise ValidationError(errors)
