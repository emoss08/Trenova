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

from django.utils.translation import gettext_lazy as _
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

    location_contacts = LocationContactSerializer(many=True, required=False)
    location_comments = LocationCommentSerializer(many=True, required=False)

    class Meta:
        """
        A class representing the metadata for the `LocationSerializer`
        class.
        """

        model = models.Location
        extra_fields = ("location_contacts", "location_comments")

    def create(self, validated_data: Any) -> models.Location:
        """Create new Location

        Args:
            validated_data (Any): Validated data

        Returns:
            models.Location: Created Location
        """

        organization = super().get_organization

        comments_data = validated_data.pop("location_comments", [])
        contacts_data = validated_data.pop("location_contacts", [])

        validated_data["organization"] = organization
        location = models.Location.objects.create(**validated_data)

        # Create the Location Comment
        if comments_data:
            for comment in comments_data:
                comment["organization"] = organization
                models.LocationComment.objects.create(location=location, **comment)

        # Create the Location Contact
        if contacts_data:
            for contact in contacts_data:
                contact["organization"] = organization
                models.LocationContact.objects.create(location=location, **contact)

        return location

    def update(  # type: ignore
        self, instance: models.Location, validated_data: Any
    ) -> models.Location:
        """Update the worker

        Args:
            instance (models.Worker): Worker instance.
            validated_data (Any): Validated data.

        Returns:
            models.Location: Location instance.
        """

        comments_data = validated_data.pop("location_comments", [])
        contacts_data = validated_data.pop("location_contacts", [])

        # Update the Location

        # TODO(WOLFRED): CHANGE THIS
        instance.code = validated_data.get("code", instance.code)
        instance.location_category = validated_data.get(
            "location_category", instance.location_category
        )
        instance.depot = validated_data.get("depot", instance.depot)
        instance.description = validated_data.get("description", instance.description)
        instance.address_line_1 = validated_data.get(
            "address_line_1", instance.address_line_1
        )
        instance.address_line_2 = validated_data.get(
            "address_line_2", instance.address_line_2
        )
        instance.city = validated_data.get("city", instance.city)
        instance.state = validated_data.get("state", instance.state)
        instance.zip_code = validated_data.get("zip_code", instance.zip_code)
        instance.longitude = validated_data.get("longitude", instance.longitude)
        instance.latitude = validated_data.get("latitude", instance.latitude)
        instance.place_id = validated_data.get("place_id", instance.place_id)
        instance.is_geocoded = validated_data.get("is_geocoded", instance.is_geocoded)

        # Update the location comments
        if comments_data:
            for comment_data in comments_data:
                comment_id = comment_data.get("id", None)
                if comment_id:
                    try:
                        location_comment = models.LocationComment.objects.get(
                            id=comment_id, location=instance
                        )
                    except models.LocationComment.DoesNotExist:
                        raise serializers.ValidationError(
                            {
                                "comments": (
                                    _(
                                        f"Location comment with id '{comment_id}' does not exist. "
                                        f"Delete the ID and try again."
                                    )
                                )
                            }
                        )

                    location_comment.update_location_comment(**comment_data)
                else:
                    comment_data["organization"] = instance.organization
                    instance.location_comments.create(**comment_data)

        # Update the location contacts.
        if contacts_data:
            for contact_data in contacts_data:
                contact_id = contact_data.get("id", None)

                if contact_id:
                    try:
                        location_contact = models.LocationContact.objects.get(
                            id=contact_id, location=instance
                        )
                    except models.LocationContact.DoesNotExist:
                        raise serializers.ValidationError(
                            {
                                "comments": (
                                    _(
                                        f"Location contact with id '{contact_id}' does not exist. "
                                        f"Delete the ID and try again."
                                    )
                                )
                            }
                        )

                    location_contact.update_location_contact(**contact_data)
                else:
                    contact_data["organization"] = instance.organization
                    instance.location_contacts.create(**contact_data)

        return instance
