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
import decimal
from typing import Any

from rest_framework import serializers

from billing import models
from shipment.models import Shipment
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

    def to_representation(self, instance: Shipment) -> dict[str, Any]:
        data = super().to_representation(instance)
        data["customer_name"] = instance.customer.name
        return data


class BillingLogEntrySerializer(GenericSerializer):
    """A serializer for the `BillingLogEntry` model.

    A serializer class for the BillingLogEntry Model. This serializer is used
    to convert the BillingLogEntry model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for BillingLogEntrySerializer

        Attributes:
            model (models.BillingLogEntry): The model that the serializer is for.
        """

        model = models.BillingLogEntry


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

    def validate_name(self, value: str) -> str:
        """Validate name does not exist for the organization. Will only apply to
        create operations.

        Args:
            value: Name of the Charge Type

        Returns:
            str: Name of the Charge Type

        """

        organization = super().get_organization

        if (
            self.instance is None
            and models.ChargeType.objects.filter(
                organization=organization,
                name__iexact=value,
            ).exists()
        ):
            raise serializers.ValidationError(
                "Charge Type with this name already exists. Please try again."
            )

        return value


class AccessorialChargeSerializer(GenericSerializer):
    """
    A serializer for the `AccessorialCharge` model.

    This serializer converts instances of the `AccessorialCharge` model into JSON
    or other data formats, and vice versa. It uses the specified fields
    (code, is_detention, charge_amount, and method) to create the serialized
    representation of the `AccessorialCharge` model.
    """

    class Meta:
        """k
        A class representing the metadata for the `AccessorialChargeSerializer` class.
        """

        model = models.AccessorialCharge

    def validate_charge_amount(self, value: decimal.Decimal) -> decimal.Decimal:
        """Validates the charge amount for an accessorial charge.

        Args:
            value (decimal.Decimal): The charge amount to be validated.

        Returns:
            decimal.Decimal: The validated charge amount.

        Raises:
            serializers.ValidationError: If the charge amount is zero or less.
        """
        if value < 1:
            raise serializers.ValidationError(
                "Charge amount must be greater than zero. Please try again."
            )

        return value

    def validate_code(self, value: str) -> str:
        """Validate code does not exist for the organization. Will only apply to
        create operations.

        Args:
            value: Code of the Accessorial Charge

        Returns:
            str: Code of the Accessorial Charge

        """

        organization = super().get_organization

        queryset = models.AccessorialCharge.objects.filter(
            organization=organization,
            code__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.AccessorialCharge):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Accessorial Charge with this `code` already exists. Please try again."
            )

        return value


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

    def validate_name(self, value: str) -> str:
        """This method validates the name of a document classification instance. It checks if a document classification
        with the same name already exists for the organization.

        Args:
            value (str): The name of the document classification.

        Returns:
            str: The validated name if no duplicates are found.

        Raises:
            serializers.ValidationError: If a document classification with the same name already exists.
        """

        organization = super().get_organization

        queryset = models.DocumentClassification.objects.filter(
            organization=organization,
            name__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.DocumentClassification):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Document Classification with this `name` already exists. Please try again."
            )

        return value


class shipmentsReadySerializer(serializers.Serializer):
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
    other_charge_amount = serializers.DecimalField(
        decimal_places=4,
        help_text="Additional Charge Amount",
        label="Additional Charge Amount",
        max_digits=19,
        required=False,
    )
    freight_charge_amount = serializers.DecimalField(
        decimal_places=4,
        help_text="Freight Charge Amount",
        label="Freight Charge Amount",
        max_digits=19,
        required=False,
    )
    sub_total = serializers.DecimalField(
        decimal_places=4,
        help_text="Sub Total",
        label="Sub Total",
        max_digits=19,
        required=False,
    )

    def find_missing_documents(self, instance: Shipment) -> tuple[list[str], bool]:
        """Determines missing documents for an order based on the rule_profile of the customer of the shipment.
        Uses the document class list in each rule_profile to find courses that require its students to
        submit documents. It will then return a list of missing document names and a boolean flag
        indicating if any documents are missing.

        Args:
            instance (Order): Order object for which to find missing documents.

        Returns:
            tuple[list[str], bool]: A tuple containing list of names of missing documents
                                    and a boolean indicating if any documents are missing.

        Note:
            This function will perform a database query for each document_class in the
            rule_profile of the customer.

        Raises:
            AttributeError: An error occurred when the order instance has no customer
                            or the customer has no rule_profile.
        """
        missing_documents = []
        is_missing_documents = False

        if rule_profile := instance.customer.rule_profile.exists():
            # Get the name of each document_class required in CustomerRuleProfile
            required_documents = rule_profile.document_class.all()

            # Query the ShipmentDocumentation for each document_class
            for required_document in required_documents:
                shipment_document = instance.shipment_documentation.filter(
                    document_class=required_document
                ).first()
                if not shipment_document:
                    missing_documents.append(required_document.name)
                    is_missing_documents = True

        return missing_documents, is_missing_documents

    def to_representation(self, instance: Shipment) -> dict[str, Any]:
        data = super().to_representation(instance)
        data["customer_name"] = instance.customer.name
        missing_documents, is_missing_documents = self.find_missing_documents(instance)
        data["missing_documents"] = missing_documents
        data["is_missing_documents"] = is_missing_documents
        return data
