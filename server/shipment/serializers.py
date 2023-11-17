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

from movements.models import Movement
from shipment import models
from utils.serializers import GenericSerializer


class ShipmentControlSerializer(GenericSerializer):
    """A serializer for the `ShipmentControl` model.

    A serializer class for the ShipmentControl model. This serializer is used
    to convert the ShipmentControl model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model
    """

    class Meta:
        """
        Metaclass for ShipmentControlSerializer

        Attributes:
            model (ShipmentControl): The model that the serializer is for.
        """

        model = models.ShipmentControl


class ShipmentTypeSerializer(GenericSerializer):
    """A serializer for the `ShipmentType` model.

    A serializer class for the ShipmentType Model. This serializer is used
    to convert the ShipmentType model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    """

    class Meta:
        """Metaclass for ShipmentTypeSerializer

        Attributes:
            model (models.ShipmentType): The model that the serializer is for.
        """

        model = models.ShipmentType

    def validate_code(self, value: str) -> str:
        """Validate the `code` field of the ShipmentType model.

        This method validates the `code` field of the ShipmentType model.
        It checks if the shipment type with the given name already exists in the organization.
        If the shipment type exists, it raises a validation error.

        Args:
            value (str): The value of the `code` field.

        Returns:
            str: The value of the `code` field.

        Raises:
            serializers.ValidationError: If the shipment type with the given code already exists in the
             organization.
        """
        organization = super().get_organization

        queryset = models.ShipmentType.objects.filter(
            organization=organization,
            code__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.ShipmentType):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Shipment type with this `code` already exists. Please try again."
            )

        return value


class ReasonCodeSerializer(GenericSerializer):
    """A serializer for the `ReasonCode` model.

    A serializer class for the ReasonCode Model. This serializer is used
    to convert the ReasonCode model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for ReasonCodeSerializer

        Attributes:
            model (models.ReasonCode): The model that the serializer is for.
        """

        model = models.ReasonCode

    def validate_code(self, value: str) -> str:
        """Validate the `code` field of the ReasonCode model.

        This method validates the `code` field of the ReasonCode model.
        It checks if the reason code with the given code already exists in the organization.
        If the reason code exists, it raises a validation error.

        Args:
            value (str): The value of the `code` field.

        Returns:
            str: The value of the `code` field.

        Raises:
            serializers.ValidationError: If the reason code with the given code already exists in the
             organization.
        """
        organization = super().get_organization

        queryset = models.ReasonCode.objects.filter(
            organization=organization,
            code__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.ReasonCode):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Reason Code with this `code` already exists. Please try again."
            )

        return value


class ShipmentSerializer(GenericSerializer):
    """A serializer for the `Order` model.

    A serializer class for the shipment Model. This serializer is used
    to convert the shipment model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    additional_charges = serializers.PrimaryKeyRelatedField(
        many=True,
        queryset=models.AdditionalCharge.objects.all(),
        help_text="Additional charges for the shipment",
        required=False,
        allow_null=True,
    )
    movements = serializers.PrimaryKeyRelatedField(
        many=True,
        queryset=Movement.objects.all(),
        help_text="Movements for the shipment",
        required=False,
        allow_null=True,
    )
    shipment_documentation = serializers.PrimaryKeyRelatedField(
        many=True,
        queryset=models.ShipmentDocumentation.objects.all(),
        help_text="Documentation for the shipment",
        required=False,
        allow_null=True,
    )
    shipment_comments = serializers.PrimaryKeyRelatedField(
        many=True,
        queryset=models.ShipmentComment.objects.all(),
        help_text="Comments for the shipment",
        required=False,
        allow_null=True,
    )

    class Meta:
        """Metaclass for ShipmentSerializer

        Attributes:
            model (models.Shipment): The model that the serializer is for.
        """

        model = models.Shipment
        extra_fields = (
            "additional_charges",
            "movements",
            "shipment_documentation",
            "shipment_comments",
        )


class ShipmentDocumentationSerializer(GenericSerializer):
    """A serializer for the `ShipmentDocumentation` model.

    A serializer class for the ShipmentDocumentation Model. This serializer is used
    to convert the ShipmentDocumentation model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for ShipmentDocumentationSerializer

        Attributes:
            model (models.ShipmentDocumentation): The model that the serializer is for.
        """

        model = models.ShipmentDocumentation


class ShipmentCommentSerializer(GenericSerializer):
    """A serializer for the `ShipmentComment` model.

    A serializer class for the ShipmentComment Model. This serializer is used
    to convert the ShipmentComment model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for ShipmentCommentSerializer

        Attributes:
            model (models.ShipmentComment): The model that the serializer is for.
        """

        model = models.ShipmentComment


class AdditionalChargeSerializer(GenericSerializer):
    """A serializer for the `AdditionalCharge` model.

    A serializer class for the AdditionalCharge Model. This serializer is used
    to convert the AdditionalCharge model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    """

    class Meta:
        """Metaclass for AdditionalChargeSerializer

        Attributes:
            model (models.AdditionalCharge): The model that the serializer is for.
        """

        model = models.AdditionalCharge


class ServiceTypeSerializer(GenericSerializer):
    """A serializer for the `ServiceType` model.

    A serializer class for the ServiceType Model. This serializer is used
    to convert the ServiceType model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for ServiceTypeSerializer

        Attributes:
            model (models.ServiceType): The model that the serializer is for.
        """

        model = models.ServiceType
