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

from route import models
from utils.serializers import GenericSerializer


class RouteSerializer(GenericSerializer):
    """A serializer class for the Route model

    The `RouteSerializer` class provides default operations
    for creating, update and deleting Routes, as well as
    listing and retrieving them.
    """

    id = serializers.UUIDField(required=False)

    class Meta:
        """
        A class representing the metadata for the `RouteSerializer`
        class.
        """

        model = models.Route
        fields = (
            "id",
            "organization",
            "origin",
            "destination",
            "total_mileage",
            "duration",
            "created",
            "modified",
        )
        read_only_fields = (
            "organization",
            "id",
            "created",
            "modified",
        )


class RouteControlSerializer(GenericSerializer):
    """A serializer for the Route Control model

    The `RouteControlSerializer` class provides default operations
    for creating, update, and deleting Route Control, as well as
    listing and retrieving data.
    """

    mileage_unit = serializers.ChoiceField(
        choices=models.RouteControl.RouteDistanceUnitChoices.choices,
        default=models.RouteControl.RouteDistanceUnitChoices.IMPERIAL,
    )
    traffic_model = serializers.ChoiceField(
        choices=models.RouteControl.RouteModelChoices.choices,
        default=models.RouteControl.RouteModelChoices.BEST_GUESS,
    )

    class Meta:
        """
        A class representing for the metadata for the
        `RouteControlSerializer` class.
        """

        model = models.RouteControl
        fields = (
            "id",
            "organization",
            "mileage_unit",
            "traffic_model",
            "generate_routes",
            "avoid_tolls",
            "avoid_highways",
            "avoid_ferries",
            "created",
            "modified",
        )
        read_only_fields = (
            "organization",
            "id",
            "created",
            "modified",
        )
