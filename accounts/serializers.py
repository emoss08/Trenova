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

from django.contrib.auth import password_validation
from django.utils.translation import gettext_lazy as _
from drf_writable_nested.serializers import WritableNestedModelSerializer
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


class JobTitleSerializer(ValidatedSerializer):
    """
    Job Title Serializer
    """

    is_active = serializers.BooleanField(required=False, default=True)

    class Meta:
        """
        Metaclass for JobTitleSerializer
        """

        model = models.JobTitle
        fields = ["id", "organization", "name", "description", "is_active"]


class UserProfileSerializer(WritableNestedModelSerializer):
    """
    User Profile Serializer
    """

    class Meta:
        """
        Metaclass for UserProfileSerializer
        """

        model = models.UserProfile
        fields = [
            "pk",
            "first_name",
            "last_name",
            "title",
            "address_line_1",
            "address_line_2",
            "city",
            "state",
            "zip_code",
            "phone",
        ]


class UserSerializer(WritableNestedModelSerializer, ValidatedSerializer):
    """
    User Serializer
    """

    profile = UserProfileSerializer()

    def update(self, instance: models.User, validated_data: dict[str, Any]) -> None:
        """Update a user

        From validated_data, pop the profile, and update the user profile
        with the profile data. Then, update the user with the remaining
        data. Finally, save the user. DRF does not support nested
        serializers, so this is a workaround.

        Args:
            instance (models.User): User instance
            validated_data (dict[str, Any]): Validated data

        Returns:
            None
        """
        profile_data = validated_data.pop("profile")
        profile = instance.profile

        for key, value in profile_data.items():
            setattr(profile, key, value)
        profile.save()

        return super().update(instance, validated_data) # type: ignore

    class Meta:
        """
        Metaclass for UserSerializer
        """

        model = models.User
        fields = (
            "pk",
            "organization",
            "department",
            "username",
            "password",
            "email",
            "is_staff",
            "is_active",
            "date_joined",
            "profile",
        )
        extra_kwargs = {
            "password": {"write_only": True, "required": False},
            "is_staff": {"read_only": True},
            "is_active": {"read_only": True},
            "date_joined": {"read_only": True},
        }
        extra_validators = {
            "username": [
                lambda value: models.User.objects.filter(username=value).count() == 0,
                "Username already exists",
            ],
            "email": [
                lambda value: models.User.objects.filter(email=value).count() == 0,
                "Email already exists",
            ],
        }


class ChangePasswordSerializer(serializers.Serializer):
    """
    Change Password Serializer
    """

    old_password = serializers.CharField(required=True)
    new_password = serializers.CharField(required=True)
    confirm_password = serializers.CharField(required=True)

    def validate_old_password(self, value: str) -> str:
        """Validate the new password

        Args:
            value (str): New password

        Returns:
            str: Validated new password
        """

        user = self.context["request"].user
        if not user.check_password(value):
            raise serializers.ValidationError(_("Old password is incorrect"))  # type: ignore
        return value

    def validate(self, data: dict[str, Any]) -> dict[str, Any]:
        """

        Args:
            data (dict[str, Any]): Data to validate

        Returns:
            dict[str, Any]: Validated data
        """

        if data["new_password"] != data["confirm_password"]:
            raise serializers.ValidationError(_("Passwords do not match"))  # type: ignore
        password_validation.validate_password(
            data["new_password"], self.context["request"].user
        )
        return data

    def save(self, **kwargs: Any) -> models.User:
        """Save the new password

        Args:
            **kwargs (Any): Keyword arguments

        Returns:
            models.User: User instance
        """

        password = self.validated_data["new_password"]
        user = self.context["request"].user
        user.set_password(password)
        user.save()
        return user


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
        fields = ["pk", "user", "created", "expires", "last_used", "key", "description"]


class TokenProvisionSerializer(serializers.Serializer):
    """
    Token Provision Serializer
    """

    username = serializers.CharField()
    password = serializers.CharField()
