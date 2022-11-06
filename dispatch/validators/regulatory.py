# -*- coding: utf-8 -*-
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

from typing import Optional

from django.core.exceptions import ValidationError
from django.utils.translation import gettext_lazy as _

from dispatch.models import DispatchControl


def validate_worker_regulatory_information(value) -> None:
    """Validates that if the dispatch control has enforced regulatory validation set to true
    then require the user to enter license number, state, expiration date and endorsement information in the worker.

    Args:
        value (WorkerProfile): Instance of worker profile

    Raises:
        ValidationError: Validate the worker regulatory information.

    Typical
    """
    dispatch_control: Optional[DispatchControl] = DispatchControl.objects.filter(
        organization=value.organization
    ).first()
    errors: dict[str, str] = {}
    if dispatch_control and dispatch_control.regulatory_check:
        if not value.license_number:
            errors["license_number"] = _(
                "Organization has regulatory check enabled. Please enter a license number."
            )
        if not value.license_state:
            errors["license_state"] = _(
                "Organization has regulatory check enabled. Please enter a license state."
            )
        if not value.license_expiration_date:
            errors["license_expiration_date"] = _(
                "Organization has regulatory check enabled."
                " Please enter a license expiration date."
            )
        if not value.endorsements:
            errors["endorsements"] = _(
                "Organization has regulatory check enabled. Please enter endorsements."
            )
        if not value.physical_due_date:
            errors["physical_due_date"] = _(
                "Organization has regulatory check enabled. Please enter a physical due date."
            )
        if not value.medical_cert_date:
            errors["medical_cert_date"] = _(
                "Organization has regulatory check enabled. Please enter a medical certificate date."
            )
        if not value.mvr_due_date:
            errors["mvr_due_date"] = _(
                "Organization has regulatory check enabled. Please enter a MVR due date."
            )
    if errors:
        raise ValidationError(errors)
