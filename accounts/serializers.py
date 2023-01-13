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

from django.contrib.auth import password_validation
from django.utils.translation import gettext_lazy as _
from rest_framework import serializers

from accounts import models
from organization.models import Department
from utils.serializers import GenericSerializer


class VerifyTokenSerializer(serializers.Serializer):
    """A serializer for token verification.

    The serializer provides a token field. The token field is used to verify the incoming token
    from the user. If the given token is valid then the user is given back the token and the user
    id in the response. Otherwise the user is given an error message.

    Attributes:
        token (serializers.CharField): The token to be verified.

    Methods:
        validate(attrs: Any) -> Any: Validate the token.
    """

    token = serializers.CharField()

    def validate(self, attrs: Any) -> Any:
        """Validate the token.

        Args:
            attrs (Any): Attributes

        Returns:
            dict[str, Any]: Validated attributes
        """

        token = attrs.get("token")

        if models.Token.objects.filter(key=token).exists():
            return {
                "token": token,
                "user_id": models.Token.objects.get(key=token).user.id,
            }
        else:
            raise serializers.ValidationError(
                "Unable to validate given token. Please try again.",
                code="authentication",
            )


class JobTitleSerializer(serializers.ModelSerializer):
    """Serializer for the JobTitle model.

    This serializer converts the JobTitle model into a format that
    can be easily converted to and from JSON, and allows for easy validation
    of the data.

    Attributes:
        is_active (serializers.BooleanField): A boolean field representing the
    """

    is_active = serializers.BooleanField(required=False, default=True)

    class Meta:
        """
        Metaclass for GeneralLedgerAccountSerializer

        Attributes:
            model (models.JobTitle): The model that the serializer
            is for.

            fields (list[str]): The fields that should be included
            in the serialized representation of the model.
        """

        model = models.JobTitle
        fields = ["id", "organization", "name", "description", "is_active"]


class UserProfileSerializer(GenericSerializer):
    """
    User Profile Serializer
    """

    title = serializers.PrimaryKeyRelatedField(
        queryset=models.JobTitle.objects.all(),
        required=False,
        allow_null=True,
    )

    class Meta:
        """
        Metaclass for UserProfileSerializer
        """

        model = models.UserProfile
        extra_fields = ("title",)
        extra_read_only_fields = (
            "id",
            "user",
        )


class UserSerializer(GenericSerializer):
    """
    User Serializer
    """

    department = serializers.PrimaryKeyRelatedField(
        queryset=Department.objects.all(),
        allow_null=True,
    )
    profile = UserProfileSerializer(required=False, allow_null=True)

    class Meta:
        """
        Metaclass for UserSerializer
        """

        model = models.User
        extra_fields = ("profile",)
        extra_kwargs = {
            "is_staff": {"read_only": True},
            "is_active": {"read_only": True},
        }

    def create(self, validated_data: Any) -> models.User:  # type: ignore
        """Create a user

        Args:
            validated_data (Any): Validated data

        Returns:
            models.User: User instance
        """

        # Get the organization of the user from the request.
        organization = super().get_organization

        # Popped data (profile)
        validated_data["organization"] = organization
        profile_data = validated_data.pop("profile", {})

        # Create the user
        user: models.User = models.User.objects.create(**validated_data)

        # Create the user profile
        if profile_data:
            models.UserProfile.objects.create(user=user, **profile_data)

        return user

    def update(self, instance: models.User, validated_data: Any) -> models.User:  # type: ignore
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

        profile_data = validated_data.pop("profile", None)

        if profile_data:
            instance.profile.update_profile(**profile_data)

        instance.update_user(**validated_data)

        return instance


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
            raise serializers.ValidationError(_("Old password is incorrect. Please try again."))  # type: ignore
        return value

    def validate(self, attrs: Any) -> Any:
        """Validate the new password

        Args:
            attrs (Any): Data to validate

        Returns:
            dict[str, Any]: Validated data
        """

        if attrs["new_password"] != attrs["confirm_password"]:
            raise serializers.ValidationError(_("Passwords do not match. Please try again."))  # type: ignore
        password_validation.validate_password(
            attrs["new_password"], self.context["request"].user
        )
        return attrs

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


class TokenSerializer(serializers.ModelSerializer):
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
        fields = ["id", "user", "created", "expires", "last_used", "key", "description"]


class TokenProvisionSerializer(serializers.Serializer):
    """
    Token Provision Serializer
    """

    username = serializers.CharField()
    password = serializers.CharField()
