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

from typing import Any

from rest_framework import serializers

from utils.serializers import GenericSerializer
from worker import helpers, models


class WorkerCommentSerializer(GenericSerializer):
    """
    Worker Comment Serializer
    """

    id = serializers.UUIDField(required=False, allow_null=True)

    class Meta:
        """
        Metaclass for WorkerCommentSerializer
        """

        model = models.WorkerComment
        extra_read_only_fields = ("worker",)

    def create(self, validated_data: Any) -> models.WorkerComment:
        raise NotImplementedError(
            "WorkerCommentSerializer should not create objects directly. Use helper functions instead."
        )


class WorkerContactSerializer(GenericSerializer):
    """
    Worker Contact Serializer
    """

    id = serializers.UUIDField(required=False, allow_null=True)

    class Meta:
        """
        Metaclass for WorkerContactSerializer
        """

        model = models.WorkerContact
        extra_read_only_fields = ("worker",)

    def create(self, validated_data: Any) -> models.WorkerContact:
        raise NotImplementedError(
            "WorkerContactSerializer should not create objects directly. Use helper functions instead."
        )


class WorkerProfileSerializer(GenericSerializer):
    """
    Worker Profile Serializer
    """

    class Meta:
        """
        Metaclass for WorkerProfileSerializer
        """

        model = models.WorkerProfile
        extra_read_only_fields = ("worker",)

    def create(self, validated_data: Any) -> models.WorkerProfile:
        raise NotImplementedError(
            "WorkerProfileSerializer should not create objects directly. Use helper functions instead."
        )


class WorkerSerializer(GenericSerializer):
    """
    Worker Serializer
    """

    # id = serializers.UUIDField(required=False, allow_null=True)
    profile = WorkerProfileSerializer(required=False)
    contacts = WorkerContactSerializer(many=True, required=False)
    comments = WorkerCommentSerializer(many=True, required=False)

    class Meta:
        """
        Metaclass for WorkerSerializer
        """

        model = models.Worker
        extra_fields = (
            "profile",
            "contacts",
            "comments",
        )

    def validate_code(self, value: str) -> str:
        """Validate the `code` field of the Worker model.

        This method validates the `code` field of the Worker model.
        It checks if the worker with the given code already exists in the organization.
        If the worker exists, it raises a validation error.

        Args:
            value (str): The value of the `code` field.

        Returns:
            str: The value of the `code` field.

        Raises:
            serializers.ValidationError: If the worker with the given code already exists in the
             organization.
        """
        organization = super().get_organization

        queryset = models.Worker.objects.filter(
            organization=organization,
            code__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.Worker):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Worker with this `code` already exists. Please try again."
            )

        return value

    def to_representation(self, instance: models.Worker) -> dict[str, Any]:
        """Customize the representation of the worker, which will be returned in API response.

        This method provides a custom serialization format, adds related objects data to response.

        Args:
            instance (models.Worker): object of Worker model to represent.

        Returns:
            dict: Dictionary containing the representation of relevant information related to the Worker.
        """
        representation = super().to_representation(instance)
        representation["profile"] = WorkerProfileSerializer(instance.profile).data
        representation["contacts"] = WorkerContactSerializer(
            instance.contacts.all(), many=True
        ).data
        representation["comments"] = WorkerCommentSerializer(
            instance.comments.all(), many=True
        ).data
        return representation

    def create(self, validated_data: Any) -> models.Worker:
        """Create a new instance of the Worker model with given validated data.

        This method creates a new worker, attaches the worker to the business unit and organization associated with the request.
        It updates the profile, contacts, and comments associated with the worker Profile.

        Args:
            validated_data (Any): data validated through serializer for the creation of worker.

        Returns:
            models.Worker: Newly created Worker instance.
        """

        # Get the organization of the user from the request.
        organization = super().get_organization

        # Get the business unit of the user from the request.
        business_unit = super().get_business_unit

        # Get the user from the request.
        user = self.context["request"].user

        # Popped data (profile, contacts, comments)
        worker_profile_data = validated_data.pop("profile", None)
        worker_contacts_data = validated_data.pop("contacts", [])
        worker_comments_data = validated_data.pop("comments", [])

        # Create the Worker.
        validated_data["organization"] = organization
        validated_data["business_unit"] = business_unit
        validated_data["entered_by"] = user
        worker = models.Worker.objects.create(**validated_data)

        # Create the Worker Profile
        helpers.create_or_update_worker_profile(
            worker=worker,
            business_unit=business_unit,
            organization=organization,
            profile_data=worker_profile_data,
        )

        # Create the Worker Contacts
        helpers.create_or_update_worker_contacts(
            worker=worker,
            business_unit=business_unit,
            organization=organization,
            worker_contacts_data=worker_contacts_data,
        )

        # Create the Worker Comments
        helpers.create_or_update_worker_comments(
            worker=worker,
            business_unit=business_unit,
            organization=organization,
            worker_comment_data=worker_comments_data,
        )

        return worker

    def update(self, instance: models.Worker, validated_data: Any) -> models.Worker:  # type: ignore[override]
        """Update an existing instance of the Worker model with given validated data.

        This method updates an existing worker, based on the data provided in the request.
        It updates the profile, contacts, and comments associated with the worker Profile.

        Args:
            instance (models.Worker): existing instance of Worker model to update.
            validated_data (Any): data validated through serializer for the updation of worker profile.

        Returns:
            models.Worker: Updated Worker instance.
        """

        worker_profile_data = validated_data.pop("profile", None)
        worker_comments_data = validated_data.pop("comments", [])
        worker_contacts_data = validated_data.pop("contacts", [])

        # Update Worker Profile
        helpers.create_or_update_worker_profile(
            worker=instance,
            business_unit=instance.organization.business_unit,
            organization=instance.organization,
            profile_data=worker_profile_data,
        )

        # Update Worker Comments
        helpers.create_or_update_worker_comments(
            worker=instance,
            business_unit=instance.organization.business_unit,
            organization=instance.organization,
            worker_comment_data=worker_comments_data,
        )

        # Update Worker Contacts
        helpers.create_or_update_worker_contacts(
            worker=instance,
            business_unit=instance.organization.business_unit,
            organization=instance.organization,
            worker_contacts_data=worker_contacts_data,
        )

        for attr, value in validated_data.items():
            setattr(instance, attr, value)
        instance.save()

        return instance
