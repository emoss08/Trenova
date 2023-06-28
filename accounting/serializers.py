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
from utils.serializers import GenericSerializer


class GeneralLedgerAccountSerializer(GenericSerializer):
    """A serializer class for the GeneralLedgerAccount model.

    This serializer is used to convert the GeneralLedgerAccount model instance into a Python
    dictionary format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    See Also:
        GenericSerializer: A generic serializer class that provides the
        functionality for the serializer.
    """

    class Meta:
        """
        Metaclass for GeneralLedgerAccountSerializer

        Attributes:
            model (models.GeneralLedgerAccount): The model that the serializer
            is for.
        """

        model = models.GeneralLedgerAccount


class RevenueCodeSerializer(GenericSerializer):
    """A serializer class for the RevenueCode model.

    This serializer is used to convert the RevenueCode model instance into a
    Python dictionary format that can be rendered into a JSON response. It also defines
    the fields that should be included in the serialized representation of the model.

    See Also:
        GenericSerializer: A generic serializer class that provides the
        functionality for the serializer.
    """

    class Meta:
        """Metaclass for RevenueCodeSerializer

        Attributes:
            model (models.RevenueCode): The model that the serializer is for.
        """

        model = models.RevenueCode


class DivisionCodeSerializer(GenericSerializer):
    """A serializer class for the DivisionCode model.

    This serializer is used to convert the DivisionCode model instance into
    a Python dictionary format that can be rendered into a JSON response.
    It also defines the fields that should be included in the serialized
    representation of the model.
    """

    def validate(self, attrs):
        self._validate_unique_code_organization(attrs)
        self._validate_account_classification(
            attrs,
            "cash_account",
            models.GeneralLedgerAccount.AccountClassificationChoices.CASH,
            "cash",
        )
        self._validate_account_type(
            attrs,
            "expense_account",
            models.GeneralLedgerAccount.AccountTypeChoices.EXPENSE,
            "expense",
        )
        self._validate_account_classification(
            attrs,
            "ap_account",
            models.GeneralLedgerAccount.AccountClassificationChoices.ACCOUNTS_PAYABLE,
            "accounts payable",
        )

        return attrs

    def _validate_unique_code_organization(self, attrs):
        code = attrs.get("code")
        division_codes = models.DivisionCode.objects.filter(
            code=code, organization=self.get_organization
        ).exclude(pk=self.instance.pk if self.instance else None)
        if division_codes:
            raise serializers.ValidationError(
                {"code": "Division code already exists. Please try again."}
            )

    def _validate_account_classification(
        self, attrs, account_key, expected_classification, account_name
    ):
        account = attrs.get(account_key)
        if account and account.account_classification != expected_classification:
            raise serializers.ValidationError(
                {
                    account_key: f"Entered account is not a {account_name} account. Please try again."
                }
            )

    def _validate_account_type(self, attrs, account_key, expected_type, account_name):
        account = attrs.get(account_key)
        if account and account.account_type != expected_type:
            raise serializers.ValidationError(
                {
                    account_key: f"Entered account is not an {account_name} account. Please try again."
                }
            )

    class Meta:
        """Metaclass for DivisionCodeSerializer

        Attributes:
            model (models.DivisionCode): The model that the serializer is for.
        """

        model = models.DivisionCode
