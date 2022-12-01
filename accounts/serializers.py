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

from accounts.models import User
from djoser.serializers import UserCreateSerializer, UserSerializer


class UserCreateSerializer(UserCreateSerializer):
    """
    Serializer for creating a new user
    """

    class Meta:
        """
        Metaclass for UserCreateSerializer
        """

        model = User
        fields = ("id", "username", "email", "password", "organization")


class UserSerializer(UserSerializer):
    """
    Serializer for user
    """

    class Meta:
        """
        Metaclass for UserSerializer
        """

        model = User
        fields = ("id", "username", "email", "organization")
