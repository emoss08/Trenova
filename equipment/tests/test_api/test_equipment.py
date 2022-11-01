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

from django.test import TestCase
from rest_framework.test import APIClient

from accounts.tests.factories.user import Userfactory

user = Userfactory()
client = APIClient()


class TestEquipment(TestCase):
    def setUp(self) -> None:
        client = APIClient()
        user = Userfactory()

        client.force_authenticate(user=user)

    def test_user_logged_in(self) -> None:
        print(client.force_authenticate(user=user))
