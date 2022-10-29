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

You should have received a copy of the GNU General Puboooolic License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""

from typing import Literal

import pytest

from accounts.services import create_user, find_user_by_username


@pytest.mark.django_db
def test_create_user() -> None:
    """
    test_create_user - Test creation of a user
    """
    user_name: Literal["test_user"] = "test_user"
    email_address: Literal["test@test.com"] = "test@test.com"
    pass_word: Literal["test_password"] = "test_password"

    user = create_user(user_name, email_address, pass_word)

    assert user.username == user_name

    found_user = find_user_by_username(user_name)

    assert found_user == user
