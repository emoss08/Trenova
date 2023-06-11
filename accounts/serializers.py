# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
#  to copy, modify, and redistribute the software, but only for non-production use or with a total -
#  of less than three server instances. Starting from the Change Date (November 16, 2026), the     -
#  software will be made available under version 2 or later of the GNU General Public License.     -
#  If you use the software in violation of this license, your rights under the license will be     -
#  terminated automatically. The software is provided "as is," and the Licensor disclaims all      -
#  warranties and conditions. If you use this license's text or the "Business Source License" name -
#  and trademark, you must comply with the Licensor's covenants, which include specifying the      -
#  Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use     -
#  Grant, and not modifying the license in any other way.                                          -
# --------------------------------------------------------------------------------------------------

from typing import Any

from django.contrib.auth import authenticate, password_validation
from django.contrib.auth.models import Group, Permission
from django.core.mail import send_mail
from drf_spectacular.utils import OpenApiExample, extend_schema_serializer
from rest_framework import serializers

from accounts import models
from organization.models import Organization
from utils.serializers import GenericSerializer


class PermissionSerializer(serializers.ModelSerializer):
    """
    Permissions Serializer
    """

    class Meta:
        """
        Metaclass for PermissionsSerializer
        """

        model = Permission
        fields = ["id", "name", "codename"]


class GroupSerializer(serializers.ModelSerializer):
    """
    Group Serializer
    """

    permissions = PermissionSerializer(many=True, read_only=True)

    class Meta:
        """
        Metaclass for GroupSerializer
        """

        model = Group
        fields = ["id", "name", "permissions"]


class JobTitleSerializer(GenericSerializer):
    """Serializer for the JobTitle model.

    This serializer converts the JobTitle model into a format that
    can be easily converted to and from JSON, and allows for easy validation
    of the data.

    Attributes:
        is_active (serializers.BooleanField): A boolean field representing the
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )
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


class JobTitleListingField(serializers.RelatedField):
    """
    Serializes a related field for a JobTitle object, returning only the name of the instance.
    """

    def to_representation(self, instance: models.JobTitle) -> str:
        """Converts a complex data instance into a primitive representation.

        Args:
            self: The serializer instance.
            instance: The data object to serialize.

        Returns:
            A serialized representation of the data object, typically a dictionary or a list of dictionaries.

        Raises:
            SerializerError: If there is an error during the serialization process.

        Example Usage:
            # In a serializer class definition
            class BookSerializer(serializers.ModelSerializer):
                author = AuthorSerializer()

        class Meta:
            model = Book
            fields = ('id', 'title', 'author')

        def to_representation(self, instance):
            representation = super().to_representation(instance)
            representation['title'] = representation['title'].upper()
            return representation
        """
        return instance.name


class UserProfileSerializer(GenericSerializer):
    """Serializes and deserializes instances of the UserProfile model.

    The UserProfileSerializer defines two fields for serializing and
    deserializing instances of the UserProfile model: `title` and `title_name`.
    The `title` field is a `PrimaryKeyRelatedField` that can be used to set the
    JobTitle associated with a UserProfile instance. The `title_name` field is a
    `JobTitleListingField` that returns only the name of the associated JobTitle as
    a read-only field.

    Attributes:
        job_title: A `PrimaryKeyRelatedField` that allows setting the associated JobTitle
        for a UserProfile instance.
        title_name: A `JobTitleListingField` that returns only the name of the associated
        JobTitle as a read-only field.

    Methods:
        to_representation(self, instance):
            Converts a UserProfile instance to a serialized representation.

        to_internal_value(self, data):
            Converts a serialized representation of a UserProfile to an instance of the model.

    Meta:
        model: The UserProfile model that the serializer should be based on.
        extra_fields: A tuple of field names that should be included in the serialized
        representation.
        extra_read_only_fields: A tuple of field names that should be read-only in the
        serialized representation.

    Example Usage:
        # In a view
        class UserProfileView(generics.RetrieveUpdateAPIView):
            serializer_class = UserProfileSerializer
            queryset = models.UserProfile.objects.all()

        # In a serializer class definition
        class UserProfileSerializer(GenericSerializer):
            job_title = serializers.PrimaryKeyRelatedField(
                queryset=models.JobTitle.objects.all(),
                required=False,
                allow_null=True,
            )

            title_name = JobTitleListingField(
                source="title",
                read_only=True,
            )

            class Meta:
                model = models.UserProfile
                extra_fields = ("title", "title_name")
                extra_read_only_fields = (
                    "id",
                    "user",
                )
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )
    job_title = serializers.PrimaryKeyRelatedField(
        queryset=models.JobTitle.objects.all(),
        required=False,
        allow_null=True,
    )

    title_name = JobTitleListingField(
        source="title",
        read_only=True,
    )

    class Meta:
        """Metadata for a Django REST framework serializer.

        The `Meta` class allows you to specify metadata about a serializer class, such as the model
        it should be based on, any additional fields to include in the serialized representation,
        and any read-only fields.

        Attributes:
            model: The model that the serializer should be based on.
            extra_fields: A tuple of field names that should be included in the serialized representation
            in addition to the fields defined on the model.
            extra_read_only_fields: A tuple of field names that should be read-only in the serialized
            representation.

        Example Usage:
            # In a serializer class definition
            class BookSerializer(serializers.ModelSerializer):
                author = AuthorSerializer()

                class Meta:
                    model = Book
                    fields = ('id', 'title', 'author')
                    extra_read_only_fields = ('id',)
        """

        model = models.UserProfile
        extra_fields = ("job_title", "title_name", "organization")
        extra_read_only_fields = (
            "id",
            "user",
        )


@extend_schema_serializer(
    examples=[
        OpenApiExample(
            "User Request",
            summary="User Request",
            value={
                "id": "b08e6e3f-28da-47cf-ad48-99fc7919c087",
                "department": "7eaaca59-7e58-4398-82e9-d6d9321d483d",
                "username": "test_user",
                "email": "test_user@example.com",
                "password": "test_password",
                "profile": {
                    "id": "a75a4b66-3f3a-48af-a089-4b7f1373f7a1",
                    "user": "b08e6e3f-28da-47cf-ad48-99fc7919c087",
                    "job_title": "bfa74d30-915f-425a-b957-15b826c3bee2",
                    "first_name": "Example",
                    "last_name": "User",
                    "profile_picture": None,
                    "address_line_1": "123 Example Location",
                    "address_line_2": "Unit 123",
                    "city": "San Antonio",
                    "state": "TX",
                    "zip_code": "12345",
                    "phone": "12345678903",
                },
            },
            request_only=True,
        ),
        OpenApiExample(
            "User Response",
            summary="User Response",
            value={
                "last_login": "2023-01-26T19:17:37.565110Z",
                "is_superuser": False,
                "id": "b08e6e3f-28da-47cf-ad48-99fc7919c087",
                "department": "7eaaca59-7e58-4398-82e9-d6d9321d483d",
                "username": "test_user",
                "email": "test_user@example.com",
                "is_staff": False,
                "date_joined": "2022-12-04T00:05:00Z",
                "groups": [
                    0,
                ],
                "user_permissions": [
                    0,
                ],
                "profile": {
                    "id": "a75a4b66-3f3a-48af-a089-4b7f1373f7a1",
                    "user": "b08e6e3f-28da-47cf-ad48-99fc7919c087",
                    "job_title": "bfa74d30-915f-425a-b957-15b826c3bee2",
                    "first_name": "Example",
                    "last_name": "User",
                    "profile_picture": "http://localhost:8000/media/profile_pictures/placeholder.png",
                    "address_line_1": "123 Example Location",
                    "address_line_2": "Unit 123",
                    "city": "San Antonio",
                    "state": "TX",
                    "zip_code": "12345",
                    "phone": "12345678903",
                },
            },
            response_only=True,
        ),
    ]
)
class UserSerializer(GenericSerializer):
    """
    User Serializer
    """

    # Make groups return the group name instead of the group id
    groups = serializers.StringRelatedField(many=True, read_only=True)
    # Make string related field return code name instead of id
    user_permissions = serializers.SlugRelatedField(
        many=True, read_only=True, slug_field="codename"
    )
    profile = UserProfileSerializer(required=False)

    class Meta:
        """
        Metaclass for UserSerializer
        """

        model = models.User
        extra_fields = ("profile", "organization")
        extra_read_only_fields = (
            "id",
            "online",
            "last_login",
            "groups",
            "user_permissions",
            "is_staff",
            "is_active",
            "is_superuser",
        )
        extra_kwargs = {
            "password": {"write_only": True, "required": False},
            "date_joined": {"read_only": True},
        }

    def create(self, validated_data: Any) -> models.User:
        """Create a user

        Args:
            validated_data (Any): Validated data

        Returns:
            models.User: User instance
        """

        # Get the organization of the user from the request.
        organization = super().get_organization
        validated_data["organization"] = organization

        # Popped data (profile)
        profile_data = validated_data.pop("profile", {})
        profile_data["organization"] = organization

        # Create the user
        user = models.User.objects.create_user(
            username=validated_data["username"],
            email=validated_data["email"],
            password=validated_data["password"],
            organization=organization,
        )

        # Create the user profile
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

        if profile_data := validated_data.pop("profile", None):
            instance.profile.update_profile(**profile_data)

        if validated_data.pop("password", None):
            raise serializers.ValidationError(
                "Password cannot be changed using this endpoint. Please use the change password endpoint."
            )

        instance.update_user(**validated_data)

        return instance


@extend_schema_serializer(
    examples=[
        OpenApiExample(
            "Change User Password Response",
            summary="Change User Password Response",
            value={"Password updated successfully."},
            response_only=True,
            status_codes=["200"],
        ),
    ]
)
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
            raise serializers.ValidationError(
                "Old password is incorrect. Please try again."
            )
        return value

    def validate(self, attrs: Any) -> Any:
        """Validate the new password

        Args:
            attrs (Any): Data to validate

        Returns:
            dict[str, Any]: Validated data
        """

        if attrs["new_password"] != attrs["confirm_password"]:
            raise serializers.ValidationError(
                "Passwords do not match. Please try again."
            )
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
        user: models.User = self.context["request"].user
        user.set_password(password)
        user.save()
        return user


class ResetPasswordSerializer(serializers.Serializer):
    """
    Reset Password Serializer
    """

    email = serializers.EmailField(required=True)

    def validate_email(self, value: str) -> str:
        """Validate the email

        Args:
            value (str): Email

        Returns:
            str: Validated email
        """

        try:
            user = models.User.objects.get(email=value)
        except models.User.DoesNotExist as e:
            raise serializers.ValidationError(
                "No user found with the given email exists. Please try again."
            ) from e

        if not user.is_active:
            raise serializers.ValidationError(
                "This user is not active. Please contact support for assistance."
            )
        return value

    def save(self, **kwargs: Any) -> models.User:
        """Save the new password

        Args:
            **kwargs (Any): Keyword arguments

        Returns:
            models.User: User instance
        """

        user = models.User.objects.get(email=self.validated_data["email"])
        new_password = models.User.objects.make_random_password()
        user.set_password(new_password)
        user.save()

        send_mail(
            "Your password has been reset",
            f"Your new password is {new_password}. Please change it as soon as you log in.",
            "noreply@monta.io",
            [user.email],
            fail_silently=False,
        )
        return user


class UpdateEmailSerializer(serializers.Serializer):
    """
    Email Change Serializer
    """

    email = serializers.EmailField(required=True)
    current_password = serializers.CharField(required=True)

    def validate(self, attrs: Any) -> Any:
        """Validate the token and new email

        Args:
            attrs (Any): Attributes

        Returns:
            dict[str, Any]: Validated attributes
        """
        current_password = attrs.get("current_password")
        email = attrs.get("email")

        user = self.context["request"].user

        if not user.check_password(current_password):
            raise serializers.ValidationError(
                {"current_password": "Current password is incorrect. Please try again."}
            )

        if user.email == email:
            raise serializers.ValidationError(
                {"email": "New email cannot be the same as the current email."}
            )

        if models.User.objects.filter(email=email).exists():
            raise serializers.ValidationError(
                {"email": "A user with the given email already exists."}
            )

        return attrs

    def save(self, **kwargs: Any) -> models.User:
        """Save the new email

        Args:
            **kwargs (Any): Keyword arguments

        Returns:
            models.User: User instance
        """

        user = self.context["request"].user
        user.email = self.validated_data["email"]
        user.save()

        return user


class VerifyTokenSerializer(serializers.Serializer):
    """A serializer for token verification.

    The serializer provides a token field. The token field is used to verify the incoming token
    from the user. If the given token is valid then the user is given back the token and the user
    id in the response. Otherwise, the user is given an error message.

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
            return attrs
        else:
            raise serializers.ValidationError(
                "Unable to validate given token. Please try again.",
                code="authentication",
            )


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


@extend_schema_serializer(
    examples=[
        OpenApiExample(
            "Token Provision Request",
            summary="Token Provision Request",
            value={
                "username": "test",
                "password": "test",
            },
            request_only=True,
        ),
        OpenApiExample(
            "Token Provision Response",
            summary="Token Provision Response",
            value={
                "user_id": "b08e6e3f-28da-47cf-ad48-99fc7919c087",
                "api_token": "756ab1e4e0d23ff3a7eda30e09ffda65cae2d623",
            },
            response_only=True,
        ),
    ]
)
class TokenProvisionSerializer(serializers.Serializer):
    """
    Token Provision Serializer
    """

    username = serializers.CharField()
    password = serializers.CharField(
        style={"input_type": "password"},
        trim_whitespace=False,
    )

    def validate(self, attrs: Any) -> Any:
        """Validate the data
        Args:
            attrs (Any): Data to validate
        Returns:
            Any
        """
        username = attrs.get("username")
        password = attrs.get("password")

        user = authenticate(username=username, password=password)

        if not user:
            raise serializers.ValidationError(
                "User with the given credentials does not exist. Please try again.",
                code="authorization",
            )
        attrs["user"] = user
        return attrs
