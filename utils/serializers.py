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
from django.utils.functional import cached_property
from knox.models import AuthToken
from rest_framework import serializers

from organization.models import Organization

_MT = TypeVar("_MT", bound=Model)


class GenericSerializer(serializers.ModelSerializer):
    """
    Generic Serializer. This works when the serializer
    doesn't have nested serializers.
    """

    class Meta:
        """
        A class representing the metadata for the `GenericSerializer`
        class.
        """

        extra_fields: list[str]
        extra_read_only_fields: list[str]
        model: _MT  # type: ignore

    def __init__(self, *args: Any, **kwargs: Any) -> None:
        """Initialize the serializer

        Args:
            args (Any): Arguments
            kwargs (Any): Keyword arguments

        Returns:
            None
        """

        super().__init__(*args, **kwargs)
        self.set_fields()

    @cached_property
    def get_organization(self) -> Organization:
        """Get the organization from the request

        Returns:
            str: Organization
        """

        if self.context["request"].user.is_authenticated:
            return self.context["request"].user.organization
        token = self.context["request"].META.get("HTTP_AUTHORIZATION", "").split(" ")[1]
        return AuthToken.objects.get(token_key=token).user.organization

    def create(self, validated_data: Any) -> _MT:  # type: ignore
        """Create the object

        Args:
            validated_data (Any): Validated data

        Returns:
            _MT: Created object
        """

        organization: Organization = self.get_organization
        validated_data["organization"] = organization

        return super().create(validated_data)

    def update(self, instance: _MT, validated_data: Any) -> _MT:
        """Update the object

        Args:
            instance (_M): Instance of the model
            validated_data (Any): Validated data

        Returns:
            _MT: Updated instance
        """
        organization: Organization = self.get_organization
        validated_data["organization"] = organization

        return super().update(instance, validated_data)

    def set_fields(self) -> None:
        """Set the fields for the serializer

        Returns:
            None
        """

        read_only_field: tuple[str, ...] = ("organization", "created", "modified")

        original_fields = getattr(self.Meta, "fields", None)

        if original_fields is not None:
            fields = original_fields
        else:
            # If reverse=True, then relations pointing to this model are returned.
            fields = [
                field.name
                for field in self.Meta.model._meta._get_fields(reverse=False)  # type: ignore
                if field.name not in read_only_field
            ]

        self.Meta.read_only_fields = read_only_field
        self.Meta.fields = tuple(fields)

        extra_fields = getattr(self.Meta, "extra_fields", None)
        if extra_fields:
            self.Meta.fields += tuple(extra_fields)

        if extra_fields and not isinstance(extra_fields, (list, tuple)):
            raise TypeError("The `extra_fields` attribute must be a list or tuple.")

        extra_read_only_fields = getattr(self.Meta, "extra_read_only_fields", None)
        if extra_read_only_fields:
            self.Meta.read_only_fields += tuple(extra_read_only_fields)

        if extra_read_only_fields and not isinstance(
            extra_read_only_fields, (list, tuple)
        ):
            raise TypeError(
                "The `extra_read_only_fields` attribute must be a list or tuple."
            )
