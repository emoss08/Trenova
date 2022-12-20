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

from dispatch import models
from utils.serializers import GenericSerializer


class CommentTypeSerializer(GenericSerializer):
    """A serializer for the CommentType model.

    The serializer provides default operations for creating, updating, and deleting
    comment types, as well as listing and retrieving comment types.It uses the
    `CommentType` model to convert the comment type instances to and from
    JSON-formatted data.

    Only authenticated users are allowed to access the view provided by this serializer.
    Filtering is also available, with the ability to filter by ID, and name.
    """

    class Meta:
        """
        A class representing the metadata for the `CommentTypeSerializer` class.
        """

        model = models.CommentType
        fields = (
            "id",
            "organization",
            "name",
            "description",
            "created",
            "modified",
        )
        read_only_fields = (
            "id",
            "organization",
            "created",
            "modified",
        )

class DelayCodeSerializer(GenericSerializer):
    """A serializer for the DelayCode model.

    The serializer provides default operations for creating, updating, and deleting
    delay codes, as well as listing and retrieving delay codes.It uses the
    `DelayCode` model to convert the comment type instances to and from
    JSON-formatted data.

    Only authenticated users are allowed to access the view provided by this serializer.
    Filtering is also available, with the ability to filter by ID, and name.
    """

    class Meta:
        """
        A class representing the metadata for the `DelayCodeSerializer` class.
        """

        model = models.DelayCode
        fields = (
            "id",
            "organization",
            "code",
            "description",
            "f_carrier_or_driver",
            "created",
            "modified",
        )
        read_only_fields = (
            "id",
            "organization",
            "created",
            "modified",
        )

class FleetCodeSerializer(GenericSerializer):
    """A serializer for the FleetCode model.

    The serializer provides default operations for creating, updating, and deleting
    Fleet Codes, as well as listing and retrieving fleet codes.It uses the
    `FleetCode` model to convert the comment type instances to and from
    JSON-formatted data.

    Only authenticated users are allowed to access the view provided by this serializer.
    Filtering is also available, with the ability to filter by ID, and name.
    """

    is_active = serializers.BooleanField(default=True)

    class Meta:
        """
        A class representing the metadata for the `FleetCodeSerializer` class.
        """

        model = models.FleetCode
        fields = (
            "id",
            "code",
            "organization",
            "revenue_goal",
            "deadhead_goal",
            "mileage_goal",
            "description",
            "created",
            "modified",
        )
        read_only_fields = (
            "id",
            "organization",
            "created",
            "modified",
        )

class DispatchControlSerializer(GenericSerializer):
    """A serializer for the DispatchControl model.

    The serializer provides default operations for creating, updating, and deleting
    Dispatch Control, as well as listing and retrieving Dispatch Control. It uses the
    `DispatchControl` model to convert the dispatch control instances to and from
    JSON-formatted data.

    Only authenticated users are allowed to access the view provided by this serializer.
    Filtering is also available, with the ability to filter by ID, and name.
    """

    record_service_incident = serializers.ChoiceField(
        choices=models.DispatchControl.ServiceIncidentControlChoices.choices,
        default=models.DispatchControl.ServiceIncidentControlChoices.NEVER
    )
    distance_method = serializers.ChoiceField(
        choices=models.DispatchControl.DistanceMethodChoices.choices,
        default=models.DispatchControl.DistanceMethodChoices.MONTA
    )

    class Meta:
        """
        A class representing the metadata for the `DispatchControlSerializer` class.
        """

        model = models.DispatchControl
        fields = (
            "id",
            "organization",
            "record_service_incident",
            "grace_period",
            "deadhead_target",
            "driver_assign",
            "trailer_continuity",
            "distance_method",
            "dupe_trailer_check",
            "regulatory_check",
            "prev_orders_on_hold",
            "generate_routes",
            "created",
            "modified",
        )
        read_only_fields = (
            "id",
            "organization",
            "created",
            "modified",
        )
