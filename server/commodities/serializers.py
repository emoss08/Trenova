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
from utils.serializers import GenericSerializer


class HazardousMaterialSerializer(GenericSerializer):
    """
    A serializer for the `HazardousMaterial` model.

    This serializer converts instances of the `HazardousMaterial` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, description, and code) to create the serialized
    representation of the `HazardousMaterial` model.
    """

    class Meta:
        """
        A class representing the metadata for the `HazardousMaterialSerializer` class.
        """

        model = models.HazardousMaterial


class CommoditySerializer(GenericSerializer):
    """
    A serializer for the `Commodity` model.

    This serializer converts instances of the `Commodity` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, description, and code) to create the serialized
    representation of the `Commodity` model.
    """

    class Meta:
        """
        A class representing the metadata for the `CommoditySerializer` class.
        """

        model = models.Commodity

    def validate_name(self, value: str) -> str:
        """Validates the name of the commodity to ensure that it is unique.

        Args:
            value: The name of the commodity to validate.

        Returns:
            str: The validated name.

        Raises:
            serializers.ValidationError: If a commodity with the same name already exists.
        """

        organization = super().get_organization

        queryset = models.Commodity.objects.filter(
            organization=organization,
            name__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.Commodity):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Commodity with this `name` already exists. Please try again."
            )

        return value
