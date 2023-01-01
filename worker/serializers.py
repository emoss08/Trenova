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

from accounts.models import Token
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
        fields = [
            "id",
            "comment_type",
            "comment",
            "entered_by",
            "created",
            "modified",
        ]
        read_only_fields = ["created", "modified"]


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
        fields = [
            "id",
            "name",
            "phone",
            "email",
            "relationship",
            "is_primary",
            "mobile_phone",
            "created",
            "modified",
        ]
        read_only_fields = (
            "created",
            "modified",
        )


class WorkerProfileSerializer(GenericSerializer):
    """
    Worker Profile Serializer
    """

    # hire_date = serializers.DateField(format="%Y-%m-%d", read_only=True)

    class Meta:
        """
        Metaclass for WorkerProfileSerializer
        """

        model = models.WorkerProfile
        fields = [
            "race",
            "sex",
            "date_of_birth",
            "license_number",
            "license_state",
            "license_expiration_date",
            "endorsements",
            "hazmat_expiration_date",
            "hm_126_expiration_date",
            "termination_date",
            "review_date",
            "physical_due_date",
            "mvr_due_date",
            "medical_cert_date",
            "created",
            "modified",
        ]
        read_only_fields = [
            "organization",
            "created",
            "modified",
            "hire_date",
        ]


class WorkerSerializer(GenericSerializer):
    """
    Worker Serializer
    """

    is_active = serializers.BooleanField(default=True)
    organization = serializers.CharField(source="organization.name", read_only=True)
    worker_type = serializers.ChoiceField(
        choices=models.Worker.WorkerType.choices,
        default=models.Worker.WorkerType.EMPLOYEE,
    )
    profile = WorkerProfileSerializer(required=False)
    contacts = WorkerContactSerializer(many=True, required=False)
    comments = WorkerCommentSerializer(many=True, required=False)

    class Meta:
        """
        Metaclass for WorkerSerializer
        """

        model = models.Worker
        fields = [
            "id",
            "organization",
            "code",
            "is_active",
            "worker_type",
            "first_name",
            "last_name",
            "address_line_1",
            "address_line_2",
            "city",
            "state",
            "zip_code",
            "depot",
            "manager",
            "entered_by",
            "created",
            "modified",
            "profile",
            "contacts",
            "comments",
        ]
        read_only_fields = [
            "organization",
            "id",
            "code",
            "created",
            "modified",
        ]

    def create(self, validated_data: Any) -> models.Worker:
        """Create the worker.

        Args:
            validated_data (Any): Validated data.

        Returns:
            Models.Worker: Worker Instance
        """

        # Get the organization of the user from the request.
        organization = super().get_organization

        # Get the user from the request.
        user = self.context["request"].user

        # Popped data (profile, contacts, comments)
        profile_data = validated_data.pop("profile", {})
        contacts_data = validated_data.pop("contacts", [])
        comments_data = validated_data.pop("comments", [])

        # Create the Worker.
        validated_data["organization"] = organization
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
            models.WorkerProfile.objects.create(worker=worker, **profile_data)

        # Create the Worker Contacts
        if contacts_data:
            for contact in contacts_data:
                contact["organization"] = organization
                models.WorkerContact.objects.create(worker=worker, **contact)

        # Create the Worker Comments
        if comments_data:
            for comment_data in comments_data:
                comment_data["organization"] = organization
                models.WorkerComment.objects.create(worker=worker, **comment_data)

        return worker

    def update(self, instance: models.Worker, validated_data: Any) -> models.Worker:  # type: ignore
        """Update the worker

        Args:
            instance (models.Worker): Worker instance.
            validated_data (Any): Validated data.

        Returns:
            models.Worker: Worker instance.
        """

        profile_data = validated_data.pop("profile", {})
        comments_data = validated_data.pop("comments", [])
        contacts_data = validated_data.pop("contacts", [])

        # Update the worker.
        instance.is_active = validated_data.get("is_active", instance.is_active)
        instance.worker_type = validated_data.get("worker_type", instance.worker_type)
        instance.first_name = validated_data.get("first_name", instance.first_name)
        instance.last_name = validated_data.get("last_name", instance.last_name)
        instance.address_line_1 = validated_data.get(
            "address_line_1", instance.address_line_1
        )
        instance.address_line_2 = validated_data.get(
            "address_line_2", instance.address_line_2
        )
        instance.city = validated_data.get("city", instance.city)
        instance.state = validated_data.get("state", instance.state)
        instance.zip_code = validated_data.get("zip_code", instance.zip_code)
        instance.depot = validated_data.get("depot", instance.depot)
        instance.manager = validated_data.get("manager", instance.manager)
        instance.entered_by = validated_data.get("entered_by", instance.entered_by)
        instance.save()

        # Update the worker profile.
        if profile_data:
            instance.profile.update_profile(**profile_data)

        # Update the worker comments.
        if comments_data:
            for comment_data in comments_data:
                comment_id = comment_data.get("id", None)
                if comment_id:
                    try:
                        worker_comment = models.WorkerComment.objects.get(
                            id=comment_id, worker=instance
                        )
                    except models.WorkerComment.DoesNotExist:
                        raise serializers.ValidationError(
                            {
                                "comments": (
                                    _(
                                        f"Worker comment with id '{comment_id}' does not exist. "
                                        f"Delete the ID and try again."
                                    )
                                )
                            }
                        )

                    worker_comment.update_location_comments(**comment_data)
                else:
                    comment_data["organization"] = instance.organization
                    instance.comments.create(**comment_data)

        # Update the worker contacts.
        if contacts_data:
            for contact_data in contacts_data:
                contact_id = contact_data.get("id", None)

                if contact_id:
                    try:
                        worker_contact = models.WorkerContact.objects.get(
                            id=contact_id, worker=instance
                        )
                    except models.WorkerContact.DoesNotExist:
                        raise serializers.ValidationError(
                            {
                                "comments": (
                                    _(
                                        f"Worker contact with id '{contact_id}' does not exist. "
                                        f"Delete the ID and try again."
                                    )
                                )
                            }
                        )

                    worker_contact.update_location_contact(**contact_data)
                else:
                    contact_data["organization"] = instance.organization
                    instance.contacts.create(**contact_data)

        return instance
