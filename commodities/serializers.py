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
        extra_fields = ("is_active", "hazard_class", "packing_group")


class CommoditySerializer(GenericSerializer):
    """
    A serializer for the `Commodity` model.

    This serializer converts instances of the `Commodity` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, description, and code) to create the serialized
    representation of the `Commodity` model.
    """

    hazmat = serializers.PrimaryKeyRelatedField(
        queryset=models.HazardousMaterial.objects.all(), allow_null=True
    )

    class Meta:
        """
        A class representing the metadata for the `CommoditySerializer` class.
        """

        model = models.Commodity
        extra_fields = ("hazmat",)
