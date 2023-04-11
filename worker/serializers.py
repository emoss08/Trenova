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

from django.db import OperationalError, transaction
from rest_framework import serializers

from accounts.models import User
from organization.models import Depot, Organization
from utils.serializers import GenericSerializer
from worker import models


class WorkerCommentSerializer(GenericSerializer):
    """
    Worker Comment Serializer
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )

    class Meta:
        """
        Metaclass for WorkerCommentSerializer
        """

        model = models.WorkerComment
        extra_fields = ("organization",)
        extra_read_only_fields = ("worker",)


class WorkerContactSerializer(GenericSerializer):
    """
    Worker Contact Serializer
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )

    class Meta:
        """
        Metaclass for WorkerContactSerializer
        """

        model = models.WorkerContact
        extra_read_only_fields = ("worker",)
        extra_fields = ("organization",)


class WorkerProfileSerializer(GenericSerializer):
    """
    Worker Profile Serializer
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )

    class Meta:
        """
        Metaclass for WorkerProfileSerializer
        """

        model = models.WorkerProfile
        extra_read_only_fields = ("worker",)
        extra_fields = ("organization",)


class WorkerSerializer(GenericSerializer):
    """
    Worker Serializer
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )
    depot = serializers.PrimaryKeyRelatedField(  # type: ignore
        queryset=Depot.objects.all(),
        allow_null=True,
        required=False,
    )
    manager = serializers.PrimaryKeyRelatedField(
        queryset=User.objects.all(),
        allow_null=True,
        required=False,
    )
    entered_by = serializers.PrimaryKeyRelatedField(
        queryset=User.objects.all(),
        allow_null=True,
        required=False,
    )
    is_active = serializers.BooleanField(default=True)
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
            "is_active",
            "depot",
            "manager",
            "entered_by",
            "profile",
            "contacts",
            "comments",
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
        """Updates a worker instance.

        Args:
            instance: A Worker instance to be updated.
            validated_data: Validated data for updating the worker.

        Returns:
            The updated Worker instance.

        Raises:
            serializers.ValidationError: If any errors occur during the update process.

        This function updates a worker instance by first updating the worker's profile,
        comments, and contacts, and then updating the worker instance itself. If comments or
        contacts are included in the validated data, the function will attempt to select and
        lock the associated rows using `select_for_update()`. If `nowait=True`, the function
        will raise a `serializers.ValidationError` immediately if the selected rows are
        locked by another transaction, without waiting for the lock to be released. If any
        errors occur during the update process, the function will raise a
        `serializers.ValidationError` with a helpful error message.
        """
        profile_data = validated_data.pop("profile", {})
        comments_data = validated_data.pop("comments", [])
        contacts_data = validated_data.pop("contacts", [])
        print(contacts_data)

        with transaction.atomic():
            if profile_data:
                instance.profile.update_worker_profile(**profile_data)

            for comment_data in comments_data:
                comment_id = comment_data.get("id", None)
                try:
                    worker_comment = (
                        models.WorkerComment.objects.select_for_update(nowait=True).get(
                            id=comment_id, worker=instance
                        )
                        if comment_id
                        else None
                    )
                except models.WorkerComment.DoesNotExist as e:
                    raise serializers.ValidationError(
                        {
                            "comments": f"Worker comment with id '{comment_id}' does not exist. Delete the ID and try again."
                        }
                    ) from e
                except OperationalError as e:
                    raise serializers.ValidationError(
                        {
                            "comments": "Worker comment is locked by another transaction. Try again later."
                        }
                    ) from e

                if worker_comment:
                    worker_comment.update_worker_comment(**comment_data)
                else:
                    comment_data["organization"] = instance.organization
                    instance.comments.create(**comment_data)

            for contact_data in contacts_data:
                print("contact_data", contact_data)
                contact_id = contact_data.get("id", None)
                try:
                    worker_contact = (
                        models.WorkerContact.objects.select_for_update(nowait=True).get(
                            id=contact_id, worker=instance
                        )
                        if contact_id
                        else None
                    )
                except models.WorkerContact.DoesNotExist as exc:
                    raise serializers.ValidationError(
                        {
                            "contacts": f"Worker contact with id '{contact_id}' does not exist. Delete the ID and try again."
                        }
                    ) from exc
                except OperationalError as exc:
                    raise serializers.ValidationError(
                        {
                            "contacts": "Worker contact is locked by another transaction. Try again later."
                        }
                    ) from exc

                if worker_contact:
                    worker_contact.update_worker_contact(**contact_data)
                else:
                    contact_data["organization"] = instance.organization
                    instance.contacts.create(**contact_data)

            instance.update_worker(**validated_data)

        return instance
