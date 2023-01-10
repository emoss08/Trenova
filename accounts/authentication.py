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

from typing import Tuple, Type

from django.utils import timezone
from django.utils.translation import gettext_lazy as _
from rest_framework import HTTP_HEADER_ENCODING, authentication, exceptions
from rest_framework.request import Request

from accounts import models


def get_authorization_header(request: Request) -> bytes:
    """
    Return request's 'Authorization:' header, as a bytestring.

    Hide some test client ickyness where the header can be unicode.
    """

    auth = request.META.get("HTTP_AUTHORIZATION", b"")
    if isinstance(auth, str):
        # Work around django test client oddness
        auth = auth.encode(HTTP_HEADER_ENCODING)
    return auth


class TokenAuthentication(authentication.TokenAuthentication):
    """
    Authentication backend for the token authentication system.
    """

    model: Type[models.Token] = models.Token

    def authenticate(self, request: Request) -> Tuple[models.User, models.Token] | None:
        """

        Args:
            request ():

        Returns:

        """

        auth: list[bytes] = get_authorization_header(request).split()

        if not auth or auth[0].lower() != self.keyword.lower().encode():
            return None

        if len(auth) == 1:
            msg = _("Invalid token header. No credentials provided.")
            raise exceptions.AuthenticationFailed(msg)  # type: ignore
        elif len(auth) > 2:
            msg = _("Invalid token header. Token string should not contain spaces.")
            raise exceptions.AuthenticationFailed(msg)  # type: ignore

        try:
            token = auth[1].decode()
        except UnicodeError:
            msg = _(
                "Invalid token header. Token string should not contain invalid characters."
            )
            raise exceptions.AuthenticationFailed(msg)  # type: ignore

        return self.authenticate_credentials(token)

    def authenticate_credentials(self, key: str) -> Tuple[models.User, models.Token]:
        """Authenticate the token

        Authenticate the given credentials. If authentication is successful,
        return a two-tuple of (user, token).

        Args:
            key (str): Token key

        Returns:
            tuple: User and token
        """

        try:
            token = (
                self.model.objects.select_related("user", "user__organization")
                .only(
                    "user__id",
                    "user__organization",
                    "key",
                    "expires",
                    "id",
                    "last_used",
                )
                .get(key=key)
            )
        except self.model.DoesNotExist:
            raise exceptions.AuthenticationFailed("Invalid token.")

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

        user = token.user

        if not user.is_active:
            raise exceptions.AuthenticationFailed(
                "User inactive or deleted. Please Try Again."
            )

        return user, token
