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
from typing import Any

from django import forms
from reports import models


class CustomReportForm(forms.ModelForm):
    """
    A ModelForm used for creating or updating CustomReport objects.

    Attributes:
        table_name (forms.ChoiceField): A ChoiceField used for selecting a table name.

    Meta:
        model (models.CustomReport): The CustomReport model used for creating or updating objects.
        fields (str): The fields to include in the form. "__all__" is used to include all fields.
    """

    table_name = forms.ChoiceField(
        choices=models.TABLE_NAME_CHOICES,
    )

    class Meta:
        model = models.CustomReport
        fields = "__all__"


class IgnorePKModelChoiceField(forms.ModelChoiceField):
    """
    A ModelChoiceField subclass that bypasses primary key validation.

    Methods:
        clean(value) -> Any:
            Cleans the form data.
    """

    def clean(self, value: Any) -> Any:
        """
        Cleans the form data.

        Args:
            value (Any): The data to clean.

        Returns:
            Any: The cleaned data.
        """
        return value


class ReportColumnForm(forms.ModelForm):
    """
    A ModelForm used for creating or updating ReportColumn objects.

    Attributes:
        column_name (IgnorePKModelChoiceField): A ModelChoiceField used for selecting a column name.

    Meta:
        model (models.ReportColumn): The ReportColumn model used for creating or updating objects.
        fields (tuple): The fields to include in the form.
    """

    column_name = IgnorePKModelChoiceField(
        queryset=models.ReportColumn.objects.none(), required=False
    )

    class Meta:
        model = models.ReportColumn
        fields = ("custom_report", "column_name", "column_order")
