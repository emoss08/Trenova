# -*- coding: utf-8 -*-
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

from typing import Optional

from .models import User


def create_user(username: str, email_address: str, pass_word: str) -> User:
    """
    create_user _summary_

    Args:
        username (str): username for the user
        email_address (str): email address for the user
        pass_word (str): password for the user

    Returns:
        User: Returns a user object
    """
    return User.objects.create_user(
        user_name=username, email=email_address, password=pass_word
    )


def find_user_by_username(user_name: str) -> Optional[User]:
    """
    filter_user_by_username _summary_

    Args:
        user_name (str): username for the user

    Returns:
        Optional[User]: Returns a user object
    """
    try:
        return User.objects.filter(username=user_name).first()
    except User.DoesNotExist:
        return None
