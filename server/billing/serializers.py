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

from djmoney.contrib.django_rest_framework import MoneyField
from rest_framework import serializers

from billing import models
from order.models import Order
from utils.serializers import GenericSerializer


class BillingControlSerializer(GenericSerializer):
    """A serializer for the `BillingControl` model.

    A serializer class for the BillingControl model. This serializer is used to convert BillingControl model
    instances into a Python dictionary format that can be rendered into a JSON response. It also defined the
    fields that should be included in the serialized representation of the model
    """

    class Meta:
        """
        Metaclass for the BillingControlSerializer

        Attributes:
            model (BillingControl): The model that the serializer is for.
        """

        model = models.BillingControl


class BillingTransferLogSerializer(GenericSerializer):
    """A serializer for the `BillingTransferLog` model.

    A serializer class for the BillingTransferLog Model. This serializer is used to convert the BillingTransferLog
    model instances into a Python dictionary format that can be rendered into a JSON response. It also defines
    the fields that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for BillingTransferLogSerializer

        Attributes:
            model (models.BillingTransferLog): The model that the serializer is for.
        """

        model = models.BillingTransferLog


class BillingQueueSerializer(GenericSerializer):
    """A serializer for the `BillingQueue` model.

    A serializer class for the BillingQueue Model. This serializer is used to convert the BillingQueue
    model instances into a Python dictionary format that can be rendered into a JSON response. It
    also defines the fields that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for BillingQueueSerializer

        Attributes:
            model (models.BillingQueue): The model that the serializer is for.
        """

        model = models.BillingQueue

    def to_representation(self, instance: Order) -> dict[str, Any]:
        data = super().to_representation(instance)
        data["customer_name"] = instance.customer.name
        return data


class BillingHistorySerializer(GenericSerializer):
    """A serializer for the `BillingHistory` model.

    A serializer class for the BillingHistory Model. This serializer is used
    to convert the BillingHistory model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for BillingHistorySerializer

        Attributes:
            model (models.BillingHistory): The model that the serializer is for.
        """

        model = models.BillingHistory


class ChargeTypeSerializer(GenericSerializer):
    """
    A serializer for the `ChargeType` model.

    This serializer converts instances of the `ChargeType` model into JSON or other data formats,
    and vice versa. It uses the specified fields (id, name, and description) to create
    the serialized representation of the `ChargeType` model.
    """

    class Meta:
        """
        A class representing the metadata for the `ChargeTypeSerializer` class.
        """

        model = models.ChargeType


class AccessorialChargeSerializer(GenericSerializer):
    """
    A serializer for the `AccessorialCharge` model.

    This serializer converts instances of the `AccessorialCharge` model into JSON
    or other data formats, and vice versa. It uses the specified fields
    (code, is_detention, charge_amount, and method) to create the serialized
    representation of the `AccessorialCharge` model.
    """

    def validate(self, attrs: dict[str, Any]) -> Any:
        if attrs.get("charge_amount") == 0:
            raise serializers.ValidationError(
                {"charge_amount": "Charge amount must be greater than zero."}
            )
        return attrs

    class Meta:
        """k
        A class representing the metadata for the `AccessorialChargeSerializer` class.
        """

        model = models.AccessorialCharge


class DocumentClassificationSerializer(GenericSerializer):
    """
    A serializer for the `DocumentClassification` model.

    This serializer converts instances of the `DocumentClassification` model into JSON or other data
    formats, and vice versa. It uses the specified fields (id, name, and description) to create the
    serialized representation of the `DocumentClassification` model.
    """

    class Meta:
        """
        A class representing the metadata for the `DocumentClassificationSerializer` class.
        """

        model = models.DocumentClassification


class OrdersReadySerializer(serializers.Serializer):
    id = serializers.UUIDField(read_only=True)
    pro_number = serializers.CharField(
        help_text="Pro Number of the Order", label="Pro Number", read_only=True
    )
    mileage = serializers.FloatField(
        allow_null=True,
        help_text="Total Mileage",
        label="Total Mileage",
        required=False,
    )
    other_charge_amount = MoneyField(
        decimal_places=4,
        help_text="Additional Charge Amount",
        label="Additional Charge Amount",
        max_digits=19,
        required=False,
    )
    freight_charge_amount = MoneyField(
        decimal_places=4,
        help_text="Freight Charge Amount",
        label="Freight Charge Amount",
        max_digits=19,
        required=False,
    )
    sub_total = MoneyField(
        decimal_places=4,
        help_text="Sub Total",
        label="Sub Total",
        max_digits=19,
        required=False,
    )

    def find_missing_documents(self, instance: Order) -> tuple[list[str], bool]:
        missing_documents = []
        is_missing_documents = False

        if billing_profile := instance.customer.billing_profile:  # type: ignore
            if rule_profile := billing_profile.rule_profile:
                # Get the name of each document_class required in CustomerRuleProfile
                required_documents = rule_profile.document_class.all()

                # Query the OrderDocumentation for each document_class
                for required_document in required_documents:
                    order_document = instance.order_documentation.filter(
                        document_class=required_document
                    ).first()
                    if not order_document:
                        missing_documents.append(required_document.name)
                        is_missing_documents = True

        return missing_documents, is_missing_documents

    def to_representation(self, instance: Order) -> dict[str, Any]:
        data = super().to_representation(instance)
        data["customer_name"] = instance.customer.name
        missing_documents, is_missing_documents = self.find_missing_documents(instance)
        data["missing_documents"] = missing_documents
        data["is_missing_documents"] = is_missing_documents
        return data
