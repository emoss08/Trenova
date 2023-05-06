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

from typing import Any, List, Tuple, Type

from django.db.models import Model
from django.utils.functional import cached_property
from rest_framework import serializers

from accounts.models import Token
from organization.models import Organization


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

        model: type[Model]
        exclude: tuple[str, ...] = ()
        extra_fields: list[str] = []
        extra_read_only_fields: list[str] = []

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
            _organization: Organization = self.context["request"].user.organization
            return _organization
        token = self.context["request"].META.get("HTTP_AUTHORIZATION", "").split(" ")[1]
        return Token.objects.get(key=token).user.organization

    def create(self, validated_data: Any) -> Any:
        """Create the object

        Args:
            validated_data (Any): Validated data

        Returns:
            _MT: Created object
        """

        organization: Organization = self.get_organization
        validated_data["organization"] = organization

        return super().create(validated_data)

    def update(self, instance: Model, validated_data: Any) -> Any:
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
        """
        Set the fields for the serializer.
        """
        read_only_fields: tuple[str, ...] = ("organization", "created", "modified")

        original_fields = getattr(self.Meta, "fields", None)
        excluded_fields = set(getattr(self.Meta, "exclude", ()))

        # If the `fields` attribute is set, then use it.
        if original_fields is not None:
            fields = set(original_fields)
        else:
            # If reverse=True, then relations pointing to this model are returned.
            fields = {
                field.name for field in self.Meta.model._meta._get_fields(reverse=False)  # type: ignore
            }
            fields -= excluded_fields | set(read_only_fields)

        self.Meta.read_only_fields = read_only_fields
        self.Meta.fields = tuple(fields)

        # Add extra fields from the `extra_fields` attribute.
        if extra_fields := set(getattr(self.Meta, "extra_fields", [])):
            if not isinstance(extra_fields, (list, set)):
                raise TypeError("The `extra_fields` attribute must be a list or set.")
            self.Meta.fields += tuple(extra_fields)

        # Add extra read-only fields from the `extra_read_only_fields` attribute.
        if extra_read_only_fields := set(
            getattr(self.Meta, "extra_read_only_fields", [])
        ):
            if not isinstance(extra_read_only_fields, (list, set)):
                raise TypeError(
                    "The `extra_read_only_fields` attribute must be a list or set."
                )
            self.Meta.read_only_fields += tuple(extra_read_only_fields)
