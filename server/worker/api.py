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

import typing

from django.db.models import Prefetch, QuerySet
from rest_framework import response, status, viewsets

from core.permissions import CustomObjectPermissions
from worker import models, serializers

if typing.TYPE_CHECKING:
    pass


class WorkerCommentViewSet(viewsets.ModelViewSet):
    queryset = models.WorkerComment.objects.all()
    serializer_class = serializers.WorkerCommentSerializer
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.WorkerComment]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "organization_id",
            "comment_type_id",
            "comment",
            "entered_by_id",
            "worker_id",
        )
        return queryset


class WorkerContactViewSet(viewsets.ModelViewSet):
    queryset = models.WorkerContact.objects.all()
    serializer_class = serializers.WorkerContactSerializer
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.WorkerContact]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "name",
            "organization_id",
            "mobile_phone",
            "worker_id",
            "is_primary",
            "relationship",
            "phone",
            "email",
        )
        return queryset


class WorkerProfileViewSet(viewsets.ModelViewSet):
    queryset = models.WorkerProfile.objects.all()
    serializer_class = serializers.WorkerProfileSerializer
    permission_classes = [CustomObjectPermissions]

    def get_queryset(self) -> QuerySet[models.WorkerProfile]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "organization_id",
            "worker_id",
            "physical_due_date",
            "hazmat_expiration_date",
            "license_number",
            "date_of_birth",
            "termination_date",
            "hire_date",
            "race",
            "mvr_due_date",
            "sex",
            "license_expiration_date",
            "hm_126_expiration_date",
            "medical_cert_date",
            "review_date",
            "license_state",
            "endorsements",
        )

        return queryset


class WorkerViewSet(viewsets.ModelViewSet):
    queryset = models.Worker.objects.all()
    serializer_class = serializers.WorkerSerializer
    permission_classes = [
        CustomObjectPermissions
    ]  # Replace with your actual permission classes
    filterset_fields = ("profiles__endorsements", "manager", "fleet_code")
    search_fields = ("first_name", "last_name", "code", "profiles__license_number")

    def create(self, request, *args, **kwargs):
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        self.perform_create(serializer)
        headers = self.get_success_headers(serializer.data)

        # Re-fetch the worker with related data
        worker = models.Worker.objects.get(pk=serializer.instance.pk)
        response_serializer = self.get_serializer(worker)

        return response.Response(
            response_serializer.data, status=status.HTTP_201_CREATED, headers=headers
        )

    def get_queryset(self):
        user_org = self.request.user.organization_id  # type: ignore

        # Fetch latest WorkerHOS IDs
        latest_hos_ids = (
            models.WorkerHOS.objects.filter(worker__organization_id=user_org)
            .order_by("worker_id", "-log_date")
            .distinct("worker_id")
            .values_list("id", flat=True)
        )

        # Fetch all relevant WorkerHOS records in one query
        relevant_hos_records = models.WorkerHOS.objects.filter(id__in=latest_hos_ids)

        queryset = (
            self.queryset.filter(organization_id=user_org)
            .select_related("profiles")
            .prefetch_related(
                "comments",
                "contacts",
                Prefetch(
                    "worker_hos", queryset=relevant_hos_records, to_attr="latest_hos"
                ),
            )
        )
        return queryset
