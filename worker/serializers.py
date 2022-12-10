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

from drf_writable_nested import WritableNestedModelSerializer
from rest_framework import serializers

from worker import models


class WorkerCommentSerializer(serializers.ModelSerializer):
    """
    Worker Comment Serializer
    """

    class Meta:
        """
        Metaclass for WorkerCommentSerializer
        """

        model = models.WorkerComment
        fields = [
            "organization",
            "id",
            "worker",
            "comment_type",
            "comment",
            "entered_by",
            "created",
            "modified",
        ]


class WorkerContactSerializer(serializers.ModelSerializer):
    """
    Worker Contact Serializer
    """

    class Meta:
        """
        Metaclass for WorkerContactSerializer
        """

        model = models.WorkerContact
        fields = [
            "organization",
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
            "created",
            "modified",
        ]


class WorkerProfileSerializer(serializers.ModelSerializer):
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
            "organization",
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
            "created",
            "modified",
            "hire_date",
        ]


class WorkerSerializer(WritableNestedModelSerializer):
    """
    Worker Serializer
    """

    profile = WorkerProfileSerializer(required=False)
    contacts = WorkerContactSerializer(many=True, required=False)
    comments = WorkerCommentSerializer(many=True, required=False)

    def create(self, validated_data):
        """
        Create the worker
        """

        profile_data = validated_data.pop("profile", [])
        contacts_data = validated_data.pop("contacts", [])
        comments_data = validated_data.pop("comments", [])

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

    def update(self, instance, validated_data) -> None:
        """
        Update the worker
        """

        profile_data = validated_data.pop("profile")
        profile = instance.profile

        for key, value in profile_data.items():
            setattr(profile, key, value)
        profile.save()

        return super().update(instance, validated_data)  # type: ignore

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
            "created",
            "modified",
        ]
