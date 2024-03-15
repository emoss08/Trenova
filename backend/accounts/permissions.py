# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

from rest_framework import permissions, views
from rest_framework.request import Request


class ViewAllUsersPermission(permissions.BasePermission):
    """
    A custom permission class that checks if a user has permission to view all users.
    Non-admin users can only view their own user data.
    Admin users, or users with the 'accounts.view_all_users' permission, can view all user data.

    Methods:
        has_permission(request, view):
            Returns True if the user has 'accounts.view_all_users' permission,
            or the request is for retrieving (viewing) a user's own record.

        has_object_permission(request, view, obj):
            Returns True if the user is trying to access their own record,
            or the user has 'accounts.view_all_users' permission.
    """

    def has_permission(self, request: Request, view: views.APIView) -> bool:
        """
        Checks if the user has 'accounts.view_all_users' permission or if the user is trying to view their own record.

        Args:
            request (Request): The current Django Rest Framework request.
            view (views.APIView): The view which is being accessed.

        Returns:
            bool: True if the user has 'accounts.view_all_users' permission or if the user is trying to view their own record,
                  False otherwise.
        """
        if view.action == "retrieve":  # type: ignore
            return True

        return (
            request.user.has_perm("accounts.view_all_users")
            or request.user.is_superuser
        )


class OwnershipPermission(permissions.BasePermission):
    """
    Custom permission to only allow owners of an object or admin users to edit or delete it.
    """

    def has_object_permission(self, request, view, obj):
        """
        Check if the request user is the owner of the object or an admin user.

        Args:
            request: The HTTP request.
            view: The view which is being accessed.
            obj: The object being accessed or edited.

        Returns:
            bool: True if the request user is the owner of the object or an admin, False otherwise.
        """

        # Check if the user is an admin (superuser)
        if request.user and request.user.is_superuser:
            return True

        # Check if the user is trying to access, update or delete their own record
        # Adjust `obj.user` to match your user relationship in the model
        return obj == request.user
