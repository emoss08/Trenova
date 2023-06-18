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
from django.db.models import Prefetch, QuerySet
from django.utils import timezone
from rest_framework import generics, permissions, response, status, views, viewsets
from rest_framework.authtoken.views import ObtainAuthToken
from rest_framework.request import Request

from accounts import models, serializers
from accounts.permissions import ViewAllUsersPermission
from utils.exceptions import InvalidTokenException


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


class UserViewSet(viewsets.ModelViewSet):
    """
    User ViewSet to manage requests to the user endpoint
    """

    serializer_class = serializers.UserSerializer
    queryset = models.User.objects.all()
    search_fields = (
        "username",
        "email",
        "profiles__first_name",
        "profiles__last_name",
    )
    filterset_fields = (
        "is_active",
        "department__name",
        "is_staff",
        "username",
    )
    permission_classes = [permissions.IsAuthenticated, ViewAllUsersPermission]

    def get_queryset(self) -> QuerySet[models.User]:
        """Filter the queryset to only include the current user

        Returns:
            QuerySet[models.User]: Filtered queryset
        """

        queryset: QuerySet[models.User] = (
            self.queryset.filter(
                organization_id=self.request.user.organization_id  # type: ignore
            )
            .select_related(
                "profiles",
            )
            .prefetch_related(
                Prefetch("groups", queryset=Group.objects.only("id", "name")),
                Prefetch(
                    "user_permissions",
                    queryset=Permission.objects.only(
                        "id", "name", "content_type__app_label", "codename"
                    ),
                ),
            )
            .only(
                "last_login",
                "is_superuser",
                "id",
                "organization_id",
                "department_id",
                "is_active",
                "username",
                "email",
                "is_staff",
                "date_joined",
                "online",
                "profiles__created",
                "profiles__modified",
                "profiles__organization_id",
                "profiles__id",
                "profiles__user",
                "profiles__job_title_id",
                "profiles__first_name",
                "profiles__last_name",
                "profiles__profile_picture",
                "profiles__address_line_1",
                "profiles__address_line_2",
                "profiles__city",
                "profiles__state",
                "profiles__zip_code",
                "profiles__phone_number",
                "profiles__is_phone_verified",
            )
        )
        return queryset


class UpdatePasswordView(generics.UpdateAPIView):
    """
    An endpoint for changing password.
    """

    throttle_scope = "auth"
    serializer_class = serializers.ChangePasswordSerializer

    def update(self, request: Request, *args: Any, **kwargs: Any) -> response.Response:
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


class ResetPasswordView(views.APIView):
    """
    An endpoint for changing password.
    """

    throttle_scope = "auth"
    serializer_class = serializers.ResetPasswordSerializer
    permission_classes = [permissions.AllowAny]

    def post(self, request: Request, *args: Any, **kwargs: Any) -> response.Response:
        """Handle update requests

        Args:
            request (Request): Request object
            *args (Any): Arguments
            **kwargs (Any): Keyword Arguments

        Returns:
            Response: Response of the updated user
        """

        serializer = self.serializer_class(data=request.data)
        if serializer.is_valid():
            serializer.save()
            return response.Response(
                {
                    "message": "Password reset successful. Please check your email for the new password."
                },
                status=status.HTTP_200_OK,
            )

        return response.Response(serializer.errors, status=status.HTTP_400_BAD_REQUEST)


class UpdateEmailView(views.APIView):
    """
    An endpoint for changing password.
    """

    throttle_scope = "auth"
    serializer_class = serializers.UpdateEmailSerializer

    def post(self, request: Request, *args: Any, **kwargs: Any) -> response.Response:
        """Handle update requests

        Args:
            request (Request): Request object
            *args (Any): Arguments
            **kwargs (Any): Keyword Arguments

        Returns:
            Response: Response of the updated user
        """

        serializer = self.serializer_class(
            data=request.data, context={"request": request}
        )
        if serializer.is_valid():
            serializer.save()
            return response.Response(
                {"message": "Email successfully changed."},
                status=status.HTTP_200_OK,
            )

        return response.Response(serializer.errors, status=status.HTTP_400_BAD_REQUEST)


class JobTitleViewSet(viewsets.ModelViewSet):
    """
    Job Title ViewSet to manage requests to the job title endpoint
    """

    serializer_class = serializers.JobTitleSerializer
    queryset = models.JobTitle.objects.all()
    filterset_fields = ["is_active", "name"]

    def get_queryset(self) -> QuerySet[models.JobTitle]:
        queryset = self.queryset.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only(
            "id",
            "is_active",
            "description",
            "name",
            "organization_id",
        )
        return queryset


class TokenVerifyView(views.APIView):
    """
    Rest API endpoint for users can verify a token
    """

    permission_classes: list[Any] = []
    serializer_class = serializers.VerifyTokenSerializer
    http_method_names = ["post"]

    def post(self, request: Request, *args: Any, **kwargs: Any) -> response.Response:
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
            token = models.Token.objects.get(key=token)
        except models.Token.DoesNotExist as e:
            raise InvalidTokenException("Token is invalid") from e

        return response.Response({"token": token.key}, status=status.HTTP_200_OK)


class TokenProvisionView(ObtainAuthToken):
    throttle_scope = "auth"
    permission_classes = (permissions.AllowAny,)
    serializer_class = serializers.TokenProvisionSerializer

    def post(self, request: Request, *args: Any, **kwargs: Any) -> response.Response:
        serializer = self.serializer_class(data=request.data)
        serializer.is_valid(raise_exception=True)
        user = serializer.validated_data["user"]
        token, _ = models.Token.objects.get_or_create(user=user)

        if token.is_expired:
            token.delete()
            token = models.Token.objects.create(user=user)

        user.online = True
        user.last_login = timezone.now()
        user.save()

        return response.Response(
            {
                "token": token.key,
                "user_id": user.id,
                "organization_id": user.organization_id,
            },
            status=status.HTTP_200_OK,
        )


class UserLogoutView(views.APIView):
    """
    Rest API endpoint for users can logout
    """

    permission_classes = [permissions.IsAuthenticated]
    http_method_names = ["post"]

    def post(self, request: Request, *args: Any, **kwargs: Any) -> response.Response:
        """Handle Post requests

        Args:
            request (Request): Request object
            *args (Any): Arguments
            **kwargs (Any): Keyword Arguments

        Returns:
            Response: Response of token and user id
        """

        user = request.user
        models.Token.objects.filter(user=user).delete()

        user.online = False  # type: ignore
        user.save()

        return response.Response(status=status.HTTP_204_NO_CONTENT)
