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

import string

from django.core.exceptions import ValidationError
from django.utils.translation import gettext_lazy as _


def validate_org_timezone(value: str) -> None:
    """Validate that the timezone is valid.

    Ensure that the timezone put in by the user is a valid timezone.

    Args:
        value (str): The timezone to validate against.

    Returns:
        None: this function does not return anything.

    Raises:
        ValidationError: If the timezone is not valid.
    """
    import pytz

    try:
        pytz.timezone(value)
    except pytz.exceptions.UnknownTimeZoneError as e:
        raise ValidationError(
            _("%(value)s is not a valid timezone"),
            params={"value": value},
        ) from e


def validate_format_string(value: str) -> None:
    """
    Validate that the format string is valid.

    Args:
        value (str): The format string to validate against.

    Returns:
        None: This function does not return anything.

    Raises:
        ValidationError: If the format string is not valid.
    """
    try:
        list(string.Formatter().parse(value))
    except ValueError as e:
        raise ValidationError(
            _("%(value)s is not a valid format string"),
            params={"value": value},
        ) from e
