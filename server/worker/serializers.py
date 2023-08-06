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

from django.db import transaction
from rest_framework import serializers

from utils.serializers import GenericSerializer
from worker import models


class WorkerCommentSerializer(GenericSerializer):
    """
    Worker Comment Serializer
    """

    id = serializers.UUIDField(required=False)

    class Meta:
        """
        Metaclass for WorkerCommentSerializer
        """

        model = models.WorkerComment
        extra_fields = ("id",)
        extra_read_only_fields = (
            "worker",
            "business_unit",
        )


class WorkerContactSerializer(GenericSerializer):
    """
    Worker Contact Serializer
    """

    id = serializers.UUIDField(required=False)

    class Meta:
        """
        Metaclass for WorkerContactSerializer
        """

        model = models.WorkerContact
        extra_read_only_fields = ("worker",)
        extra_fields = (
            "organization",
            "business_unit",
            "id",
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
        extra_fields = (
            "organization",
            "business_unit",
        )


class WorkerSerializer(GenericSerializer):
    """
    Worker Serializer
    """

    id = serializers.UUIDField(
        required=False,
        allow_null=True,
    )
    code = serializers.CharField(
        required=False,
        allow_null=True,
    )
    profile = WorkerProfileSerializer(required=False)
    contacts = WorkerContactSerializer(many=True, required=False)
    comments = WorkerCommentSerializer(many=True, required=False)

    class Meta:
        """
        Metaclass for WorkerSerializer
        """

        model = models.Worker
        extra_fields = (
            "organization",
            "business_unit",
            "is_active",
            "depot",
            "manager",
            "entered_by",
            "profile",
            "contacts",
            "comments",
            "code",
        )

    def create(self, validated_data: Any) -> models.Worker:
        """Create the worker.

        Args:
            validated_data (Any): Validated data.

        Returns:
            Models.Worker: Worker Instance
        """
        # Get the organization of the user from the request.
        organization = super().get_organization

        # Get the business unit of the user from the request.
        business_unit = super().get_business_unit

        # Get the user from the request.
        user = self.context["request"].user

        # Popped data (profile, contacts, comments)
        profile_data = validated_data.pop("profile", {})
        contacts_data = validated_data.pop("contacts", [])
        comments_data = validated_data.pop("comments", [])

        # Create the Worker.
        validated_data["organization"] = organization
        validated_data["business_unit"] = business_unit
        validated_data["entered_by"] = user
        worker = models.Worker.objects.create(**validated_data)

        # Create the Worker Profile
        if profile_data:
            # Due to the worker profile signal being a thing, we need to
            # delete the worker profile if it exists. Then we can create
            # a new one from the requests.

            worker_profile = models.WorkerProfile.objects.get(worker=worker)
            worker_profile.delete()

            profile_data["organization"] = organization
            profile_data["business_unit"] = business_unit
            models.WorkerProfile.objects.create(worker=worker, **profile_data)

        # Create the Worker Contacts
        if contacts_data:
            for contact in contacts_data:
                contact["organization"] = organization
                contact["business_unit"] = business_unit
                models.WorkerContact.objects.create(worker=worker, **contact)

        # Create the Worker Comments
        if comments_data:
            for comment_data in comments_data:
                comment_data["organization"] = organization
                comment_data["business_unit"] = business_unit
                models.WorkerComment.objects.create(worker=worker, **comment_data)

        return worker

    def update(self, instance: models.Worker, validated_data: Any) -> models.Worker:  # type: ignore
        """The update function is a bit more complicated. It first extracts the profile, comments and
        contacts data from the validated_data dictionary. Then it enters an atomic transaction block
        to ensure that all of these operations are performed in one database transaction.

        Args:
            self: Refer to the instance of the class
            instance(models.Worker): Pass in the instance of the model that is being updated
            validated_data(Any): Pass the validated data to the update function

        Returns:
            The instance of the worker that was updated
        """
        profile_data = validated_data.pop("profile", {})
        comments_data = validated_data.pop("comments", [])
        contacts_data = validated_data.pop("contacts", [])

        with transaction.atomic():
            if profile_data:
                instance.profile.update_worker_profile(**profile_data)

            for comment_data in comments_data:
                comment_id = comment_data.pop("id", None)
                defaults = {
                    **comment_data,
                    "organization": instance.organization,
                    "business_unit": instance.organization.business_unit,
                }
                if comment_id:
                    updated = models.WorkerComment.objects.filter(
                        id=comment_id, worker=instance
                    ).update(**defaults)
                    if not updated:
                        raise serializers.ValidationError(
                            {
                                "comments": f"Worker comment with id '{comment_id}' does not exist."
                            }
                        )
                else:
                    instance.comments.create(**defaults)

            for contact_data in contacts_data:
                contact_id = contact_data.pop("id", None)
                defaults = {
                    **contact_data,
                    "organization": instance.organization,
                    "business_unit": instance.organization.business_unit,
                }
                if contact_id:
                    updated = models.WorkerContact.objects.filter(
                        id=contact_id, worker=instance
                    ).update(**defaults)
                    if not updated:
                        raise serializers.ValidationError(
                            {
                                "contacts": f"Worker contact with id '{contact_id}' does not exist."
                            }
                        )
                else:
                    instance.contacts.create(**defaults)

            for attr, value in validated_data.items():
                setattr(instance, attr, value)
            instance.save()

        return instance