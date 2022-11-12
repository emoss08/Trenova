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

from typing import Optional, Type

from django.contrib import admin
from django.db.models import Model, QuerySet
from django.forms import BaseModelForm
from django.http import HttpRequest

from accounts.models import User, UserProfile
from organization.models import Organization


class GenericModel(Model):
    """
    Generic Model
    """

    organization: Organization

    def __str__(self) -> str:
        return self.organization.name


class AuthHttpRequest(HttpRequest):
    """
    Authenticated HTTP Request
    """

    user: User

    @property
    def profile(self) -> Optional[UserProfile]:
        """Get User Profile

        Returns:
            Optional[UserProfile]: User Profile
        """
        return self.user.profile if hasattr(self.user, "profile") else None


class GenericAdmin(admin.ModelAdmin):
    """
    Generic Admin Class for all models
    """

    exclude: tuple[str, ...] = ("organization",)

    def get_queryset(self, request: HttpRequest) -> QuerySet[Model]:
        """Get Queryset

        Args:
            request (HttpRequest): Request Object

        Returns:
            QuerySet[Model]: Queryset of Model
        """
        return (
            super()
            .get_queryset(request)
            .filter(organization=request.user.organization)  # type: ignore
        )

    def save_model(
            self,
            request: AuthHttpRequest,
            obj: GenericModel,
            form: Type[BaseModelForm],
            change: bool,
    ) -> None:
        """Save Model

        Args:
            request (AuthHttpRequest): Request Object
            obj (GenericModel): Model Object
            form (Type[BaseModelForm]): Form Class
            change (bool): If the model is being changed

        Returns:
            None
        """
        obj.organization = request.user.organization
        super().save_model(request, obj, form, change)

    def save_formset(self, request, form, formset, change) -> None:
        """Save Formset

        Args:
            request (Any): Request Object
            form (Any): Form Object
            formset (Any): Formset Object
            change (Any): If the model is being changed

        Returns:
            None
        """
        instances = formset.save(commit=False)
        for instance in instances:
            instance.organization = request.user.organization
            instance.save()
        formset.save_m2m()
        super().save_formset(request, form, formset, change)

    def get_autocomplete_fields(self, request: HttpRequest) -> list[str]:
        """Get Autocomplete Fields

        Args:
            request (HttpRequest): Request Object

        Returns:
            tuple[str, ...]: Autocomplete Fieldss
        """
        autocomplete_fields = []
        for field in self.model._meta.get_fields():
            if field.is_relation and field.many_to_one:
                autocomplete_fields.append(field.name)
        return autocomplete_fields


class GenericStackedInline(admin.StackedInline):
    """
    Generic Admin Stacked for all Models with Organization Exclusion
    """

    extra = 0
    exclude: tuple[str, ...] = ("organization",)

    def get_queryset(self, request: HttpRequest) -> QuerySet[Model]:
        """Get Queryset

        Args:
            request (HttpRequest): Request Object

        Returns:
            QuerySet[Model]: Queryset of Model
        """
        return (
            super()
            .get_queryset(request)
            .filter(organization=request.user.organization)  # type: ignore
        )

    def get_autocomplete_fields(self, request: HttpRequest):
        """Get Autocomplete Fields

        Args:
            request (HttpRequest): Request Object

        Returns:
            tuple[str, ...]: Autocomplete Fields
        """
        autocomplete_fields = []
        for field in self.model._meta.get_fields():
            if field.is_relation and field.many_to_one:
                autocomplete_fields.append(field.name)
        return autocomplete_fields


class GenericTabularInline(admin.TabularInline):
    """
    Generic Admin Tabular Inline with Organizaiton Exclusion
    """

    extra = 0
    exclude: tuple[str, ...]
