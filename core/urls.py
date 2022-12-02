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
from rest_framework_simplejwt.views import (
    TokenObtainSlidingView,
    TokenRefreshSlidingView,
)

from accounts.views import GenericTokenVerifyView

urlpatterns = [
    path('token/', TokenObtainSlidingView.as_view(), name='token_obtain'),
    path('token/refresh/', TokenRefreshSlidingView.as_view(), name='token_refresh'),
    path('token/verify/', GenericTokenVerifyView.as_view(), name='token_verify'),
]

# urlpatterns = [
#     path("auth/", include("djoser.urls")),
#     path("auth/", include("djoser.urls.jwt")),
# ]
