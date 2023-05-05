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

from django.db.models import QuerySet, Prefetch

from utils.views import OrganizationMixin
from worker import models, serializers


class WorkerCommentViewSet(OrganizationMixin):
    queryset = models.WorkerComment.objects.all()
    serializer_class = serializers.WorkerCommentSerializer

    def get_queryset(self) -> QuerySet[models.WorkerComment]:
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).only(
            "id",
            "organization_id",
            "comment_type_id",
            "comment",
            "entered_by_id",
            "worker_id",
        )
        return queryset


class WorkerContactViewSet(OrganizationMixin):
    queryset = models.WorkerContact.objects.all()
    serializer_class = serializers.WorkerContactSerializer

    def get_queryset(self) -> QuerySet[models.WorkerContact]:
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
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


class WorkerProfileViewSet(OrganizationMixin):
    queryset = models.WorkerProfile.objects.all()
    serializer_class = serializers.WorkerProfileSerializer

    def get_queryset(self) -> QuerySet[models.WorkerProfile]:
        queryset = self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
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


class WorkerViewSet(OrganizationMixin):
    """A viewset for viewing and editing workers in the system.

    The viewset provides default operations for creating, updating, and deleting workers,
    as well as listing and retrieving workers. It uses the `WorkerSerializer` class to
    convert the worker instances to and from JSON-formatted data.
    """

    queryset = models.Worker.objects.all()
    serializer_class = serializers.WorkerSerializer

    def get_queryset(self) -> QuerySet[models.Worker]:
        """Returns a queryset of workers for the current user's organization.

        The queryset includes related fields such as profiles, manager(user), depot, organization,
        entered_by(user). It also prefetches related comments and contacts.

        Returns:
            QuerySet[models.Worker]: A queryset of workers for the current user's organization.
        """
        queryset = (
            self.queryset.filter(organization=self.request.user.organization)  # type: ignore
            .select_related(
                "profiles",
            )
            .prefetch_related(
                Prefetch(
                    lookup="comments",
                    queryset=models.WorkerComment.objects.filter(
                        organization=self.request.user.organization  # type: ignore
                    ).only(
                        "id",
                        "entered_by_id",
                        "organization_id",
                        "comment",
                        "comment_type_id",
                        "worker_id",
                    ),
                ),
                Prefetch(
                    lookup="contacts",
                    queryset=models.WorkerContact.objects.filter(
                        organization=self.request.user.organization  # type: ignore
                    ).only(
                        "id",
                        "phone",
                        "organization_id",
                        "name",
                        "worker_id",
                        "is_primary",
                        "relationship",
                        "email",
                        "mobile_phone",
                    ),
                ),
            )
            .only(
                "id",
                "city",
                "state",
                "address_line_1",
                "address_line_2",
                "is_active",
                "worker_type",
                "manager_id",
                "entered_by_id",
                "first_name",
                "last_name",
                "zip_code",
                "code",
                "fleet_id",
                "depot_id",
                "organization_id",
                "profiles__license_expiration_date",
                "profiles__hazmat_expiration_date",
                "profiles__sex",
                "profiles__review_date",
                "profiles__mvr_due_date",
                "profiles__medical_cert_date",
                "profiles__license_number",
                "profiles__date_of_birth",
                "profiles__race",
                "profiles__license_state",
                "profiles__organization_id",
                "profiles__physical_due_date",
                "profiles__worker_id",
                "profiles__hire_date",
                "profiles__endorsements",
                "profiles__hm_126_expiration_date",
                "profiles__termination_date",
            )
        )
        return queryset
