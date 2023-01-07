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

from django.contrib.auth import authenticate
from django.db.models import QuerySet
from django_filters.rest_framework import DjangoFilterBackend
from rest_framework import permissions, status
from rest_framework.exceptions import AuthenticationFailed
from rest_framework.generics import UpdateAPIView
from rest_framework.request import Request
from rest_framework.response import Response
from rest_framework.views import APIView

from accounts import models, serializers
from utils.exceptions import InvalidTokenException
from utils.views import OrganizationViewSet


class UserViewSet(OrganizationViewSet):
    """
    User ViewSet to manage requests to the user endpoint
    """

    permission_classes = [permissions.IsAuthenticated]
    filter_backends = [DjangoFilterBackend]
    filterset_fields = ["organization", "department", "profile"]
    serializer_class = serializers.UserSerializer
    queryset = models.User.objects.all()

    def get_queryset(self) -> QuerySet[models.User]:  # type: ignore
        """Filter the queryset to only include the current user

        Returns:
            QuerySet[models.User]: Filtered queryset
        """

        return self.queryset.filter(organization=self.request.user.organization).select_related(  # type: ignore
            "organization",
            "profile",
            "profile__title",
            "profile__user",
            "department",
        )


class UpdatePasswordView(UpdateAPIView):
    """
    An endpoint for changing password.
    """

    serializer_class = serializers.ChangePasswordSerializer

    def update(self, request: Request, *args: Any, **kwargs: Any) -> Response:
        """Update the password

        Args:
            request (Request): The request object
            *args (Any): Arguments
            **kwargs (Any): Keyword arguments

        Returns:
            Response: The response object
        """

        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        user = serializer.save()

        if hasattr(user, "token"):
            user.token.delete()

        token, created = models.Token.objects.get_or_create(user=user)
        return Response(
            {
                "token": token.key,
            },
            status=status.HTTP_200_OK,
        )


class TokenProvisionView(APIView):
    """
    Rest API endpoint for users can create a token
    """

    permission_classes = []

    def post(self, request: Request, *args: Any, **kwargs: Any) -> Response:
        """Handle Post requests

        Args:
            request (Request): Request object
            *args (Any): Arguments
            **kwargs (Any): Keyword Arguments

        Returns:
            Response: Response of token and user id
        """

        serializer = serializers.TokenProvisionSerializer(data=request.data)
        serializer.is_valid()

        username = serializer.data.get("username")
        password = serializer.data.get("password")

        if not username or not password:
            raise AuthenticationFailed({"message": "Username or password is missing"})
        user = authenticate(username=username, password=password)

        if not user:
            raise AuthenticationFailed({"message": "Invalid credentials"})

        token, created = models.Token.objects.get_or_create(user=user)

        # if the token is expired then create a new one for the user rather than returning the old one
        if token.is_expired:
            token.delete()
            token = models.Token.objects.create(user=user)  # type: ignore

        return Response(
            {
                "user_id": user.pk,
                "api_token": token.key,
            },
            status=status.HTTP_200_OK,
        )


class TokenVerifyView(APIView):
    """
    Rest API endpoint for users can verify a token
    """

    permission_classes = []
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
            token = models.Token.objects.get(key=token)
        except models.Token.DoesNotExist:
            raise InvalidTokenException("Token is invalid")

        return Response(
            {
                "user_id": token.user.id,
                "api_token": token.key,
            },
            status=status.HTTP_200_OK,
        )


class JobTitleViewSet(OrganizationViewSet):
    """
    Job Title ViewSet to manage requests to the job title endpoint
    """

    permission_classes = (permissions.IsAuthenticated,)
    serializer_class = serializers.JobTitleSerializer
    queryset = models.JobTitle.objects.all()

    def get_queryset(self) -> QuerySet[models.JobTitle]:
        """Filter the queryset to only include the current user

        Returns:
            QuerySet[models.JobTitle]: Filtered queryset
        """

        return self.queryset.filter(
            organization=self.request.user.organization  # type: ignore
        ).select_related("organization")
