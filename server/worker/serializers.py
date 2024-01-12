# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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
        fields = "__all__"
        read_only_fields = ("worker", "organization", "business_unit")
        extra_kwargs = {
            "organization": {"required": False},
            "business_unit": {"required": False},
            "worker": {"required": False},
        }

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
        fields = "__all__"
        read_only_fields = ("worker", "organization", "business_unit")
        extra_kwargs = {
            "worker": {"required": False},
            "organization": {"required": False},
            "business_unit": {"required": False},
        }

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
        fields = "__all__"
        read_only_fields = ("worker", "organization", "business_unit")
        extra_kwargs = {
            "organization": {"required": False},
            "business_unit": {"required": False},
            "worker": {"required": False},
        }

    def create(self, validated_data: Any) -> models.WorkerProfile:
        raise NotImplementedError(
            "WorkerProfileSerializer should not create objects directly. Use helper functions instead."
        )


class WorkerHOSSerializer(GenericSerializer):
    class Meta:
        model = models.WorkerHOS
        extra_read_only_fields = ("worker",)
        fields = "__all__"

    def create(self, validated_data: Any) -> models.WorkerHOS:
        raise NotImplementedError(
            "WorkerHOSSerializer should not create objects directly. Use helper functions instead."
        )


class WorkerSerializer(GenericSerializer):
    """
    Worker Serializer
    """

    profile = WorkerProfileSerializer(required=False)
    contacts = WorkerContactSerializer(many=True, required=False)
    comments = WorkerCommentSerializer(many=True, required=False)
    current_hos = serializers.SerializerMethodField()

    class Meta:
        """
        Metaclass for WorkerSerializer
        """

        model = models.Worker
        fields = "__all__"
        read_only_fields = ("organization", "business_unit")
        extra_kwargs = {
            "organization": {"required": False},
            "business_unit": {"required": False},
            "code": {"required": False, "allow_null": True},
        }

    def get_current_hos(self, obj: models.Worker) -> Any:
        # Use the prefetched latest_hos
        if hasattr(obj, "latest_hos") and obj.latest_hos:
            return WorkerHOSSerializer(obj.latest_hos[0]).data
        return None

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

    def update(self, instance: models.Worker, validated_data: Any) -> models.Worker:
        """Update an existing instance of the Worker model with given validated data.

        This method updates an existing worker, based on the data provided in the request.
        It updates the profile, contacts, and comments associated with the worker Profile.

        Args:
            instance (models.Worker): existing instance of Worker model to update.
            validated_data (Any): data validated through serializer for the updation of worker profile.

        Returns:
            models.Worker: Updated Worker instance.
        """

        # Get the organization of the user from the request.
        organization = super().get_organization

        # Get the business unit of the user from the request.
        business_unit = super().get_business_unit

        # Popped data (profile, contacts, comments)
        worker_profile_data = validated_data.pop("profile", None)
        worker_comments_data = validated_data.pop("comments", [])
        worker_contacts_data = validated_data.pop("contacts", [])

        # Update Worker Profile
        helpers.create_or_update_worker_profile(
            worker=instance,
            business_unit=business_unit,
            organization=organization,
            profile_data=worker_profile_data,
        )

        # Update Worker Comments
        helpers.create_or_update_worker_comments(
            worker=instance,
            business_unit=business_unit,
            organization=organization,
            worker_comment_data=worker_comments_data,
        )

        # Update Worker Contacts
        helpers.create_or_update_worker_contacts(
            worker=instance,
            business_unit=business_unit,
            organization=organization,
            worker_contacts_data=worker_contacts_data,
        )

        for attr, value in validated_data.items():
            setattr(instance, attr, value)
        instance.save()

        return instance
