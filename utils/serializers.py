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

from typing import Any, TypeVar

from django.db.models import Model
from rest_framework import serializers

_MT = TypeVar("_MT", bound=Model)
_M = TypeVar("_M", Model, Any)

class GenericSerializer(serializers.ModelSerializer):
    """
    Generic Serializer
    """

    def create(self, validated_data: Any) -> _M:
        """ Create the object

        Args:
            validated_data (dict[str, Any]): Validated data

        Returns:

        """
        validated_data["organization"] = self.context["request"].user.organization
        return super().create(validated_data)

    def update(self, instance: _MT, validated_data: Any) -> _MT:
        """
        Update
        """
        validated_data["organization"] = self.context["request"].user.organization
        return super().update(instance, validated_data)

    def to_representation(self, instance: _MT) -> dict[str, Any]:
        """
        To representation
        """
        return super().to_representation(instance)