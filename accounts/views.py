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

from rest_framework import generics, permissions, status, viewsets
from rest_framework.authtoken.models import Token
from rest_framework.authtoken.serializers import AuthTokenSerializer
from rest_framework.authtoken.views import ObtainAuthToken
from rest_framework.request import Request
from rest_framework.response import Response
from rest_framework_simplejwt.exceptions import InvalidToken, TokenError

from accounts import models, serializers


class UserViewSet(viewsets.ModelViewSet):
    """
    User ViewSet to manage requests to the user endpoint
    """

    permission_classes = [permissions.IsAuthenticated]
    serializer_class: type[serializers.UserSerializer] = serializers.UserSerializer
    queryset = models.User.objects.all()

    def update(self, request, *args, **kwargs):
        """Update the user

        Args:
            request (Request): Request
            *args: Arguments
            **kwargs: Keyword arguments

        Returns:
            Response: Response
        """
        user = request.user
        profile = request.data.pop("profile", None)

        for field in profile:
            if field != "user":
                setattr(user.profile, field, profile[field])
        user.profile.save()
        serializer = self.get_serializer(user, data=request.data, partial=True)
        serializer.is_valid(raise_exception=True)
        self.perform_update(serializer)

        return Response(serializer.data)


class UserProfileViewSet(viewsets.ModelViewSet):
    """
    User ViewSet to manage requests to the user endpoint
    """

    permission_classes = [permissions.IsAuthenticated]
    serializer_class: type[
        serializers.UserProfileSerializer
    ] = serializers.UserProfileSerializer
    queryset = models.UserProfile.objects.all()


class TokenObtainView(ObtainAuthToken):
    """
    Token Obtain View
    """

    def post(self, request: Request, *args: Any, **kwargs: Any) -> Response:
        """Handle Post requests

        Args:
            request (Request): Request object
            *args (Any): Arguments
            **kwargs (Any): Keyword Arguments

        Returns:
            Response: Response of token and user id
        """
        serializer: AuthTokenSerializer = self.serializer_class(data=request.data)
        serializer.is_valid(raise_exception=True)
        user = serializer.validated_data["user"]
        token, created = Token.objects.get_or_create(user=user)
        return Response({"token": token.key, "user_id": user.id})


class TokenVerifyView(generics.GenericAPIView):
    """
    If the token is valid return it back in the response
    """

    serializer_class: type[
        serializers.VerifyTokenSerializer
    ] = serializers.VerifyTokenSerializer

    def post(self, request: Request, *args: Any, **kwargs: Any) -> Response:
        """Verify the token and return it back

        Args:
            request (Request): Request object
            *args (Any): Arguments
            **kwargs (Any): Keyword Arguments

        Returns:
            Response: Response of token and user id
        """

        serializer = self.get_serializer(data=request.data)

        try:
            serializer.is_valid(raise_exception=True)

        except TokenError as token_e:
            raise InvalidToken(token_e.args[0])

        return Response(serializer.validated_data, status=status.HTTP_200_OK)
