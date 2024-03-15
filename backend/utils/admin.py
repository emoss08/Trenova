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
from collections.abc import Sequence
from typing import Any

from django import forms
from django.contrib import admin
from django.core.exceptions import ImproperlyConfigured
from django.db.models import QuerySet
from django.db.models.base import Model
from django.http import HttpRequest


class GenericAdmin[_M: Model](admin.ModelAdmin[_M]):
    """
    Generic Admin Class for all models
    """

    autocomplete: bool = True

    def get_queryset(self, request: HttpRequest) -> QuerySet[_M]:
        """Get Queryset for Model

        Args:
            request (HttpRequest): Request Object

        Returns:
            QuerySet[Model]: Queryset of Model
        """
        return (
            super()
            .get_queryset(request)
            .select_related(*self.get_autocomplete_fields(request))
            .filter(organization_id=request.user.organization_id)  # type: ignore
        )

    def save_model(
        self,
        request: HttpRequest,
        obj: _M,
        form: type[forms.BaseModelForm],
        change: bool,
    ) -> None:
        """Save Model Instance

        Args:
            request (HttpRequest): Request Object
            obj (_M): Generic Model Object
            form (Type[BaseModelForm]): Form Class
            change (bool): If the model is being changed

        Returns:
            None
        """
        obj.organization = request.user.organization  # type: ignore
        obj.business_unit = request.user.organization.business_unit  # type: ignore
        super().save_model(request, obj, form, change)

    def save_formset(
        self, request: HttpRequest, form: Any, formset: Any, change: Any
    ) -> None:
        """Save Formset for Inline Models

        Args:
            request (HttpRequest): Request Object
            form (Any): Form Object
            formset (Any): Formset Object
            change (Any): If the model is being changed

        Returns:
            None
        """
        instances = formset.save(commit=False)
        for instance in instances:
            instance.organization = request.user.organization  # type: ignore
            instance.business_unit = request.user.business_unit  # type: ignore
            instance.save()
        formset.save_m2m()
        super().save_formset(request, form, formset, change)

    def get_form(
        self,
        request: HttpRequest,
        obj: _M | None = None,
        change: bool = False,
        **kwargs: Any,
    ) -> type[forms.ModelForm[_M]]:
        """Get Form for Model

        Args:
            change (bool): If the model is being changed
            request (HttpRequest): Authenticated Request Object
            obj (Optional[_M]): Model Object
            **kwargs (Any): Keyword Arguments

        Returns:
            Type[ModelForm[_M]]: Form Class
        """
        form = super().get_form(request, obj, **kwargs)
        for field in form.base_fields:
            if field == "organization":
                form.base_fields[field].initial = request.user.organization  # type: ignore
                form.base_fields[field].widget = form.base_fields[field].hidden_widget()
            elif field == "business_unit":
                form.base_fields[
                    field
                ].initial = request.user.organization.business_unit  # type: ignore
                form.base_fields[field].widget = form.base_fields[field].hidden_widget()
            form.base_fields[field].widget.attrs["placeholder"] = field.title()

        return form

    def get_autocomplete_fields(self, request: HttpRequest) -> Sequence[str]:
        """Get Autocomplete Fields

        Args:
            request (HttpRequest): Request Object

        Returns:
            Sequence[str]: Autocomplete Fields
        """
        if self.autocomplete:
            if not self.search_fields:
                raise ImproperlyConfigured(
                    f"{self.__class__.__name__} must define search_fields"
                    " when self.autocomplete is True"
                )

            return [
                field.name
                for field in self.model._meta.get_fields()
                if field.is_relation and field.many_to_one
            ]
        return []


class GenericStackedInline[_C: Model, _P: Model](admin.StackedInline[_C, _P]):
    """
    Generic Admin Stacked for all Models with Organization Exclusion
    """

    model: type[_C]
    extra = 0

    def get_queryset(self, request: HttpRequest) -> QuerySet[_C]:
        """Get Queryset
        Args:
            request (HttpRequest): Request Object
        Returns:
            QuerySet[_C]: Queryset of Model
        """
        return (
            super()
            .get_queryset(request)
            .select_related(*self.get_autocomplete_fields(request))
            .filter(organization_id=request.user.organization_id)  # type: ignore
        )

    def get_autocomplete_fields(self, request: HttpRequest) -> Sequence[str]:
        """Get Autocomplete Fields

        Returns:
            Sequence[str]: Autocomplete Fields
        """
        return [
            field.name
            for field in self.model._meta.get_fields()
            if field.is_relation and field.many_to_one
        ]


class GenericTabularInline[_C: Model, _P: Model](GenericStackedInline[_C, _P]):
    """
    Generic Admin Tabular Inline with Organization Exclusion
    """

    template = "admin/edit_inline/tabular.html"
