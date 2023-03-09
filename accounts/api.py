# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  Monta is free software: you can redistribute it and/or modify                                   -
#  it under the terms of the GNU General Public License as published by                            -
#  the Free Software Foundation, either version 3 of the License, or                               -
#  (at your option) any later version.                                                             -
#                                                                                                  -
#  Monta is distributed in the hope that it will be useful,                                        -
#  but WITHOUT ANY WARRANTY; without even the implied warranty of                                  -
#  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the                                   -
#  GNU General Public License for more details.                                                    -
#                                                                                                  -
#  You should have received a copy of the GNU General Public License                               -
#  along with Monta.  If not, see <https://www.gnu.org/licenses/>.                                 -
# --------------------------------------------------------------------------------------------------

from typing import Any
from django.db.models import QuerySet
from rest_framework import permissions, status
from rest_framework.authtoken.views import ObtainAuthToken
from rest_framework.generics import UpdateAPIView
from rest_framework.request import Request
from rest_framework.response import Response
from rest_framework.views import APIView
from accounts import models, serializers
from utils.exceptions import InvalidTokenException
from utils.views import OrganizationMixin


class UserViewSet(OrganizationMixin):
    """
    User ViewSet to manage requests to the user endpoint
    """

    serializer_class = serializers.UserSerializer
    queryset = models.User.objects.all()
    filterset_fields = ["department__name", "is_staff"]

    def get_queryset(self) -> QuerySet[models.User]:  # type: ignore
        """Filter the queryset to only include the current user

        Returns:
            QuerySet[models.User]: Filtered queryset
        """

        return (
            self.queryset.filter(organization_id=self.request.user.organization_id)  # type: ignore
            .select_related(
                "organization",
                "profiles",
                "profiles__title",
                "profiles__user",
                "department",
            )
            .values(
                "last_login",
                "is_superuser",
                "id",
                "department_id",
                "username",
                "email",
                "is_staff",
                "date_joined",
                "organization_id",
                "profiles__user",
                "profiles__title",
                "profiles__first_name",
                "profiles__last_name",
                "profiles__profile_picture",
                "profiles__address_line_1",
                "profiles__address_line_2",
                "profiles__city",
                "profiles__state",
                "profiles__phone_number",
                "profiles__zip_code",
                "profiles__is_phone_verified",
            )
        )


class UpdatePasswordView(UpdateAPIView):
    """
    An endpoint for changing password.
    """

    throttle_scope = "auth"
    serializer_class = serializers.ChangePasswordSerializer

    def update(self, request: Request, *args: Any, **kwargs: Any) -> Response:
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
        return Response(
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


class TokenVerifyView(APIView):
    """
    Rest API endpoint for users can verify a token
    """

    permission_classes: list[Any] = []
    serializer_class = serializers.VerifyTokenSerializer

    def post(self, request: Request, *args: Any, **kwargs: Any) -> Response:
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

        return Response(
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

    def post(self, request: Request, *args: Any, **kwargs: Any) -> Response:
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

        return Response(
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
