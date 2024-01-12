"""
COPYRIGHT 2022 Trenova

This file is part of Trenova.

Trenova is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Trenova is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Trenova.  If not, see <https://www.gnu.org/licenses/>.
"""

# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
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

from typing import Any

from django.db.models.base import ModelBase

from integration.models import Integration


class IntegrationBase:
    """
    Blank for now.
    """

    model: type[ModelBase]
    headers: dict[str, Any] = {}
    integration: Integration

    def _check(self):
        """Checks to make sure that the type of the global variables are correct.

        Raises:
            TypeError: If the model is not a subclass of django.db.models.base.ModelBase.
            TypeError: If the headers is not a dictionary.
        """
        if not isinstance(self.model, ModelBase):
            raise TypeError(
                f"""{self.__class__.__name__}.model must be a subclass of ModelBase,
                 not {type(self.model)}"""
            )

        if not isinstance(self.headers, dict):
            raise TypeError(
                f"{self.__class__.__name__}.headers must be a dictionary, not {type(self.headers)}"
            )
