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

from location import models
from organization.models import Organization
from utils.serializers import GenericSerializer


class LocationCategorySerializer(GenericSerializer):
    """A serializer for the LocationCategory model

    The serializer provides default operations for creating, update and deleting
    Location Category, as well as listing and retrieving them.
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )

    class Meta:
        """
        A class representing the metadata for the `LocationCategorySerializer`
        class.
        """

        model = models.LocationCategory
        extra_fields = (
            "name",
            "description",
            "organization",
        )


class LocationContactSerializer(GenericSerializer):
    """A serializer for the LocationContact model

    The serializer provides default operations for creating, update and deleting
    Location Contact, as well as listing and retrieving them.
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )

    class Meta:
        """
        A class representing the metadata for the `LocationContactSerializer`
        class.
        """

        model = models.LocationContact
        extra_fields = ("organization",)


class LocationCommentSerializer(GenericSerializer):
    """A serializer for the LocationComment model

    The serializer provides default operations for creating, update and deleting
    Location Comment information, as well as listing and retrieving them.
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )

    class Meta:
        """
        A class representing the metadata for the `LocationCommentSerializer`
        class.
        """

        model = models.LocationComment
        extra_fields = ("organization",)


class LocationSerializer(GenericSerializer):
    """A serializer for the Location model.

    The serializer provides default operations for creating, update and deleting
    Location information, as well as listing and retrieving them.
    """

    wait_time_avg = serializers.DurationField(read_only=True)
    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )
    location_category = serializers.PrimaryKeyRelatedField(
        queryset=models.LocationCategory.objects.all(),
        allow_null=True,
        required=False,
    )
    location_contacts = serializers.PrimaryKeyRelatedField(
        queryset=models.LocationContact.objects.all(),
        many=True,
        allow_null=True,
        required=False,
    )
    location_comments = serializers.PrimaryKeyRelatedField(
        queryset=models.LocationComment.objects.all(),
        many=True,
        allow_null=True,
        required=False,
    )

    class Meta:
        """
        A class representing the metadata for the `LocationSerializer`
        class.
        """

        model = models.Location
        extra_fields = (
            "wait_time_avg",
            "organization",
            "location_category",
            "location_contacts",
            "location_comments",
        )
