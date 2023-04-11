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

from rest_framework import serializers

from accounting import models
from organization.models import Organization
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

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )
    is_active = serializers.BooleanField(default=True)
    account_type = serializers.ChoiceField(
        choices=models.GeneralLedgerAccount.AccountTypeChoices.choices
    )
    cash_flow_type = serializers.ChoiceField(
        choices=models.GeneralLedgerAccount.CashFlowTypeChoices.choices,
        allow_null=True,
        required=False,
    )
    account_sub_type = serializers.ChoiceField(
        choices=models.GeneralLedgerAccount.AccountSubTypeChoices.choices,
        allow_null=True,
        required=False,
    )
    account_classification = serializers.ChoiceField(
        choices=models.GeneralLedgerAccount.AccountClassificationChoices.choices,
        allow_null=True,
        required=False,
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
            "organization",
            "is_active",
            "account_type",
            "cash_flow_type",
            "account_sub_type",
            "account_classification",
        )


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

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )
    expense_account = serializers.PrimaryKeyRelatedField(
        queryset=models.GeneralLedgerAccount.objects.filter(
            account_type=models.GeneralLedgerAccount.AccountTypeChoices.EXPENSE
        ),
        allow_null=True,
        required=False,
    )
    revenue_account = serializers.PrimaryKeyRelatedField(
        queryset=models.GeneralLedgerAccount.objects.filter(
            account_type=models.GeneralLedgerAccount.AccountTypeChoices.REVENUE
        ),
        allow_null=True,
        required=False,
    )

    class Meta:
        """Metaclass for RevenueCodeSerializer

        Attributes:
            model (models.RevenueCode): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.RevenueCode
        extra_fields = ("organization", "expense_account", "revenue_account")


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

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )
    is_active = serializers.BooleanField(default=True)
    cash_account = serializers.PrimaryKeyRelatedField(
        queryset=models.GeneralLedgerAccount.objects.filter(
            account_classification=models.GeneralLedgerAccount.AccountClassificationChoices.CASH
        ),
        allow_null=True,
        required=False,
    )
    ap_account = serializers.PrimaryKeyRelatedField(
        queryset=models.GeneralLedgerAccount.objects.filter(
            account_type=models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE
        ),
        allow_null=True,
        required=False,
    )
    expense_account = serializers.PrimaryKeyRelatedField(
        queryset=models.GeneralLedgerAccount.objects.filter(
            account_type=models.GeneralLedgerAccount.AccountTypeChoices.EXPENSE
        ),
        allow_null=True,
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
        extra_fields = (
            "organization",
            "is_active",
            "cash_account",
            "ap_account",
            "expense_account",
        )
