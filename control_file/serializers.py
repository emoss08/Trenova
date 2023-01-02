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

from control_file import models
from utils.serializers import GenericSerializer


class GoogleAPISerializer(GenericSerializer):
    """
    A serializer for the `GoogleAPI` model.

    This serializer converts instances of the `GoogleAPI` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, description, and code) to create the serialized
    representation of the `GoogleAPI` model.
    """

    mileage_unit = serializers.ChoiceField(
        choices=models.GoogleAPI.GoogleRouteDistanceUnitChoices.choices,
        default=models.GoogleAPI.GoogleRouteDistanceUnitChoices.IMPERIAL,
    )
    traffic_model = serializers.ChoiceField(
        choices=models.GoogleAPI.GoogleRouteModelChoices.choices,
        default=models.GoogleAPI.GoogleRouteModelChoices.BEST_GUESS,
    )

    class Meta:
        """
        A class representing the metadata for the `GoogleAPISerializer` class.
        """

        model = models.GoogleAPI
        fields = (
            "id",
            "organization",
            "api_key",
            "mileage_unit",
            "traffic_model",
            "add_customer_location",
            "add_location",
            "created",
            "modified",
        )
        read_only_fields = (
            "organization",
            "id",
            "created",
            "modified",
        )
