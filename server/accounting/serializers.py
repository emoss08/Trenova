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
import typing

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

    def to_representation(self, instance: models.RevenueCode) -> dict[str, typing.Any]:
        """The to_representation method is used to convert an instance of the models.RevenueCode class into a dictionary
        representation. It extends the functionality of the GenericSerializer's to_representation method.

        The resulting dictionary will contain the representation of the revenue code instance, including additional
        fields 'rev_account_num' and 'exp_account_num'. 'rev_account_num' will be set to the revenue account number
        if it exists, otherwise it will be set to None. 'exp_account_num' will be set to the expense account number.

        Notes:
            This method does not modify the instance itself, it only converts its data into a dictionary
            representation.

        Args:
            instance(RevenueCode): An instance of the models.RevenueCode class representing a revenue code.

        Returns:
            A dictionary containing the representation of the revenue code instance.
        """
        data = super().to_representation(instance=instance)
        data["rev_account_num"] = (
            instance.revenue_account.account_number
            if instance.revenue_account
            else None
        )
        data["exp_account_num"] = (
            instance.expense_account.account_number
            if instance.expense_account
            else None
        )
        return data


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
