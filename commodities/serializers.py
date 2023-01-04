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

from commodities import models
from utils.serializers import GenericSerializer


class HazardousMaterialSerializer(GenericSerializer):
    """
    A serializer for the `HazardousMaterial` model.

    This serializer converts instances of the `HazardousMaterial` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, description, and code) to create the serialized
    representation of the `HazardousMaterial` model.
    """

    is_active = serializers.BooleanField(default=True)
    hazard_class = serializers.ChoiceField(
        choices=models.HazardousMaterial.HazardousClassChoices.choices
    )
    packing_group = serializers.ChoiceField(
        choices=models.HazardousMaterial.PackingGroupChoices.choices
    )

    class Meta:
        """
        A class representing the metadata for the `HazardousMaterialSerializer` class.
        """

        model = models.HazardousMaterial
        fields = (
            "id",
            "organization",
            "is_active",
            "name",
            "description",
            "hazard_class",
            "packing_group",
            "erg_number",
            "proper_shipping_name",
            "created",
            "modified",
        )
        read_only_fields = (
            "organization",
            "id",
            "created",
            "modified",
        )


class CommoditySerializer(GenericSerializer):
    """
    A serializer for the `Commodity` model.

    This serializer converts instances of the `Commodity` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, description, and code) to create the serialized
    representation of the `Commodity` model.
    """

    hazmat = HazardousMaterialSerializer(required=False)
    unit_of_measure = serializers.ChoiceField(
        models.Commodity.UnitOfMeasureChoices.choices
    )

    class Meta:
        """
        A class representing the metadata for the `CommoditySerializer` class.
        """

        model = models.Commodity
        fields = (
            "id",
            "organization",
            "name",
            "description",
            "min_temp",
            "max_temp",
            "set_point_temp",
            "unit_of_measure",
            "is_hazmat",
            "hazmat",
            "created",
            "modified",
        )
        read_only_fields = (
            "organization",
            "id",
            "created",
            "modified",
        )

    def create(self, validated_data: Any) -> models.Commodity:
        """Create a new commodity.

        Args:
            validated_data (Any): The validated data.

        Returns:
            models.Commodity: The newly created commodity.
        """

        # Get the organization from the user if they are using basic auth.
        organization = super().get_organization

        hazmat_data = validated_data.pop("hazmat", {})

        if hazmat_data:
            hazmat = models.HazardousMaterial.objects.create(
                organization=organization, **hazmat_data
            )
            validated_data["hazmat"] = hazmat

        commodity = models.Commodity.objects.create(
            organization=organization, **validated_data
        )
        return commodity

    def update(  # type: ignore
        self, instance: models.Commodity, validated_data: Any
    ) -> models.Commodity:
        """Update an existing commodity.

        Args:
            instance (models.Commodity): The commodity to update.
            validated_data (Any): The validated data.

        Returns:
            models.Commodity: The updated commodity.
        """

        hazmat_data = validated_data.pop("hazmat", {})

        if hazmat_data and instance.hazmat:
            instance.hazmat.update_hazmat(**hazmat_data)

        instance.update_commodity(**validated_data)

        return instance
