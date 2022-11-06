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
from django.core.exceptions import ValidationError
from django.utils.translation import gettext_lazy as _


def validate_org_timezone(value: str) -> None:
    """Validate that the timezone is valid.

    Ensure that the timezone put in by the user is a valid timezone.

    Args:
        value (str):

    Returns:
        None

    Raises:
        ValidationError: If the timezone is not valid.
    """
    import pytz

    try:
        pytz.timezone(value)
    except pytz.exceptions.UnknownTimeZoneError:
        raise ValidationError(
            _("%(value)s is not a valid timezone"),
            params={"value": value},
        )


def validate_org_time_format(value: str) -> None:
    """Validate that the time format is valid.

    # NOTE: NO LONGER USED ,but leaving for migration purposes

    Ensure that the time format put in by the user is a valid time format.

    Args:
        value (str):

    Returns:
        None

    Raises:
        ValidationError: If the time format is not valid.
    """

    time_formats: list[str] = [
        "HH:mm",
        "HH:mm:ss",
    ]

    if value not in time_formats:
        raise ValidationError(
            _("%(value)s is not a valid time format"),
            params={"value": value},
        )
