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
from typing import Any, override

from rest_framework import serializers

from location import helpers, models
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
        extra_read_only_fields = ("location",)


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
        extra_read_only_fields = ("location",)


class LocationSerializer(GenericSerializer):
    """A serializer for the Location model.

    The serializer provides default operations for creating, update and deleting
    Location information, as well as listing and retrieving them.
    """

    wait_time_avg = serializers.DurationField(required=False)
    location_comments = LocationCommentSerializer(many=True, required=False)
    location_contacts = LocationContactSerializer(many=True, required=False)
    location_color = serializers.CharField(
        source="location_category.color", read_only=True
    )

    class Meta:
        """
        A class representing the metadata for the `LocationSerializer`
        class.
        """

        model = models.Location
        extra_fields = (
            "wait_time_avg",
            "location_comments",
            "location_contacts",
            "location_color",
        )

    def create(self, validated_data: Any) -> models.Location:
        """Create a new instance of the Location model with given validated data.

        This executes the creation of new Location, attaches the Location to the business unit 
        and organization associated with the request. It updates the Location contacts & comments
        associated with the Location.

        Args:
            validated_data (Any): Data validated through serializer for creating a Location.

        Returns:
           models.Location: Newly created Location instance.
        """

        # Get the organization of the user from the request.
        organization = super().get_organization

        # Get the business unit of the user from the request.
        business_unit = super().get_business_unit

        # Popped data (comments, contacts)
        location_comments_data = validated_data.pop("location_comments", [])
        location_contacts_data = validated_data.pop("location_contacts", [])

        # Create the Location.
        validated_data["organization"] = organization
        validated_data["business_unit"] = business_unit
        location = models.Location.objects.create(**validated_data)

        # Create the Location Comments
        helpers.create_or_update_location_comments(
            location=location,
            business_unit=business_unit,
            organization=organization,
            location_comments_data=location_comments_data,
        )

        # Create the Location Contacts
        helpers.create_or_update_location_contacts(
            location=location,
            business_unit=business_unit,
            organization=organization,
            location_contacts_data=location_contacts_data,
        )

        return location

    @override
    def update(self, instance: models.Location, validated_data: Any) -> models.Location:
        """Update an existing instance of the Location model with given validated data.

        This method updates an existing Location, based on the data provided in the request.
        It updates the Location contacts & comments associated with the Location.

        Args:
            instance (models.Location): Existing instance of Location model to update.
            validated_data (Any): Data validated through serializer for updating a Location.

        Returns:
            models.Location: Updated Location instance.
        """

        # Get the organization of the user from the request.
        organization = super().get_organization

        # Get the business unit of the user from the request.
        business_unit = super().get_business_unit

        # Popped data (comments, contacts)
        location_comments_data = validated_data.pop("location_comments", [])
        location_contacts_data = validated_data.pop("location_contacts", [])

        # Create or update the location comments.
        if location_comments_data:
            helpers.create_or_update_location_comments(
                location=instance,
                business_unit=business_unit,
                organization=organization,
                location_comments_data=location_comments_data,
            )

        # Create or update the location contacts.
        if location_contacts_data:
            helpers.create_or_update_location_contacts(
                location=instance,
                business_unit=business_unit,
                organization=organization,
                location_contacts_data=location_contacts_data,
            )

        # Update the location.
        for attr, value in validated_data.items():
            setattr(instance, attr, value)
        instance.save()

        return instance
