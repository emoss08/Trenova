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

from typing import Any

from django.db.models import QuerySet
from django.http import HttpRequest, JsonResponse
from rest_framework import viewsets
from braces import views
from django.contrib.auth import mixins
from django.core.exceptions import ImproperlyConfigured
from django.utils.decorators import method_decorator
from django.views import generic
from django.views.decorators.http import require_safe
from django.views.decorators.vary import vary_on_cookie


class OrganizationViewSet(viewsets.ModelViewSet):
    """
    Organization ViewSet to manage requests to the organization endpoint
    """

    def get_queryset(self) -> QuerySet:
        """Filter the queryset to only include the current user's organization

        Returns:

        """

        return self.queryset.filter(organization=self.request.user.organization.id).select_related(  # type: ignore
            "organization",
        )


@method_decorator(require_safe, name="dispatch")
@method_decorator(vary_on_cookie, name="dispatch")
class GenericTemplateView(
    mixins.LoginRequiredMixin,
    views.PermissionRequiredMixin,
    generic.TemplateView
):
    model_lookup = None
    permission_required = None

    def __init__(self, *args, **kwargs):
        super().__init__(*args, **kwargs)
        self.permission_required = self._get_permission_required()

    def _get_permission_required(self) -> str:
        if not self.model_lookup:
            raise ImproperlyConfigured(
                f"{self.__class__.__name__} is missing a model_lookup attribute."
            )

        app_label = self.model_lookup._meta.app_label
        model_name = self.model_lookup._meta.model_name
        return f"{app_label}.{model_name}.view_{model_name}"

    def get_context_data(self, **kwargs) -> dict[str, Any]:
        context = self.context_data or {}
        context.update(kwargs)
        return super().get_context_data(**context)


class GenericCreateView(
    mixins.LoginRequiredMixin, views.PermissionRequiredMixin, generic.CreateView
):
    """
    Generic Create View to be used as a base for all views that require login and permissions
    """

    append_organization: bool = True

    def post(self, request: HttpRequest, *args, **kwargs: Any) -> JsonResponse:
        """Handle post requests

        Args:
            request (HttpRequest): Request object
            *args: Arguments
            **kwargs: Keyword arguments

        Returns:
            JsonResponse: Response
        """

        form = self.form_class(request.POST)

        if form.is_valid():
            if self.append_organization:
                form.save(commit=False)
                form.instance.organization = self.request.user.organization
            form.save()

            return JsonResponse(
                {"result": "success", "message": "New Record Created Successfully!"},
                status=201,
            )

        return JsonResponse(
            {
                "result": "error",
                "errors": form.errors,
            },
            status=400,
        )
