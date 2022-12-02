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

from rest_framework.authtoken.models import Token
from rest_framework.request import Request
from rest_framework.response import Response
from rest_framework.views import APIView
from rest_framework_simplejwt.views import TokenVerifyView, TokenViewBase

from accounts import serializers


class UserView(APIView):
    """
    User Information View
    """

    def get(self, request: Request, *args: Any, **kwargs: Any) -> Response:
        """
        Take token as input and return user information
        """
        user = Token.objects.get(key=request.headers["Authorization"][6:]).user
        user_instance = serializers.UserSerializer(user)
        return Response(user_instance.data)
        # user_information = serializers.UserProfileSerializer(user).data
        # return Response(user_information)


class UserProfileView(APIView):
    """
    User Profile View
    """

    def get(self, request: Request, *args: Any, **kwargs: Any) -> Response:
        """
        Take token as input and return user profile information
        """
        user = Token.objects.get(key=request.headers["Authorization"][6:]).user
        user_profile = serializers.UserProfileSerializer(user).data
        return Response(user_profile)


class TokenObtainView(TokenViewBase):
    """
    Token Obtain View
    """
    serializer_class = serializers.CreateTokenSerializer


class GenericTokenVerifyView(TokenVerifyView):
    """
    If the token is valid return it back in the response
    """

    serializer_class = serializers.VerifyTokenSerializer
