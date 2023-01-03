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

from typing import Dict, List, Type, TypeVar, Any

from asgiref.sync import async_to_sync
from django.db.models import Model
from django.db.models.base import ModelBase
from django.utils.datastructures import ImmutableList

import httpx

from integration.models import Integration

_M = TypeVar("_M", bound=Model)


class IntegrationBase:
    """
    Blank for now.
    """

    model: Type[ModelBase]
    headers: Dict[str, Any] = {}
    integration: Integration

    def __init__(self):
        """
        Initializes the IntegrationBase class.
        """
        self._check()

    def _check(self):
        """Checks to make sure that the type of the global variables are correct.

        Raises:
            TypeError: If the model is not a subclass of django.db.models.base.ModelBase.
            TypeError: If the headers is not a dictionary.
        """
        if not isinstance(self.model, ModelBase):
            raise TypeError(
                f"{self.__class__.__name__}.model must be a subclass of ModelBase, not {type(self.model)}"
            )

        if not isinstance(self.headers, Dict):
            raise TypeError(
                f"{self.__class__.__name__}.headers must be a dictionary, not {type(self.headers)}"
            )
