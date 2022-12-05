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

    model = models.Token

    def authenticate(self, request: Request) -> None | tuple:
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
            raise exceptions.AuthenticationFailed(msg)
        elif len(auth) > 2:
            msg = _("Invalid token header. Token string should not contain spaces.")
            raise exceptions.AuthenticationFailed(msg)

        try:
            token = auth[1].decode()
        except UnicodeError:
            msg = _(
                "Invalid token header. Token string should not contain invalid characters."
            )
            raise exceptions.AuthenticationFailed(msg)

        return self.authenticate_credentials(token)

    def authenticate_credentials(self, key: str) -> tuple:
        """Authenticate the token

        Authenticate the given credentials. If authentication is successful,
        return a two-tuple of (user, token).

        Args:
            key (str): Token key

        Returns:
            tuple: User and token
        """

        model: type[models.Token] = self.get_model()

        try:
            token = model.objects.prefetch_related("user").get(key=key)
        except model.DoesNotExist:
            raise exceptions.AuthenticationFailed("Invalid token")

        if (
            not token.last_used
            or (timezone.now() - token.last_used).total_seconds() > 60
        ):
            models.Token.objects.filter(pk=token.pk).update(last_used=timezone.now())

        if token.is_expired:
            raise exceptions.AuthenticationFailed("Token has expired")

        user = token.user

        if not user.is_active:
            raise exceptions.AuthenticationFailed("User inactive or deleted")

        return user, token
