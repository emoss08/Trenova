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
from rest_framework import permissions
from rest_framework.exceptions import AuthenticationFailed
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
    serializer_class: type[serializers.UserSerializer] = serializers.UserSerializer
    queryset = models.User.objects.all().select_related("organization")

    def get_queryset(self) -> QuerySet[models.User]:
        """Filter the queryset to only include the current user

        Returns:
            QuerySet[models.User]: Filtered queryset
        """

        return self.queryset.filter(organization=self.request.user.organization.id).select_related(  # type: ignore
            "organization",
            "profiles",
            "profiles__title",
            "department",
        )


class TokenProvisionView(APIView):
    """
    Rest API endpoint for users can create a token
    """

    permission_classes: list[Any] = []

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
            raise AuthenticationFailed("Username or password is not provided")
        user = authenticate(username=username, password=password)

        if not user:
            raise AuthenticationFailed("Invalid username or password")

        token = models.Token(user=user)  # type: ignore
        token.save()

        # Return the token and full user details
        return Response(
            {
                "token": token.key,
                "user": serializers.UserSerializer(user).data,
            },
        )


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
            token = models.Token.objects.get(key=token)
        except models.Token.DoesNotExist:
            raise InvalidTokenException("Token is invalid")

        return Response(
            {
                "token": token.key,
                "user": serializers.UserSerializer(token.user).data,
            },
        )
