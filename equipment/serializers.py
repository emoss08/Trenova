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

from rest_framework import serializers

from .models import Equipment, EquipmentMaintenancePlan, EquipmentManufacturer, EquipmentType


class EquipmentSerializer(serializers.ModelSerializer):
    """
    Serializer for Equipment Model.
    """

    class Meta:
        """
        Metaclass for EquipmentSerializer.
        """

        model: Type[Equipment] = Equipment
        fields: tuple[str, ...] = (
            "id",
            "equipment_type",
            "is_active",
            "description",
            "license_plate_number",
            "vin_number",
            "odometer",
            "engine_hours",
            "manufacturer",
            "manufactured_date",
            "model",
            "model_year",
            "state",
            "leased",
            "leased_date",
            "hos_exempt",
            "aux_power_unit_type",
            "fuel_draw_capacity",
            "num_of_axles",
            "transmission_manufacturer",
            "transmission_type",
            "has_berth",
            "has_electronic_engine",
            "highway_use_tax",
            "owner_operated",
            "ifta_qualified",
        )

    def to_representation(self, instance: Equipment) -> dict:
        """Serialize Equipment objects to JSON.

        Args:
            instance (Equipment): The Equipment object to serialize.

        Returns:
            dict: The serialized Equipment object.
        """
        data = super().to_representation(instance)
        return {k: v if v is not None else "" for k, v in data.items()}


class EquipmentManufacturerSerializer(serializers.ModelSerializer):
    """
    Serializer for Equipment Manufacturer Model.
    """

    class Meta:
        """
        Metaclass for EquipmentManufacturerSerializer.
        """

        model: Type[EquipmentManufacturer] = EquipmentManufacturer
        fields: tuple[str, ...] = (
            "id",
            "description",
        )

    def create(self, validated_data: dict) -> EquipmentManufacturer:
        """Create a new EquipmentManufacturer.

        Args:
            validated_data (dict): The validated data.

        Returns:
            EquipmentManufacturer: The new EquipmentManufacturer.
        """
        validated_data["organization"] = self.context[
            "request"
        ].user.profile.organization
        return super().create(validated_data)


class EquipmentTypeSerializer(serializers.ModelSerializer):
    """
    Serializer for Equipment Type Model.
    """

    class Meta:
        """
        Metaclass for EquipmentTypeSerializer.
        """

        model: Type[EquipmentType] = EquipmentType
        fields: tuple[str, ...] = (
            "id",
            "name",
            "description",
        )

    def create(self, validated_data: dict) -> EquipmentType:
        """Create a new EquipmentType.

        Args:
            validated_data (dict): The validated data.

        Returns:
            EquipmentType: The new EquipmentType.
        """
        validated_data["organization"] = self.context[
            "request"
        ].user.profile.organization
        return super().create(validated_data)


class EquipmentMaintenancePlanSerializer(serializers.ModelSerializer):
    """
    Serializer for Equipment Maintenance Plan Model.
    """

    class Meta:
        """
        Metaclass for EquipmentMaintenancePlanSerializer.
        """

        model: Type[EquipmentMaintenancePlan] = EquipmentMaintenancePlan
        fields: tuple[str, ...] = (
            "id",
            "equipment_types",
            "description",
            "by_distance",
            "by_time",
            "by_engine_hours",
            "miles",
            "months",
            "engine_hours",
        )

    def create(self, validated_data: dict) -> EquipmentMaintenancePlan:
        """Create a new EquipmentMaintenancePlan.

        Args:
            validated_data (dict): The validated data.

        Returns:
            EquipmentMaintenancePlan: The new EquipmentMaintenancePlan.
        """
        validated_data["organization"] = self.context[
            "request"
        ].user.profile.organization
        return super().create(validated_data)

    def to_representation(self, instance: EquipmentMaintenancePlan) -> dict:
        """Serialize EquipmentMaintenancePlan objects to JSON.

        Args:
            instance (EquipmentMaintenancePlan): The EquipmentMaintenancePlan object to serialize.

        Returns:
            dict: The serialized EquipmentMaintenancePlan object.
        """
        data = super().to_representation(instance)
        data["equipment_types"] = [et.name for et in instance.equipment_types.all()]
        return data

    def validate(self, attrs) -> dict:
        """Validate the data.

        Args:
            attrs (dict): The data to validate.

        Returns:
            dict: The validated data.
        """

        if attrs["by_distance"] and not attrs["miles"]:
            raise serializers.ValidationError(
                "Miles is required when by distance is checked."
            )
        if attrs["by_time"] and not attrs["months"]:
            raise serializers.ValidationError(
                "Months is required when by time is checked."
            )
        if attrs["by_engine_hours"] and not attrs["engine_hours"]:
            raise serializers.ValidationError(
                "Engine hours is required when by engine hours is checked."
            )
        return attrs
