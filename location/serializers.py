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

from location import models
from utils.serializers import GenericSerializer


class LocationCategorySerializer(GenericSerializer):
    """A serializer for the LocationCategory model

    The serializer provides default operations for creating, update and deleting
    Location Category, as well as listing and retrieving them.
    """

    class Meta:
        """
        A class representing the metadata for the `LocationCategorySerializer`
        class.
        """

        model = models.LocationCategory
        extra_fields = ("name", "description")


class LocationContactSerializer(GenericSerializer):
    """A serializer for the LocationContact model

    The serializer provides default operations for creating, update and deleting
    Location Contact, as well as listing and retrieving them.
    """

    class Meta:
        """
        A class representing the metadata for the `LocationContactSerializer`
        class.
        """

        model = models.LocationContact


class LocationCommentSerializer(GenericSerializer):
    """A serializer for the LocationComment model

    The serializer provides default operations for creating, update and deleting
    Location Comment information, as well as listing and retrieving them.
    """

    class Meta:
        """
        A class representing the metadata for the `LocationCommentSerializer`
        class.
        """

        model = models.LocationComment


class LocationSerializer(GenericSerializer):
    """A serializer for the Location model.

    The serializer provides default operations for creating, update and deleting
    Location information, as well as listing and retrieving them.
    """

    location_category = serializers.PrimaryKeyRelatedField(
        queryset=models.LocationCategory.objects.all(),
        allow_null=True,
    )
    location_contacts = serializers.PrimaryKeyRelatedField(
        queryset=models.LocationContact.objects.all(),
        many=True,
        allow_null=True,
    )
    location_comments = serializers.PrimaryKeyRelatedField(
        queryset=models.LocationComment.objects.all(),
        many=True,
        allow_null=True,
    )

    class Meta:
        """
        A class representing the metadata for the `LocationSerializer`
        class.
        """

        model = models.Location
        extra_fields = ("location_category", "location_contacts", "location_comments")
