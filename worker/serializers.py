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

from drf_writable_nested import WritableNestedModelSerializer
from rest_framework import serializers

from utils.serializers import GenericSerializer
from worker import models


class WorkerCommentSerializer(GenericSerializer):
    """
    Worker Comment Serializer
    """

    class Meta:
        """
        Metaclass for WorkerCommentSerializer
        """

        model = models.WorkerComment
        fields = [
            "id",
            "worker",
            "comment_type",
            "comment",
            "entered_by",
            "created",
            "modified",
        ]
        read_only_fields = ["id", "created", "modified"]


class WorkerContactSerializer(GenericSerializer):
    """
    Worker Contact Serializer
    """

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
        read_only_fields = [
            "organization",
            "id",
            "created",
            "modified",
        ]


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

    def create(self, validated_data: dict) -> models.WorkerProfile:
        """
        Create Worker Profile

        Create a worker profile.

        Args:
            validated_data (dict): The validated data.

        Returns:
            models.WorkerProfile: The worker profile.
        """
        return models.WorkerProfile.objects.create(**validated_data)


class WorkerSerializer(WritableNestedModelSerializer):
    """
    Worker Serializer
    """

    organization = serializers.CharField(source="organization.name", read_only=True)
    profile = WorkerProfileSerializer(required=False, allow_null=True)
    contacts = WorkerContactSerializer(many=True, required=False)
    comments = WorkerCommentSerializer(many=True, required=False)

    class Meta:
        """
        Metaclass for WorkerSerializer
        """

        model = models.Worker
        fields = [
            "organization",
            "id",
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
        """
        Create the worker
        """

        profile_data = validated_data.pop("profile", [])
        contacts_data = validated_data.pop("contacts", [])
        comments_data = validated_data.pop("comments", [])

        validated_data["organization"] = self.context["request"].user.organization
        profile_data["organization"] = self.context["request"].user.organization

        worker = models.Worker.objects.create(**validated_data)

        if profile_data:
            models.WorkerProfile.objects.create(worker=worker, **profile_data)

        if contacts_data:
            for contact_data in contacts_data:
                models.WorkerContact.objects.create(worker=worker, **contact_data)

        if comments_data:
            for comment_data in comments_data:
                models.WorkerComment.objects.create(worker=worker, **comment_data)

        return worker

    def update(self, instance, validated_data: Any) -> models.Worker:
        """
        Update the worker
        """
        super().update(instance, validated_data)

        profile_data = validated_data.pop("profile", [])
        profile = instance.profile

        if profile_data:
            for key, value in profile_data.items():
                setattr(profile, key, value)
            profile.save()

        contacts_data = validated_data.pop("contacts", [])
        contacts = instance.contacts.all()

        if contacts_data:
            for contact in contacts:
                contact.delete()

            for contact_data in contacts_data:
                models.WorkerContact.objects.create(worker=instance, **contact_data)

        for contact_data in contacts_data:
            models.WorkerContact.objects.create(worker=instance, **contact_data)

        comments_data = validated_data.pop("comments", [])

        if comments_data:
            for comment in instance.comments.all():
                comment.delete()

            for comment_data in comments_data:
                models.WorkerComment.objects.create(worker=instance, **comment_data)

        return instance