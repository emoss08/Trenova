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


from django.contrib.auth.backends import ModelBackend

from .models import User

UserModel: type[User] = User


class UserBackend(ModelBackend):
    """User Authentication backend

    This class is used to authenticate users using their user id.
    Returns the user object if the user is authenticated.
    Along with related profile, title and organization objects.
    """

    def get_user(self, user_id: int) -> User | None:
        """Get the user object.

        Args:
            user_id (int): parameter to get the primary key of the user.

        Returns:
            User | None: Returns a user object if the user is authenticated.
        """
        try:
            user = (
                UserModel._default_manager.only("id")
                .select_related("profile", "organization")
                .get(pk__exact=user_id)
            )
        except UserModel.DoesNotExist:
            return None

        return user if self.user_can_authenticate(user) else None
