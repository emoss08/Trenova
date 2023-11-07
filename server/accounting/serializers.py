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


class TagSerializer(GenericSerializer):
    """A serializer class for the Tag model.

    This serializer is used to convert the Tag model instance into a Python
    dictionary format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    See Also:
        GenericSerializer: A generic serializer class that provides the
        functionality for the serializer.
    """

    class Meta:
        """
        Metaclass for TagSerializer

        Attributes:
            model (models.Tag): The model that the serializer
            is for.
        """

        model = models.Tag


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

    def validate_account_number(self, value: str) -> str:
        """Validate account number does not exist for the organization. Will only apply to
        create operations.

        Args:
            value: Account Number of the General Ledger Account

        Returns:
            str: Account Number of the General Ledger Account
        """
        organization = super().get_organization

        queryset = models.GeneralLedgerAccount.objects.filter(
            organization=organization,
            account_number__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.GeneralLedgerAccount):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "General Ledger Account with this `account_number` already exists. Please try again."
            )

        return value


class RevenueCodeSerializer(GenericSerializer):
    """A serializer class for the RevenueCode model.

    This serializer is used to convert the RevenueCode model instance into a
    Python dictionary format that can be rendered into a JSON response. It also defines
    the fields that should be included in the serialized representation of the model.

    See Also:
        GenericSerializer: A generic serializer class that provides the
        functionality for the serializer.
    """

    rev_account_num = serializers.CharField(
        source="revenue_account.account_number", read_only=True
    )
    exp_account_num = serializers.CharField(
        source="expense_account.account_number", read_only=True
    )

    class Meta:
        """Metaclass for RevenueCodeSerializer

        Attributes:
            model (models.RevenueCode): The model that the serializer is for.
        """

        model = models.RevenueCode
        extra_fields = ("rev_account_num", "exp_account_num")

    def validate_code(self, value: str) -> str:
        """Validate code does not exist for the organization. Will only apply to
        create operations and update operations that change the code.

        Args:
            value: Code of the Revenue Code

        Returns:
            str: Code of the Revenue Code
        """

        organization = super().get_organization

        queryset = models.RevenueCode.objects.filter(
            organization=organization,
            code__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.RevenueCode):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Revenue Code with this `code` already exists. Please try again."
            )

        return value


class DivisionCodeSerializer(GenericSerializer):
    """A serializer class for the DivisionCode model.

    This serializer is used to convert the DivisionCode model instance into
    a Python dictionary format that can be rendered into a JSON response.
    It also defines the fields that should be included in the serialized
    representation of the model.
    """

    class Meta:
        """Metaclass for DivisionCodeSerializer

        Attributes:
            model (models.DivisionCode): The model that the serializer is for.
        """

        model = models.DivisionCode

    def validate_code(self, value: str) -> str:
        """Validate code does not exist for the organization. Will only apply to
        create operations.

        Args:
            value: Name of the Division Code

        Returns:
            str: Name of the Division Code
        """

        organization = super().get_organization

        queryset = models.DivisionCode.objects.filter(
            organization=organization,
            code__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.DivisionCode):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Division Code with this `code` already exists. Please try again."
            )

        return value


class FinancialTransactionSerializer(GenericSerializer):
    """A serializer class for the FinancialTransaction model.

    This serializer is used to convert the FinancialTransaction model instance into a Python
    dictionary format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    See Also:
        GenericSerializer: A generic serializer class that provides the
        functionality for the serializer.
    """

    class Meta:
        """
        Metaclass for FinancialTransactionSerializer

        Attributes:
            model (models.FinancialTransaction): The model that the serializer
            is for.
        """

        model = models.FinancialTransaction


class ReconciliationQueueSerializer(GenericSerializer):
    """A serializer class for the ReconciliationQueue model.

    This serializer is used to convert the ReconciliationQueue model instance into a Python
    dictionary format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    See Also:
        GenericSerializer: A generic serializer class that provides the
        functionality for the serializer.
    """

    class Meta:
        """
        Metaclass for ReconciliationQueueSerializer

        Attributes:
            model (models.ReconciliationQueue): The model that the serializer
            is for.
        """

        model = models.ReconciliationQueue


class AccountingControlSerializer(GenericSerializer):
    """A serializer for the AccountingControl model.

    The serializer provides default operations for creating, updating, and deleting
    Dispatch Control, as well as listing and retrieving Accounting Control. It uses the
    `AccountingControl` model to convert the accounting control instances to and from
    JSON-formatted data.

    Only authenticated users are allowed to access the view provided by this serializer.
    Filtering is also available, with the ability to filter by ID, and name.
    """

    class Meta:
        """
        A class representing the metadata for the `AccountingControlSerializer` class.
        """

        model = models.AccountingControl
