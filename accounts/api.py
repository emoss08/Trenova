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
from datetime import timedelta
from typing import Any

from django.contrib.auth import login, logout
from django.contrib.auth.models import Group, Permission
from django.db.models import Prefetch, QuerySet
from django.middleware import csrf
from django.utils import timezone
from rest_framework import generics, permissions, response, status, views, viewsets
from rest_framework.authtoken.views import ObtainAuthToken
from rest_framework.request import Request

from accounts import models, serializers
from accounts.permissions import ViewAllUsersPermission
from utils.exceptions import InvalidTokenException


class GroupViewSet(viewsets.ModelViewSet):
    """
    Group ViewSet to manage requests to the group endpoint
    """

    serializer_class = serializers.GroupSerializer
    queryset = Group.objects.all()
    filterset_fields = ["name"]
    ordering_fields = "__all__"
    permission_classes = [permissions.IsAuthenticated]


class PermissionViewSet(viewsets.ModelViewSet):
    """
    Permission ViewSet to manage requests to the permission endpoint
    """

    serializer_class = serializers.PermissionSerializer
    queryset = Permission.objects.all()
    filterset_fields = ["name"]
    ordering_fields = "__all__"
    permission_classes = [permissions.IsAuthenticated]


class UserViewSet(viewsets.ModelViewSet):
    """
    User ViewSet to manage requests to the user endpoint
    """

    serializer_class = serializers.UserSerializer
    queryset = models.User.objects.all()
    search_fields = (
        "username",
        "email",
        "profiles__first_name",
        "profiles__last_name",
    )
    filterset_fields = (
        "is_active",
        "department__name",
        "is_staff",
        "username",
    )
    permission_classes = [permissions.IsAuthenticated, ViewAllUsersPermission]

    def get_queryset(self) -> QuerySet[models.User]:
        """The get_queryset function is used to filter the queryset of users by organization.
        This function is called in the get_queryset method of UserViewSet, which is a subclass
        of ModelViewSet. The get_queryset method of ModelViewSet returns a QuerySet object that
        is filtered by organization and only includes certain fields from the User model.

        Args:
            self: Refer to the class itself

        Returns:
            A queryset of user objects that are filtered by the organization_id
        """
        queryset: QuerySet[models.User] = (
            self.queryset.filter(
                organization_id=self.request.user.organization_id  # type: ignore
            )
            .select_related(
                "profiles",
            )
            .prefetch_related(
                Prefetch("groups", queryset=Group.objects.only("id", "name")),
                Prefetch(
                    "user_permissions",
                    queryset=Permission.objects.only(
                        "id", "name", "content_type__app_label", "codename"
                    ),
                ),
            )
            .only(
                "last_login",
                "is_superuser",
                "id",
                "organization_id",
                "department_id",
                "is_active",
                "username",
                "email",
                "is_staff",
                "date_joined",
                "online",
                "profiles__created",
                "profiles__modified",
                "profiles__organization_id",
                "profiles__id",
                "profiles__user",
                "profiles__job_title_id",
                "profiles__first_name",
                "profiles__last_name",
                "profiles__profile_picture",
                "profiles__address_line_1",
                "profiles__address_line_2",
                "profiles__city",
                "profiles__state",
                "profiles__zip_code",
                "profiles__phone_number",
                "profiles__is_phone_verified",
            )
        )
        return queryset


class UpdatePasswordView(generics.UpdateAPIView):
    """
    An endpoint for changing password.
    """

    throttle_scope = "auth"
    serializer_class = serializers.ChangePasswordSerializer

    def update(self, request: Request, *args: Any, **kwargs: Any) -> response.Response:
        """The update function is used to update the password of a user. The function takes in
        the request and returns a response. The serializer is then called on the data from the
        request, which validates it and saves it.

        Args:
            self: Represent the instance of the class
            request: Request: Get the request data from the user
            *args: Any: Pass in a variable number of arguments
            **kwargs: Any: Pass in the keyword arguments

        Returns:
            A response object with a status of 200 and the message &quot;password updated successfully&quot;
        """
        serializer = self.get_serializer(data=request.data)
        serializer.is_valid(raise_exception=True)
        serializer.save()
        return response.Response(
            "Password updated successfully",
            status=status.HTTP_200_OK,
        )


class ResetPasswordView(views.APIView):
    """
    An endpoint for changing password.
    """

    throttle_scope = "auth"
    serializer_class = serializers.ResetPasswordSerializer
    permission_classes = [permissions.AllowAny]

    def post(self, request: Request, *args: Any, **kwargs: Any) -> response.Response:
        """The post function is used to reset the password of a user.
        The function takes in the email address of the user and sends an email with a new password.
        The new password is generated using random characters from string.ascii_letters, digits and punctuation.

        Args:
            self: Represent the instance of the class
            request(Request): Get the request object
            *args(Any): Pass a non-keyworded, variable-length argument list to the function
            **kwargs(Any): Pass in keyword arguments

        Returns:
            response.Response: A response object with a success message or an error message
        """
        serializer = self.serializer_class(data=request.data)
        if serializer.is_valid():
            serializer.save()
            return response.Response(
                {
                    "message": "Password reset successful. Please check your email for the new password."
                },
                status=status.HTTP_200_OK,
            )

        return response.Response(serializer.errors, status=status.HTTP_400_BAD_REQUEST)


class UpdateEmailView(views.APIView):
    """
    An endpoint for changing password.
    """

    throttle_scope = "auth"
    serializer_class = serializers.UpdateEmailSerializer

    def post(self, request: Request, *args: Any, **kwargs: Any) -> response.Response:
        """The post function is used to change the email of a user.
        The function takes in a request and returns either an error message or
        a success message depending on whether the serializer was valid or not.

        Args:
            self: Represent the instance of the class
            request(Request): Get the request object
            *args(Any): Catch any additional arguments that are passed to the function
            **kwargs(Any): Pass in the user id to the serializer

        Returns:
            A response object
        """
        serializer = self.serializer_class(
            data=request.data, context={"request": request}
        )
        if serializer.is_valid():
            serializer.save()
            return response.Response(
                {"message": "Email successfully changed."},
                status=status.HTTP_200_OK,
            )

        return response.Response(serializer.errors, status=status.HTTP_400_BAD_REQUEST)


class JobTitleViewSet(viewsets.ModelViewSet):
    """
    Job Title ViewSet to manage requests to the job title endpoint
    """

    serializer_class = serializers.JobTitleSerializer
    queryset = models.JobTitle.objects.all()
    filterset_fields = ["status", "name"]
    search_fields = ("name", "status")

    def get_queryset(self) -> QuerySet[models.JobTitle]:
        """The get_queryset function is used to filter the queryset of JobTitles by organization_id.
        It also allows for a query parameter, expand_users, which will prefetch related users if set to 'true'.
        This is useful when you want to get all job titles and their associated users in one request.

        Args:
            self: Represent the instance of the class

        Returns:
            QuerySet[models.JobTitle]: Filtered queryset of JobTitles
        """
        expand_users = self.request.query_params.get("expand_users", "false")

        queryset = models.JobTitle.objects.filter(
            organization_id=self.request.user.organization_id  # type: ignore
        ).only("id", "organization_id", "status", "description", "name", "job_function")

        # Prefetch related users if `expand_users` query param is 'true' or 'True'
        if expand_users.lower() == "true":
            queryset = queryset.prefetch_related(
                Prefetch(
                    "profile__user",
                    queryset=models.User.objects.only("username"),
                )
            )

        return queryset


class TokenVerifyView(views.APIView):
    """
    Rest API endpoint for users can verify a token
    """

    permission_classes = []
    http_method_names = ["post"]

    def post(self, request: Request, *args: Any, **kwargs: Any) -> response.Response:
        """The post function is used to refresh the token.

        Args:
            self: Represent the instance of the object itself
            request(Request): Get the request object
            *args(Any): Pass in a list of arguments
            **kwargs(Any): Pass in any additional arguments to the function

        Returns:
            A response object, which is a dictionary with the following keys:

        """
        token = request.COOKIES.get("auth_token")

        if request.user.is_anonymous:
            return response.Response(
                {"message": "User is not authenticated"},
                status=status.HTTP_401_UNAUTHORIZED,
            )

        token_expire_days = request.user.organization.token_expiration_days  # type: ignore

        if not token:
            raise InvalidTokenException("No token provided")

        try:
            token_obj = models.Token.objects.get(key=token)
            # Update token's expiration time
            token_obj.expires = timezone.now() + timedelta(
                days=token_expire_days
            )  # set expires to 1 day from now
            token_obj.save()

            data = {
                "user_id": token_obj.user_id,
                "organization_id": token_obj.user.organization_id,  # type: ignore
            }

            res = response.Response(data, status=status.HTTP_200_OK)

            # Set the token in the cookies again
            res.set_cookie(
                key="auth_token",
                value=token_obj.key,
                expires=token_obj.expires,
                httponly=True,
                secure=False,
                samesite="Lax",
                domain=None,
            )

            return res

        except models.Token.DoesNotExist as e:
            raise InvalidTokenException("Token is invalid") from e


class TokenProvisionView(ObtainAuthToken):
    throttle_scope = "auth"
    permission_classes = (permissions.AllowAny,)
    serializer_class = serializers.TokenProvisionSerializer

    def post(self, request: Request, *args: Any, **kwargs: Any) -> response.Response:
        """The post function is used to log in a user.
        It takes the username and password from the request body, validates them, and returns an auth token if successful.
        If unsuccessful it will return a 400 error code with an error message explaining why.

        Args:
            self: Represent the instance of the object itself
            request(Request): Get the request from the client
            *args(Any): Pass a variable number of arguments to the function
            **kwargs(Any): Pass in the keyword arguments

        Returns:
            response.Response: A response object with the status code 200
        """
        serializer = self.serializer_class(data=request.data)
        serializer.is_valid(raise_exception=True)
        user = serializer.validated_data["user"]
        token, _ = models.Token.objects.get_or_create(user=user)
        res = response.Response(status=status.HTTP_200_OK)

        if token.is_expired:
            token.delete()
            token = models.Token.objects.create(user=user)

        if user.is_active:
            login(request, user)
            user.online = True
            user.last_login = timezone.now()
            user.save()

        res.set_cookie(
            key="auth_token",
            value=token.key,
            expires=token.expires,
            httponly=True,
            secure=False,
            samesite="Lax",
            domain=None,
        )
        csrf.get_token(request)

        res.data = {
            "user_id": user.id,
            "organization_id": user.organization_id,
        }

        return res


class UserLogoutView(views.APIView):
    """
    Rest API endpoint for users can logout
    """

    permission_classes = [permissions.IsAuthenticated]
    http_method_names = ["post"]

    def post(self, request: Request, *args: Any, **kwargs: Any) -> response.Response:
        """The post function logs out the user and deletes their auth_token cookie.

        Args:
            self: Represent the instance of the class
            request(Request): Get the user object from the request
            *args(Any): Pass in a tuple of arguments to the function
            **kwargs(Any): Pass in any keyword arguments that you may want to use

        Returns:
            A response object with the status code 200

        """
        user = request.user

        if user.is_authenticated:
            logout(request)
            user.online = False  # type: ignore
            user.save()

        res = response.Response(status=status.HTTP_200_OK)
        res.delete_cookie("auth_token")

        return res
