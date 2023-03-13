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

from django.contrib.auth.models import Group, Permission
from django.db.models import QuerySet
from rest_framework import (
    generics,
    permissions,
    request,
    response,
    status,
    views,
    viewsets,
)
from rest_framework.authtoken.views import ObtainAuthToken

from accounts import models, serializers
from utils.exceptions import InvalidTokenException
from utils.permissions import MontaModelPermissions
from utils.views import OrganizationMixin


class GroupViewSet(viewsets.ModelViewSet):
    """
    Group ViewSet to manage requests to the group endpoint
    """

    serializer_class = serializers.GroupSerializer
    queryset = Group.objects.all()
    filterset_fields = ["name"]
    ordering_fields = "__all__"
    permission_classes = [permissions.IsAuthenticated]


class PermissionViewSet(viewsets.ModelViewSet):
    """
    Permission ViewSet to manage requests to the permission endpoint
    """

    serializer_class = serializers.PermissionSerializer
    queryset = Permission.objects.all()
    filterset_fields = ["name"]
    ordering_fields = "__all__"
    permission_classes = [permissions.IsAuthenticated]


class UserViewSet(OrganizationMixin):
    """
    User ViewSet to manage requests to the user endpoint
    """

    serializer_class = serializers.UserSerializer
    queryset = models.User.objects.all()
    search_fields = ["username", "email", "profiles__first_name", "profiles__last_name"]
    filterset_fields = ["department__name", "is_staff", "username"]
    ordering_fields = "__all__"
    permission_classes = [MontaModelPermissions]

    def get_queryset(self) -> QuerySet[models.User]:  # type: ignore
        """Filter the queryset to only include the current user

        Returns:
            QuerySet[models.User]: Filtered queryset
        """

        return self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).select_related(
            "organization",
            "profiles",
            "profiles__title",
            "profiles__user",
            "department",
        )


class UpdatePasswordView(generics.UpdateAPIView):
    """
    An endpoint for changing password.
    """

    throttle_scope = "auth"
    serializer_class = serializers.ChangePasswordSerializer

    def update(
        self, request: request.Request, *args: Any, **kwargs: Any
    ) -> response.Response:
        """Handle update requests

        Args:
            request (Request): Request object
            *args (Any): Arguments
            **kwargs (Any): Keyword Arguments

        Returns:
            Response: Response of the updated user
        """

        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        serializer.save()
        return response.Response(
            "Password updated successfully",
            status=status.HTTP_200_OK,
        )


class JobTitleViewSet(OrganizationMixin):
    """
    Job Title ViewSet to manage requests to the job title endpoint
    """

    serializer_class = serializers.JobTitleSerializer
    queryset = models.JobTitle.objects.all()
    filterset_fields = ["is_active", "name"]

    def get_queryset(self) -> QuerySet[models.JobTitle]:
        """Filter the queryset to only include the current user

        Returns:
            QuerySet[models.JobTitle]: Filtered queryset
        """

        return self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).select_related("organization")


class TokenVerifyView(views.APIView):
    """
    Rest API endpoint for users can verify a token
    """

    permission_classes: list[Any] = []
    serializer_class = serializers.VerifyTokenSerializer

    def post(
        self, request: request.Request, *args: Any, **kwargs: Any
    ) -> response.Response:
        """Handle Post requests
        Args:
            request (Request): Request object
            *args (Any): Arguments
            **kwargs (Any): Keyword Arguments
        Returns:
            Response: Response of token and user id
        """

        serializer = self.serializer_class(data=request.data)
        serializer.is_valid(raise_exception=True)

        token = serializer.data.get("token")

        try:
            token = (
                models.Token.objects.select_related("user")
                .only("key", "user__id")
                .get(key=token)
            )
        except models.Token.DoesNotExist as e:
            raise InvalidTokenException("Token is invalid") from e

        user = (
            models.User.objects.select_related(
                "profiles",
                "profiles__title",
                "organization",
                "department",
            )
            .only(
                "id",
                "username",
                "email",
                "last_login",
                "is_staff",
                "is_superuser",
                "organization__id",
                "department__id",
                "profiles__first_name",
                "profiles__last_name",
                "profiles__title__name",
                "profiles__address_line_1",
                "profiles__address_line_2",
                "profiles__city",
                "profiles__state",
                "profiles__zip_code",
                "profiles__phone_number",
                "profiles__is_phone_verified",
            )
            .get(id=token.user.id)
        )

        return response.Response(
            {
                "id": user.id,
                "username": user.username,
                "email": user.email,
                "first_name": user.profile.first_name,
                "last_name": user.profile.last_name,
                "full_name": f"{user.profile.first_name} {user.profile.last_name}",
                "organization_id": user.organization.id,
                "department_id": user.department.id if user.department else None,
                "job_title": user.profile.title.name,
                "is_staff": user.is_staff,
                "is_superuser": user.is_superuser,
                "address_line_1": user.profile.address_line_1,
                "address_line_2": user.profile.address_line_2,
                "city": user.profile.city,
                "state": user.profile.state,
                "zip_code": user.profile.zip_code,
                "full_address": user.profile.get_full_address_combo,
                "phone_number": user.profile.phone_number,
                "phone_verified": user.profile.is_phone_verified,
                "token": token.key,
            }
        )


class TokenProvisionView(ObtainAuthToken):
    throttle_scope = "auth"
    permission_classes = (permissions.AllowAny,)
    serializer_class = serializers.TokenProvisionSerializer

    def post(
        self, request: request.Request, *args: Any, **kwargs: Any
    ) -> response.Response:
        serializer = self.serializer_class(data=request.data)
        serializer.is_valid(raise_exception=True)
        user_obj = serializer.validated_data["user"]
        token, _ = models.Token.objects.get_or_create(user=user_obj)
        user = (
            models.User.objects.select_related(
                "profiles", "profiles__title", "organization", "department"
            )
            .only(
                "id",
                "username",
                "email",
                "profiles__first_name",
                "profiles__last_name",
                "organization__id",
                "department__id",
                "profiles__title__name",
                "is_staff",
                "is_superuser",
                "profiles__address_line_1",
                "profiles__address_line_2",
                "profiles__city",
                "profiles__state",
                "profiles__zip_code",
                "profiles__phone_number",
                "profiles__is_phone_verified",
            )
            .get(id=user_obj.id)
        )
        if token.is_expired:
            token.delete()
            token = models.Token.objects.create(user=user_obj)

        return response.Response(
            {
                "id": user.id,
                "username": user.username,
                "email": user.email,
                "first_name": user.profile.first_name,
                "last_name": user.profile.last_name,
                "full_name": f"{user.profile.first_name} {user.profile.last_name}",
                "organization_id": user.organization.id,
                "department_id": user.department.id if user.department else None,
                "job_title": user.profile.title.name,
                "is_staff": user.is_staff,
                "is_superuser": user.is_superuser,
                "address_line_1": user.profile.address_line_1,
                "address_line_2": user.profile.address_line_2,
                "city": user.profile.city,
                "state": user.profile.state,
                "zip_code": user.profile.zip_code,
                "full_address": user.profile.get_full_address_combo,
                "phone_number": user.profile.phone_number,
                "phone_verified": user.profile.is_phone_verified,
                "token": token.key,
            }
        )
