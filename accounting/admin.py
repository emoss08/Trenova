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

from typing import Any, Optional, Type

from django.contrib import admin
from django.forms import ModelForm

from core.generics.admin import GenericAdmin

from .models import GeneralLedgerAccount, RevenueCode


@admin.register(GeneralLedgerAccount)
class GeneralLedgerAccountAdmin(GenericAdmin[GeneralLedgerAccount]):
    """
    General Ledger Account Admin
    """

    model: Type[GeneralLedgerAccount] = GeneralLedgerAccount
    list_display: tuple[str, ...] = (
        "id",
        "description",
    )
    search_fields: tuple[str, ...] = (
        "id",
        "description",
    )

    def get_form(
        self, request, obj: Optional[GeneralLedgerAccount] = None, **kwargs: Any
    ) -> Type[ModelForm[Any]]:
        """Get Form for Model

        Args:
            request (HttpRequest): Request Object
            obj (Optional[GeneralLedgerAccount]): General Ledger Account Object
            **kwargs (Any): Keyword Arguments

        Returns:
            Type[ModelForm[Any]]: Form Class
        """
        form = super().get_form(request, obj, **kwargs)
        form.base_fields["account_number"].widget.attrs[
            "placeholder"
        ] = "0000-0000-0000-0000"
        form.base_fields["account_number"].widget.attrs["value"] = "0000-0000-0000-0000"
        return form


@admin.register(RevenueCode)
class RevenueCodeAdmin(GenericAdmin[RevenueCode]):
    """
    Revenue Code Admin
    """

    model: Type[RevenueCode] = RevenueCode
    list_display: tuple[str, ...] = (
        "id",
        "code",
        "description",
    )
    search_fields: tuple[str, ...] = (
        "id",
        "code",
        "description",
    )
