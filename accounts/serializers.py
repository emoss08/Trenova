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
from typing import Any, OrderedDict

from django.contrib.auth import authenticate
from rest_framework import serializers
from rest_framework.authtoken.models import Token

from accounts import models


class CreateTokenSerializer(serializers.Serializer):
    """
    Create Token Serializer
    """

    username = serializers.CharField()
    password = serializers.CharField()

    def validate(self, attrs: OrderedDict[str, Any]) -> dict[str, Any]:
        """Validate the username and password

        Get the username and password from the
        request and authenticate the user.
        If the user is authenticated, create a
        token for the user and return it.


        Args:
            attrs (OrderedDict[str, Any]): The request data

        Returns:

        """
        username = attrs.get("username")
        password = attrs.get("password")

        user = authenticate(
            request=self.context.get("request"),
            username=username,
            password=password,
        )

        if not user:
            msg = "Unable to authenticate with provided credentials."
            raise serializers.ValidationError(msg, code="authentication")

        token = Token.objects.get(user=user).key

        if token:
            return {"token": token}
        else:
            new_token = Token.objects.create(user=user)
            return {"token": new_token.key}


class VerifyTokenSerializer(serializers.Serializer):
    """
    Verify Token Serializer
    """

    token = serializers.CharField()

    def validate(self, attrs: OrderedDict[str, Any]) -> dict[str, Any]:
        """Validate the token

        Args:
            attrs (OrderedDict): Attributes

        Returns:
            dict[str, Any]: Validated attributes
        """
        token = attrs.get("token")

        if Token.objects.filter(key=token).exists():
            return {"token": token}
        else:
            msg = "Unable to authenticate with provided credentials."
            raise serializers.ValidationError(msg, code="authentication")


class UserProfileSerializer(serializers.ModelSerializer):
    """
    User Profile Serializer
    """

    address = serializers.SerializerMethodField("get_address")

    def get_address(self, obj: models.UserProfile) -> str:
        """Get the address

        Args:
            obj (models.User): The user

        Returns:
            str: The address
        """
        return obj.get_full_address_combo

    class Meta:
        model = models.UserProfile
        fields = [
            "user",
            "first_name",
            "last_name",
            "title",
            "address",
            "address_line_1",
            "address_line_2",
            "city",
            "state",
            "zip_code",
            "phone",
        ]


class UserSerializer(serializers.ModelSerializer):
    """
    User Serializer
    """

    profile = UserProfileSerializer()
    organization = serializers.SerializerMethodField("get_organization")

    def get_organization(self, obj: models.User) -> str:
        """Get the organization of the user

        Args:
            obj (models.User): The user

        Returns:
            str: The organization
        """
        return obj.organization.name

    class Meta:
        """
        Metaclass for UserSerializer
        """

        model: type[models.User] = models.User
        fields = (
            "id",
            "organization",
            "department",
            "username",
            "email",
            "is_staff",
            "is_active",
            "date_joined",
            "profile",
        )
