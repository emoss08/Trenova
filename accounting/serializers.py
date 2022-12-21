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

from rest_framework import serializers
from rest_framework.exceptions import ValidationError

from accounting import models
from utils.serializers import GenericSerializer


class GeneralLedgerAccountSerializer(GenericSerializer):
    """GeneralLedgerAccountSerializer

        A serializer class for the GeneralLedgerAccount model. This serializer is used
        to convert the GeneralLedgerAccount model instance into a Python dictionary
        format that can be rendered into a JSON response. It also defines the fields
        that should be included in the serialized representation of the model.

    Attributes:
        is_active (serializers.BooleanField): A boolean field representing the
        active status of the account. Defaults to True.

        account_type (serializers.ChoiceField): A choice field representing the
        type of the account. The choices are taken from the AccountTypeChoices
        model field.

        cash_flow_type (serializers.ChoiceField): A choice field representing the
        cash flow type of the account. The choices are taken from the
        CashFlowTypeChoices model field.

        account_sub_type (serializers.ChoiceField): A choice field representing the
        sub_type of the account. The choices are taken from the
        AccountSubTypeChoices model field.

        account_classification (serializers.ChoiceField): A choice field representing
        the classification of the account. The choices are taken from the
        AccountClassificationChoices model field.

    Metaclass Attributes:
        model (models.GeneralLedgerAccount): The GeneralLedgerAccount model that
        this serializer is associated with.

        fields (tuple of str): A tuple of field names that should be included in the
        serialized representation of the model.

        read_only_fields (tuple of str): A tuple of field names that should be
        included in the serialized representation of the model, but should be
        treated as read-only and not modifiable by the client.
    """

    is_active = serializers.BooleanField(default=True)
    account_type = serializers.ChoiceField(
        choices=models.GeneralLedgerAccount.AccountTypeChoices.choices
    )
    cash_flow_type = serializers.ChoiceField(
        choices=models.GeneralLedgerAccount.CashFlowTypeChoices.choices
    )
    account_sub_type = serializers.ChoiceField(
        choices=models.GeneralLedgerAccount.AccountSubTypeChoices.choices
    )
    account_classification = serializers.ChoiceField(
        choices=models.GeneralLedgerAccount.AccountClassificationChoices.choices
    )

    class Meta:
        """
        Metaclass for GeneralLedgerAccountSerializer
        """

        model = models.GeneralLedgerAccount
        fields = (
            "id",
            "is_active",
            "organization",
            "account_number",
            "description",
            "account_type",
            "cash_flow_type",
            "account_sub_type",
            "account_classification",
            "created",
            "modified",
        )
        read_only_fields = (
            "organization",
            "id",
            "code",
            "created",
            "modified",
        )


class RevenueCodeSerializer(serializers.ModelSerializer):
    """RevenueCodeSerializer

    A serializer class for the RevenueCode model. This serializer is used to
    convert the RevenueCode model instance into a Python dictionary format that
    can be rendered into a JSON response. It also defines the fields that should be
    included in the serialized representation of the model.

    Attributes:
        expense_account (serializers.PrimaryKeyRelatedField): A primary key related
        field representing the expense account associated with the revenue code.
        The queryset is filtered to only include accounts with an
        AccountTypeChoices value of EXPENSE.

        revenue_account (serializers.PrimaryKeyRelatedField): A primary key related
        field representing the revenue account associated with the revenue code.
        The queryset is filtered to only include accounts with an
        AccountTypeChoices value of REVENUE.

    Metaclass Attributes:
        model (models.RevenueCode): The RevenueCode model that this serializer is
        associated with.

        fields (tuple of str): A tuple of field names that should be included in the
        serialized representation of the model.
    """

    expense_account = serializers.PrimaryKeyRelatedField(
        queryset=models.GeneralLedgerAccount.objects.filter(
            account_type=models.GeneralLedgerAccount.AccountTypeChoices.EXPENSE
        )
    )
    revenue_account = serializers.PrimaryKeyRelatedField(
        queryset=models.GeneralLedgerAccount.objects.filter(
            account_type=models.GeneralLedgerAccount.AccountTypeChoices.REVENUE
        )
    )

    class Meta:
        """
        Metaclass for RevenueCodeSerializer
        """

        model = models.RevenueCode
        fields = (
            "id",
            "organization",
            "code",
            "description",
            "expense_account",
            "revenue_account",
            "created",
            "modified",
        )
