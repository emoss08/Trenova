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
from rest_framework import permissions, status, viewsets
from rest_framework.exceptions import AuthenticationFailed
from rest_framework.request import Request
from rest_framework.response import Response
from rest_framework.views import APIView

from accounts import models, serializers
from utils.exceptions import InvalidTokenException


class UserViewSet(viewsets.ModelViewSet):
    """
    User ViewSet to manage requests to the user endpoint
    """

    permission_classes = [permissions.IsAuthenticated]
    serializer_class: type[serializers.UserSerializer] = serializers.UserSerializer
    queryset = models.User.objects.all()


class UserProfileViewSet(viewsets.ModelViewSet):
    """
    User ViewSet to manage requests to the user endpoint
    """

    permission_classes = [permissions.IsAuthenticated]
    serializer_class: type[
        serializers.UserProfileSerializer
    ] = serializers.UserProfileSerializer
    queryset = models.UserProfile.objects.all()


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
            raise AuthenticationFailed("Username or password is not provided")
        user = authenticate(username=username, password=password)

        if not user:
            raise AuthenticationFailed("Invalid username or password")

        token = models.Token(user=user)
        token.save()
        data = {"token": token.key, "user_id": user.id}

        return Response(data, status=status.HTTP_200_OK)


class TokenVerifyView(APIView):
    """
    Rest API endpoint for users can verify a token
    """

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

        data = {"token": token.key, "user_id": token.user.id}

        return Response(data, status=status.HTTP_200_OK)
