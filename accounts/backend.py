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
