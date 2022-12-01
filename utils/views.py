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

from braces import views
from django.contrib.auth import mixins
from django.views import generic


class GenericTemplateView(
    mixins.LoginRequiredMixin, views.PermissionRequiredMixin, generic.TemplateView
):
    """
    Generic Template view with LoginRequiredMixin and PermissionRequiredMixin
    """
    pass


class GenericListView(
    mixins.LoginRequiredMixin, views.PermissionRequiredMixin, generic.ListView
):
    """
    Generic List view with LoginRequiredMixin and PermissionRequiredMixin
    """
    pass


class GenericView(
    mixins.LoginRequiredMixin, views.PermissionRequiredMixin, generic.View
):
    """
    Generic View with LoginRequiredMixin and PermissionRequiredMixin
    """
    pass


class GenericCreateView(
    mixins.LoginRequiredMixin, views.PermissionRequiredMixin, generic.CreateView
):
    """
    Generic Create view with LoginRequiredMixin and PermissionRequiredMixin
    """
    pass


class GenericUpdateView(
    mixins.LoginRequiredMixin, views.PermissionRequiredMixin, generic.UpdateView
):
    """
    Generic Update view with LoginRequiredMixin and PermissionRequiredMixin
    """
    pass


class GenericDeleteView(
    mixins.LoginRequiredMixin, views.PermissionRequiredMixin, generic.DeleteView
):
    """
    Generic Delete view with LoginRequiredMixin and PermissionRequiredMixin
    """
    pass
