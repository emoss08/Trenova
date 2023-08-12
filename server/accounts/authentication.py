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


from accounts import models
from django.utils import timezone
from rest_framework import HTTP_HEADER_ENCODING, authentication, exceptions
from rest_framework.request import Request


def get_authorization_header(request: Request) -> bytes:
    auth = request.META.get("HTTP_AUTHORIZATION", b"")
    if isinstance(auth, str):
        auth: bytes = auth.encode(HTTP_HEADER_ENCODING)  # type: ignore
    return auth


class BearerTokenAuthentication(authentication.BaseAuthentication):
    keyword = "Bearer"
    model = models.Token

    def authenticate(self, request: Request) -> tuple[models.User, models.Token] | None:
        auth: list[bytes] = get_authorization_header(request).split()

        if auth and auth[0].lower() == self.keyword.lower().encode():
            if len(auth) == 1:
                raise exceptions.AuthenticationFailed(
                    "Invalid token header. No credentials provided. Please try again."
                )
            elif len(auth) > 2:
                raise exceptions.AuthenticationFailed(
                    "Invalid token header. Token string should not contain spaces. Please try again."
                )

            try:
                token = auth[1].decode()
            except UnicodeError as e:
                raise exceptions.AuthenticationFailed(
                    "Invalid token header. Token string should not contain invalid characters."
                ) from e
        else:
            token = request.COOKIES.get("auth_token")  # type: ignore

            if not token:
                return None

        return self.authenticate_credentials(key=token)

    def authenticate_credentials(self, *, key: str) -> tuple[models.User, models.Token]:
        try:
            token = (
                self.model.objects.select_related("user")
                .only(
                    "user_id",
                    "user__is_active",
                    "user__organization_id",
                    "key",
                    "expires",
                    "last_used",
                )
                .get(key=key)
            )
        except self.model.DoesNotExist as e:
            raise exceptions.AuthenticationFailed("Invalid token.") from e

        self.validate_token(token=token)

        return token.user, token

    @staticmethod
    def validate_token(*, token: models.Token) -> None:
        if (
            not token.last_used
            or (timezone.now() - token.last_used).total_seconds() > 60
        ):
            token.last_used = timezone.now()
            token.save(update_fields=["last_used"])

        if token.is_expired and token.expires:
            raise exceptions.AuthenticationFailed(
                f"Token expired at {token.expires.strftime('%Y-%m-%d %H:%M:%S')}. Please login again."
            )

        if not token.user.is_active:
            raise exceptions.AuthenticationFailed(
                "User inactive or deleted. Please Try Again."
            )
