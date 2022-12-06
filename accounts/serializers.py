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

from rest_framework import serializers

from accounts import models
from utils.serailizers import ValidatedSerializer


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

        if models.Token.objects.filter(key=token).exists():
            # Query the user from the token and return the ID of the user
            return {
                "token": token,
                "user_id": models.Token.objects.get(key=token).user.id,
            }
        else:
            msg = "Unable to validate given token"
            raise serializers.ValidationError(msg, code="authentication")


class UserProfileSerializer(ValidatedSerializer):
    """
    User Profile Serializer
    """

    user = serializers.PrimaryKeyRelatedField(queryset=models.User.objects.all())
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
        """
        Metaclass for UserProfileSerializer
        """

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


class UserSerializer(ValidatedSerializer):
    """
    User Serializer
    """

    profile = UserProfileSerializer()

    class Meta:
        """
        Metaclass for UserSerializer
        """

        model = models.User
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


    def update(self, instance: models.User, validated_data: dict) -> models.User:
        """Update the user

        Args:
            instance (models.User): The user
            validated_data (dict): The validated data

        Returns:
            models.User: The updated user
        """

        profile = validated_data.pop("profile")
        profile_instance = instance.profile
        profile_instance.first_name = profile.get("first_name")
        profile_instance.last_name = profile.get("last_name")
        profile_instance.title = profile.get("title")
        profile_instance.address_line_1 = profile.get("address_line_1")
        profile_instance.address_line_2 = profile.get("address_line_2")
        profile_instance.city = profile.get("city")
        profile_instance.state = profile.get("state")
        profile_instance.zip_code = profile.get("zip_code")
        profile_instance.phone = profile.get("phone")
        profile_instance.save()

        return super().update(instance, validated_data)


class TokenSerializer(ValidatedSerializer):
    """
    Serializer for Token model
    """

    key = serializers.CharField(
        min_length=40, max_length=40, allow_blank=True, required=False
    )
    user = UserSerializer()

    class Meta:
        """
        Metaclass for TokenSerializer
        """

        model: type[models.Token] = models.Token
        fields = ("id", "user", "created", "expires", "last_used", "key", "description")


class TokenProvisionSerializer(serializers.Serializer):
    """
    Token Provision Serializer
    """

    username = serializers.CharField()
    password = serializers.CharField()
