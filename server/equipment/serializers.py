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

from typing import Any

from rest_framework import serializers

from equipment import helpers, models
from utils.serializers import GenericSerializer


class EquipmentTypeDetailSerializer(GenericSerializer):
    """A serializer for the EquipmentTypeDetail model

    The serializer provides default operations for creating, update and deleting
    Equipment Type Detail, as well as listing and retrieving them.
    """

    class Meta:
        """
        A class representing the metadata for the `EquipmentTypeDetailSerializer`
        class.
        """

        model = models.EquipmentTypeDetail
        extra_read_only_fields = ("equipment_type",)


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
        extra_fields = ("equipment_type_details",)

    def validate_name(self, value: str) -> str:
        """Validate the `name` field of the EquipmentType model.

        This method validates the `name` field of the EquipmentType model.
        It checks if the equipment type with the given name already exists in the organization.
        If the equipment type exists, it raises a validation error.

        Args:
            value (str): The value of the `name` field.

        Returns:
            str: The value of the `name` field.

        Raises:
            serializers.ValidationError: If the equipment type with the given name already exists in the
             organization.
        """
        organization = super().get_organization

        queryset = models.EquipmentType.objects.filter(
            organization=organization,
            name__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.EquipmentType):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Equipment Type with this `name` already exists. Please try again."
            )

        return value

    def create(self, validated_data: Any) -> models.EquipmentType:
        """Create new Equipment Type

        Args:
            validated_data (Any): Validated data

        Returns:
            models.EquipmentType: Created EquipmentType
        """
        detail_data = validated_data.pop("equipment_type_details", {})
        organization = super().get_organization
        business_unit = super().get_business_unit

        equipment_type = models.EquipmentType.objects.create(
            organization=organization, business_unit=business_unit, **validated_data
        )

        # Create Equipment Type Details
        helpers.create_or_update_equip_type_details(
            business_unit=business_unit,
            equipment_type=equipment_type,
            detail_data=detail_data,
            organization=organization,
        )

        return equipment_type

    def update(  # type: ignore
        self, instance: models.EquipmentType, validated_data: Any
    ) -> models.EquipmentType:
        """Update Equipment Type

        Args:
            instance (models.EquipmentType): EquipmentType instance
            validated_data (Any): Validated data

        Returns:
            models.EquipmentType: Updated EquipmentType
        """

        detail_data = validated_data.pop("equipment_type_details", {})

        # Create Equipment Type Details
        helpers.create_or_update_equip_type_details(
            business_unit=instance.business_unit,
            equipment_type=instance,
            detail_data=detail_data,
            organization=instance.organization,
        )

        for attr, value in validated_data.items():
            setattr(instance, attr, value)
        instance.save()

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

    def validate_name(self, value: str) -> str:
        """Validate the `name` field of the Equipment Manufacturer model.

        This method validates the `name` field of the Equipment Manufacturer model.
        It checks if the equipment manufacturer with the given name already exists in the organization.
        If the equipment manufacturer exists, it raises a validation error.

        Args:
            value (str): The value of the `name` field.

        Returns:
            str: The value of the `name` field.

        Raises:
            serializers.ValidationError: If the equipment manufacturer with the given name already exists in the
             organization.
        """
        organization = super().get_organization

        queryset = models.EquipmentManufacturer.objects.filter(
            organization=organization,
            name__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.EquipmentManufacturer):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Equipment Manufacturer with this `name` already exists. Please try again."
            )

        return value


class TractorSerializer(GenericSerializer):
    """A serializer for the Tractor model

    The serializer provides default operations for creating, update and deleting
    Tractors, as well as listing and retrieving them.
    """

    class Meta:
        """
        A class representing the metadata for the `TractorSerializer` class.
        """

        model = models.Tractor

    def validate_code(self, value: str) -> str:
        """Validate the `code` field of the Tractor model.

        This method validates the `code` field of the Tractor model.
        It checks if the tractor with the given code already exists in the organization.
        If the tractor exists, it raises a validation error.

        Args:
            value (str): The value of the `code` field.

        Returns:
            str: The value of the `code` field.

        Raises:
            serializers.ValidationError: If the tractor with the given code already exists in the
             organization.
        """
        organization = super().get_organization

        queryset = models.Tractor.objects.filter(
            organization=organization,
            code__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.Tractor):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Tractor with this `code` already exists. Please try again."
            )

        return value


class TrailerSerializer(GenericSerializer):
    """A serializer for the Trailer model

    The serializer provides default operations for creating, update and deleting
    Trailers, as well as listing and retrieving them.
    """

    class Meta:
        """
        A class representing the metadata for the `TrailerSerializer` class.
        """

        model = models.Trailer

    def validate_code(self, value: str) -> str:
        """Validate the `code` field of the Trailer model.

        This method validates the `code` field of the Trailer model.
        It checks if the trailer with the given code already exists in the organization.
        If the trailer exists, it raises a validation error.

        Args:
            value (str): The value of the `code` field.

        Returns:
            str: The value of the `code` field.

        Raises:
            serializers.ValidationError: If the trailer with the given code already exists in the
             organization.
        """
        organization = super().get_organization

        queryset = models.Trailer.objects.filter(
            organization=organization,
            code__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.Trailer):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Trailer with this `code` already exists. Please try again."
            )

        return value


class EquipmentMaintenancePlanSerializer(GenericSerializer):
    """A serializer for the EquipmentMaintenancePlan model

    The serializer provides default operations for creating, update and deleting
    Equipment Maintenance Plan, as well as listing and retrieving them.
    """

    class Meta:
        """
        A class representing the metadata for the `EquipmentMaintenancePlanSerializer`
        class.
        """

        model = models.EquipmentMaintenancePlan

    def validate_name(self, value: str) -> str:
        """Validate the `name` field of the EquipmentMaintenancePlan model.

        This method validates the `name` field of the EquipmentMaintenancePlan model.
        It checks if the equipment maintenance plan with the given name already exists in the organization.
        If the equipment maintenance plan exists, it raises a validation error.

        Args:
            value (str): The value of the `name` field.

        Returns:
            str: The value of the `name` field.

        Raises:
            serializers.ValidationError: If the equipment maintenance plan with the given code already exists in the
             organization.
        """
        organization = super().get_organization

        queryset = models.EquipmentMaintenancePlan.objects.filter(
            organization=organization,
            name__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.EquipmentMaintenancePlan):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Equipment Maintenance Plan with this `name` already exists. Please try again."
            )

        return value
