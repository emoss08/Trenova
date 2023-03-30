# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2023 MONTA                                                                         -
#                                                                                                  -
#  This file is part of Monta.                                                                     -
#                                                                                                  -
#  The Monta software is licensed under the Business Source License 1.1. You are granted the right -
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

from typing import Any, Union, Type, Tuple

from django.contrib import admin
from django.forms import ModelForm
from django.http import HttpRequest

from utils.admin import GenericAdmin

from .models import DivisionCode, GeneralLedgerAccount, RevenueCode


@admin.register(GeneralLedgerAccount)
class GeneralLedgerAccountAdmin(GenericAdmin[GeneralLedgerAccount]):
    """
    General Ledger Account Admin
    """

    model: Type[GeneralLedgerAccount] = GeneralLedgerAccount
    list_display: Tuple[str, ...] = (
        "id",
        "account_number",
        "description",
    )
    search_fields: Tuple[str, ...] = (
        "id",
        "description",
    )

    def get_form(
        self,
        request: HttpRequest,
        obj: Union[GeneralLedgerAccount, None] = None,
        change: bool = False,
        **kwargs: Any,
    ) -> Type[ModelForm[GeneralLedgerAccount]]:
        """Get Form for Model

        Args:
            change (bool): If the model is being changed
            request (HttpRequest): Request Object
            obj (Optional[GeneralLedgerAccount]): General Ledger Account Object
            **kwargs (Any): Keyword Arguments

        Returns:
            Type[ModelForm[Any]]: Form Class's
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
    list_display: Tuple[str, ...] = (
        "code",
        "description",
    )
    search_fields: Tuple[str, ...] = (
        "code",
        "description",
    )


@admin.register(DivisionCode)
class DivisionCodeAdmin(GenericAdmin[DivisionCode]):
    """
    Division Code Admin
    """

    model: Type[DivisionCode] = DivisionCode
    list_display: Tuple[str, ...] = (
        "code",
        "description",
    )
    search_fields: Tuple[str, ...] = (
        "code",
        "description",
    )
