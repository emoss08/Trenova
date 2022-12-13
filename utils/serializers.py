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

from accounts.models import Token
from organization.models import Organization

_MT = TypeVar("_MT", bound=Model)
_M = TypeVar("_M", Model, Any)


class GenericSerializer(serializers.ModelSerializer):
    """
    Generic Serializer. This works when the serializer
    doesn't have nested serializers.
    """

    read_only_fields = [
        "organization",
        "id",
        "created",
        "modified",
    ]

    def _get_organization(self) -> Organization:
        """Get the organization from the request

        Returns:
            str: Organization
        """

        if self.context["request"].user.is_authenticated:
            return self.context["request"].user.organization
        else:
            token = (
                self.context["request"].META.get("HTTP_AUTHORIZATION", "").split(" ")[1]
            )
            return Token.objects.get(key=token).user.organization

    def create(self, validated_data: Any) -> _M:
        """Create the object

        Args:
            validated_data (dict[str, Any]): Validated data

        Returns:
            _M: Created object
        """

        organization: Organization = self._get_organization()
        validated_data["organization"] = organization

        return super().create(validated_data)

    def update(self, instance: _MT, validated_data: Any) -> _MT:
        """Update the object

        Args:
            instance (_MT): Instance of the model
            validated_data (Any): Validated data

        Returns:
            _MT: Updated instance
        """

        organization: Organization = self._get_organization()
        validated_data["organization"] = organization

        return super().update(instance, validated_data)
