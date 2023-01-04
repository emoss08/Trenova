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
from typing import Type

from utils.serializers import GenericSerializer
from stops import models


class QualifierCodeSerializer(GenericSerializer):
    """A serializer class for the Qualifier Code model

    The `QualifierCodeSerializer` class provides default operations
    for creating, update and deleting Integration, as well as
    listing and retrieving them.
    """

    class Meta:
        """
        A class representing the metadata for the `QualifierCodeSerializer`
        class.
        """

        model: Type[models.QualifierCode] = models.QualifierCode


class StopSerializer(GenericSerializer):
    """A serializer class for the Qualifier Code model

    The `QualifierCodeSerializer` class provides default operations
    for creating, update and deleting Integration, as well as
    listing and retrieving them.
    """

    class Meta:
        """
        A class representing the metadata for the `StopSerializer`
        class.
        """

    model: Type[models.Stop]
