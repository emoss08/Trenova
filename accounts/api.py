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

from typing import Any, Optional

from django.contrib.auth import login
from django.db.models import QuerySet
from knox.views import LoginView as KnoxLoginView
from rest_framework import permissions, status
from rest_framework.authtoken.serializers import AuthTokenSerializer
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

        return self.queryset.filter(organization=self.request.user.organization).select_related(  # type: ignore
            "organization",
            "profiles",
            "profiles__title",
            "profiles__user",
            "department",
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


class LoginView(KnoxLoginView):
    """
    A Django view that handles user authentication and login using token-based authentication provided by KnoxLoginView.

    This view is accessible to all users, regardless of whether they are authenticated or not. It accepts HTTP POST requests
    containing user data, validates the user data using the `AuthTokenSerializer`, logs the user in using Django's built-in
    `login()` function, and returns an HTTP response object containing the authentication token.

    Attributes:
        permission_classes (tuple): A tuple of permission classes that allow any user to access this view.

    Methods:
        post(request, format=None): Handle user authentication and login using an HTTP POST request.
    """

    permission_classes = (permissions.AllowAny,)

    def post(self, request: Request, format: Optional[str] = None) -> Response:
        """
        Handle user authentication and login using an HTTP POST request.

        Args:
            request (Request): The HTTP request object containing user data.
            format (str, optional): The format of the response data (default=None).

        Returns:
            Response: The HTTP response object containing the authentication token.

        Raises:
            ValidationError: If the user data is invalid.

        Notes:
            This method validates the user data using the `AuthTokenSerializer` and logs the user in using Django's built-in
            `login()` function. The method then returns an HTTP response object containing the authentication token.
        """
        serializer = AuthTokenSerializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        user = serializer.validated_data["user"]
        login(request, user)
        return super(LoginView, self).post(request, format=None)
