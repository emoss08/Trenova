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

from rest_framework import serializers

from commodities import models
from organization.models import Organization
from utils.serializers import GenericSerializer


class HazardousMaterialSerializer(GenericSerializer):
    """
    A serializer for the `HazardousMaterial` model.

    This serializer converts instances of the `HazardousMaterial` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, description, and code) to create the serialized
    representation of the `HazardousMaterial` model.
    """
    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all(),
    )
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
        extra_fields = ("organization", "is_active", "hazard_class", "packing_group")


class CommoditySerializer(GenericSerializer):
    """
    A serializer for the `Commodity` model.

    This serializer converts instances of the `Commodity` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, description, and code) to create the serialized
    representation of the `Commodity` model.
    """
    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all(),
    )
    unit_of_measure = serializers.ChoiceField(
        choices=models.Commodity.UnitOfMeasureChoices.choices
    )
    hazmat = serializers.PrimaryKeyRelatedField(
        queryset=models.HazardousMaterial.objects.all(),
        allow_null=True,
        required=False,
    )

    class Meta:
        """
        A class representing the metadata for the `CommoditySerializer` class.
        """

        model = models.Commodity
        extra_fields = ("organization", "unit_of_measure", "hazmat",)
