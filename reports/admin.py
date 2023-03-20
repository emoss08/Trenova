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

import json
from typing import Any
import requests
from django.contrib import admin
from django.http import HttpRequest, HttpResponse

from accounts.selectors import get_user_auth_token_from_request
from reports import models, forms
from utils.admin import GenericAdmin, GenericTabularInline


class ReportColumnAdmin(GenericTabularInline[models.ReportColumn, models.CustomReport]):
    """
    Admin class for managing ReportColumn objects as a tabular inline in the CustomReport admin interface.

    Args:
        GenericTabularInline: A generic inline class for tabular inlines.
        models.ReportColumn: The ReportColumn model for which to create the admin interface.
        models.CustomReport: The CustomReport model that the ReportColumn objects belong to.

    Attributes:
        model (models.ReportColumn): The ReportColumn model to be used.
        form (forms.ReportColumnForm): The form used for creating or updating ReportColumn objects.
        extra (int): The number of extra forms to display in the inline.
        fieldsets (tuple): A tuple containing a single fieldset with the fields to display for the inline.
    """

    model = models.ReportColumn
    form = forms.ReportColumnForm
    extra = 1
    fieldsets = (
        (
            None,
            {
                "fields": (
                    "report",
                    "column_name",
                    "column_order",
                )
            },
        ),
    )


@admin.register(models.CustomReport)
class CustomReportAdmin(GenericAdmin[models.CustomReport]):
    """
    Admin class for managing CustomReport objects in the Django admin interface.

    Args:
        GenericAdmin: A generic admin class for managing Django models.
        models.CustomReport: The CustomReport model for which to create the admin interface.

    Attributes:
        list_display (tuple): A tuple containing the fields to display in the list view for CustomReport objects.
        list_filter (tuple): A tuple containing the fields to use as filters in the list view for CustomReport objects.
        search_fields (tuple): A tuple containing the fields to use for searching CustomReport objects.
        change_form_template (str): The name of the template to use for the change form view for CustomReport objects.
        inlines (tuple): A tuple containing the inline classes to use in the change form view for CustomReport objects.
        ordering (tuple): A tuple containing the fields to use for ordering CustomReport objects in the list view.
        fieldsets (tuple): A tuple containing a single fieldset with the fields to display in the change form view.

    Methods:
        save_formset(request, form, formset, change) -> None:
            Custom implementation of the save_formset method to validate ReportColumn objects before saving.
        render_change_form(request, context, add=False, change=None, form_url=None, obj=None) -> HttpResponse:
            Custom implementation of the render_change_form method to add an authentication token to the context.
    """

    list_display = ("name", "table", "organization")
    list_filter = ("organization",)
    search_fields = ("name", "table")
    change_form_template = "admin/reports/customreport_change_form.html"
    inlines = (ReportColumnAdmin,)
    ordering = ("name",)
    fieldsets = (
        (
            None,
            {
                "fields": (
                    "name",
                    "table",
                )
            },
        ),
    )

    def save_formset(
        self, request: HttpRequest, form: Any, formset: Any, change: Any
    ) -> None:
        """This method overrides the default behavior of Django's save_formset method to validate ReportColumn objects before saving.

        Args:
            request (HttpRequest): The HTTP request object containing the formset data.
            form (Any): The form used for creating or updating CustomReport objects.
            formset (Any): The formset used for creating or updating ReportColumn objects.
            change (Any): A flag indicating whether this is an update or a create operation.

        Returns:
            None: This method does not return anything.
        """

        if formset.model == models.ReportColumn:
            for form in formset.forms:
                if not form.cleaned_data.get("DELETE"):
                    column_name = form.cleaned_data.get("column_name")
                    table_name = form.cleaned_data["report"].table

                    if table_name and column_name:
                        # Replace the following URL with the appropriate URL for your API endpoint
                        api_url = f"http://localhost:8000/api/table_columns/?table_name={table_name}"
                        response = requests.get(api_url)
                        data = json.loads(response.text)
                        columns = [col["name"] for col in data["columns"]]

                        if column_name not in columns:
                            form.add_error(
                                "column_name",
                                f"Select a valid choice. {column_name} is not one of the available choices.",
                            )
                        else:
                            form.cleaned_data["column_name"] = column_name
                            form.instance.column_name = column_name
        super().save_formset(request, form, formset, change)

    def render_change_form(
        self,
        request: HttpRequest,
        context: dict[str, Any],
        add: bool = False,
        change: bool = False,
        form_url: str = "",
        obj: models.CustomReport | None = None,
    ) -> HttpResponse:
        """
        Custom implementation of the render_change_form method to add an authentication token to the context.

        Args:
            request (HttpRequest): The HTTP request object for the change form view.
            context (dict[str, Any]): The context dictionary for the change form view.
            add (bool, optional): A flag indicating whether this is an add or update operation. Defaults to False.
            change (bool, optional): A flag indicating whether this is a change operation. Defaults to None.
            form_url (str, optional): The URL for the form. Defaults to None.
            obj (models.CustomReport, optional): The CustomReport object being edited. Defaults to None.

        Returns:
            HttpResponse: The HTTP response object for the change form view.

        """

        context["auth_token"] = get_user_auth_token_from_request(request=request)
        return super().render_change_form(request, context, add, change, form_url, obj)


@admin.register(models.ScheduledReport)
class ScheduledReportAdmin(GenericAdmin[models.ScheduledReport]):
    """
    Admin class for managing ScheduledReport objects in the Django admin interface.

    Args:
        GenericAdmin: A generic admin class for managing Django models.
        models.ScheduledReport: The ScheduledReport model for which to create the admin interface.

    Attributes:
        list_display (tuple): A tuple containing the fields to display in the list view for ScheduledReport objects.
        search_fields (tuple): A tuple containing the fields to use for searching ScheduledReport objects.
        list_filter (tuple): A tuple containing the fields to use as filters in the list view for ScheduledReport objects.
        ordering (tuple): A tuple containing the fields to use for ordering ScheduledReport objects in the list view.
    """

    list_display = ("report", "organization")
    search_fields = ("report",)
    list_filter = ("organization",)
    ordering = ("schedule_type",)
