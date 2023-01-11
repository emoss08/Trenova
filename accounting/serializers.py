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

from accounting import models
from utils.serializers import GenericSerializer


class GeneralLedgerAccountSerializer(GenericSerializer):
    """A serializer class for the GeneralLedgerAccount model.

    This serializer is used to convert the GeneralLedgerAccount model instance into a Python
    dictionary format that can be rendered into a JSON response. It also defines the fields
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

    See Also:
        GenericSerializer: A generic serializer class that provides the
        functionality for the serializer.
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

        Attributes:
            model (models.GeneralLedgerAccount): The model that the serializer
            is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.GeneralLedgerAccount
        extra_fields = (
            "is_active",
            "account_type",
            "cash_flow_type",
            "account_sub_type",
            "account_classification",
        )

    def update(self, instance, validated_data):
        print("I'm being called.")

class RevenueCodeSerializer(GenericSerializer):
    """A serializer class for the RevenueCode model.

    This serializer is used to convert the RevenueCode model instance into a
    Python dictionary format that can be rendered into a JSON response. It also defines
    the fields that should be included in the serialized representation of the model.

    Attributes:
        expense_account (serializers.PrimaryKeyRelatedField): A primary key related
        field representing the expense account associated with the revenue code.
        The queryset is filtered to only include accounts with an
        AccountTypeChoices value of EXPENSE.

        revenue_account (serializers.PrimaryKeyRelatedField): A primary key related
        field representing the revenue account associated with the revenue code.
        The queryset is filtered to only include accounts with an
        AccountTypeChoices value of REVENUE.

    See Also:
        GenericSerializer: A generic serializer class that provides the
        functionality for the serializer.
    """

    expense_account = GeneralLedgerAccountSerializer()
    revenue_account = GeneralLedgerAccountSerializer()

    class Meta:
        """Metaclass for RevenueCodeSerializer

        Attributes:
            model (models.RevenueCode): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.RevenueCode
        extra_fields = ("expense_account", "revenue_account")

    def update(
        self, instance: models.RevenueCode, validated_data: Any
    ) -> models.RevenueCode:
        """Update the instance with the validated data.

        Args:
            instance (models.RevenueCode): The instance to update.
            validated_data (Any): The validated data.

        Returns:
            models.RevenueCode: The updated instance.
        """

        expense_account_data = validated_data.pop("expense_account")
        revenue_account_data = validated_data.pop("revenue_account")

        if expense_account_data:
            instance.expense_account.update_gl_account(**expense_account_data)

        if revenue_account_data:
            instance.revenue_account.update_gl_account(**revenue_account_data)

        instance.update_revenue_code(**validated_data)

        return instance


class DivisionCodeSerializer(GenericSerializer):
    """A serializer class for the DivisionCode model.

    This serializer is used to convert the DivisionCode model instance into
    a Python dictionary format that can be rendered into a JSON response.
    It also defines the fields that should be included in the serialized
    representation of the model.

    Attributes:
        cash_account (serializers.PrimaryKeyRelatedField): A primary key related
        field representing the cash account associated with the division code.
        The queryset is filtered to only include accounts with an
        AccountTypeChoices value of CASH.

        expense_account (serializers.PrimaryKeyRelatedField): A primary key related
        field representing the expense account associated with the division code.
        The queryset is filtered to only include accounts with an
        AccountTypeChoices value of EXPENSE.

        ap_account (serializers.PrimaryKeyRelatedField): A primary key related
        field representing the accounts payable account associated with the
        division code. The queryset is filtered to only include accounts with an
        AccountTypeChoices value of ACCOUNTS_PAYABLE.

    See Also:
        GenericSerializer: A generic serializer class that provides the
        functionality for the serializer.
    """

    cash_account = serializers.PrimaryKeyRelatedField(
        queryset=models.GeneralLedgerAccount.objects.filter(
            account_type=models.GeneralLedgerAccount.AccountClassificationChoices.CASH
        ),
        required=False,
    )
    ap_account = serializers.PrimaryKeyRelatedField(
        queryset=models.GeneralLedgerAccount.objects.filter(
            account_type=models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE
        ),
        required=False,
    )
    expense_account = serializers.PrimaryKeyRelatedField(
        queryset=models.GeneralLedgerAccount.objects.filter(
            account_type=models.GeneralLedgerAccount.AccountTypeChoices.EXPENSE
        ),
        required=False,
    )

    class Meta:
        """Metaclass for DivisionCodeSerializer

        Attributes:
            model (models.DivisionCode): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.DivisionCode
        extra_fields = ("cash_account", "ap_account", "expense_account")
