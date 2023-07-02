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

    def validate(self, attrs: Any) -> Any:
        """The validate function is called by the serializer's .is_valid() method. It runs field-level
        validations on the data, and then it calls a series of custom validation functions that are defined
        in this class. The custom validation functions are prefixed with an underscore to indicate that
        they're private methods (i.e., not intended for use outside of this class). Each one takes two
        arguments: attrs, which is a dictionary containing all of the data; and key, which is a string
        indicating what field we're validating.

        Args:
            self: Access the attributes of the class
            attrs: Pass in the validated data

        Returns:
            The validated data
        """
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

    def _validate_unique_code_organization(self, attrs: Any) -> None:
        """The _validate_unique_code_organization function is a helper function that validates the uniqueness of the code field.
        It checks to see if there are any other division codes with the same code and organization as this one, excluding itself if it exists.
        If there are, then it raises a serializers.ValidationError.

        Args:
            self: Access the instance of the class
            attrs (Any): Get the code from the request

        Returns:
            None: This function does not return anything.
        """
        code = attrs.get("code")
        division_codes = models.DivisionCode.objects.filter(
            code=code, organization=self.get_organization
        ).exclude(
            pk=self.instance.id if self.instance else None
        )  # type: ignore
        if division_codes:
            raise serializers.ValidationError(
                {"code": "Division code already exists. Please try again."}
            )

    def _validate_account_classification(
        self,
        attrs: Any,
        account_key: str,
        expected_classification: str,
        account_name: str,
    ) -> None:
        """The _validate_account_classification function is a helper function that validates the account classification of an
        account. It takes in four arguments: attrs, account_key, expected_classification and account_name. The attrs argument
        is the attributes dictionary passed to the serializer's create method (or update method). The account key argument is
        the name of the field on which we want to perform validation. The expected classification argument is a string that
        represents what type of classification we expect for this particular field (e.g., 'asset', 'liability', etc.). Finally,
        the last parameter represents what type of

        Args:
            self: Make the function a method of the class
            attrs (Any): Pass in the attributes of the serializer
            account_key(str): Specify the name of the account field in attrs
            expected_classification(str): Determine the account_classification of the account
            account_name(str): Specify the name of the account that is being validated

        Returns:
            None: This function does not return anything.

        Raises:
            serializers.ValidationError: If the account classification of the account does not match the expected classification,
        """
        account = attrs.get(account_key)
        if account and account.account_classification != expected_classification:
            raise serializers.ValidationError(
                {
                    account_key: f"Entered account is not a {account_name} account. Please try again."
                }
            )

    def _validate_account_type(
        self, attrs: Any, account_key: str, expected_type: str, account_name: str
    ) -> None:
        """The _validate_account_type function is a helper function that validates the account type of an account.
        It takes in four arguments: self, attrs, account_key and expected_type. The first argument is the serializer instance itself.
        The second argument is a dictionary containing all of the attributes to be validated (attrs). The third argument
        is a string representing which attribute we want to validate (account_key). And finally, the fourth argument
        is another string representing what type of account we expect this attribute to be (expected_type).

        Args:
            self: Refer to the object itself
            attrs (Any): Pass the attributes of the serializer to this function
            account_key (str): Get the account from attrs
            expected_type (str): Specify the type of account that is expected
            account_name (str): Specify the account type in the error message

        Returns:
            None: This function does not return anything.

        Raises:
            serializers.ValidationError: If the account type is not the expected type, then raise a ValidationError.
        """
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
