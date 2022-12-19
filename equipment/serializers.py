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
from typing import Any

from rest_framework import serializers

from equipment import models
from utils.serializers import GenericSerializer


class EquipmentTypeDetailSerializer(GenericSerializer):
    """A serializer for the EquipmentTypeDetail model

    The serializer provides default operations for creating, update and deleting
    Equipment Type Detail, as well as listing and retrieving them.
    """

    equipment_class = serializers.ChoiceField(
        choices=models.EquipmentTypeDetail.EquipmentClassChoices.choices
    )

    class Meta:
        """
        A class representing the metadata for the `EquipmentTypeDetailSerializer`
        class.
        """

        model = models.EquipmentTypeDetail
        fields = (
            "equipment_class",
            "fixed_cost",
            "variable_cost",
            "height",
            "length",
            "width",
            "weight",
            "idling_fuel_usage",
            "exempt_from_tolls",
            "created",
            "modified",
        )
        read_only_fields = (
            "organization",
            "id",
            "created",
            "modified",
        )


class EquipmentTypeSerializer(GenericSerializer):
    """A serializer for the EquipmentType model.

    The serializer provides default operations for creating, updating, and deleting
    Equipment Types, as well as listing and retrieving them.
    """

    equipment_type_details = EquipmentTypeDetailSerializer(required=False)

    class Meta:
        """
        A class representing the metadata for the `EquipmentTypeSerializer` class.
        """

        model = models.EquipmentType
        fields = (
            "organization",
            "id",
            "description",
            "created",
            "modified",
            "equipment_type_details",
        )
        read_only_fields = (
            "organization",
            "created",
            "modified",
        )

    def create(self, validated_data: Any) -> models.EquipmentType:
        """Create new Equipment Type

        Args:
            validated_data (Any): Validated data

        Returns:
            models.EquipmentType: Created EquipmentType
        """
        detail_data = validated_data.pop("equipment_type_details", {})
        organization = super().get_organization

        equipment_type = models.EquipmentType.objects.create(
            organization=organization, **validated_data
        )

        if detail_data:
            _details = models.EquipmentTypeDetail.objects.get(
                organization=organization, equipment_type=equipment_type
            )
            if _details:
                _details.delete()

            models.EquipmentTypeDetail.objects.create(
                organization=organization, equipment_type=equipment_type, **detail_data
            )

        return equipment_type

    def update(self, instance: models.EquipmentType, validated_data: Any) -> models.EquipmentType:
        """Update Equipment Type

        Args:
            instance (models.EquipmentType): EquipmentType instance
            validated_data (Any): Validated data

        Returns:
            models.EquipmentType: Updated EquipmentType
        """

        detail_data = validated_data.pop("equipment_type_details", {})

        instance.id = validated_data.get("id", instance.id)
        instance.description = validated_data.get("description", instance.description)
        instance.save()

        if detail_data:
            instance.equipment_type_details.update_details(**detail_data)

        return instance

class EquipmentManufacturerSerializer(GenericSerializer):
    """A serializer for the EquipmentManufacturer Model

    The serializer provides default operations for creating, update and deleting
    Equipment Manufacturer, as well as listing and retrieving them.
    """

    class Meta:
        """
        A class representing the metadata for the `EquipmentManufacturerSerializer`
        class.
        """

        model = models.EquipmentManufacturer
        fields = "__all_"
