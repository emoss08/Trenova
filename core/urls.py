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

from django.urls import path
from rest_framework import routers

from accounts import views as user_views

router = routers.SimpleRouter()

router.register(r'users', user_views.UserViewSet)
router.register(r'profiles', user_views.UserProfileViewSet)

urlpatterns = [
    path("token/", user_views.TokenObtainView.as_view(), name="token_obtain"),
    path("token/verify/", user_views.TokenVerifyView.as_view(), name="token_verify"),
]

urlpatterns += router.urls
