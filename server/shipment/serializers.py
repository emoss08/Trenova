# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
#                                                                                                  -
#  The Trenova software is licensed under the Business Source License 1.1. You are granted the right
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

from movements.serializers import MovementSerializer
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
        fields = "__all__"
        read_only_fields = ("organization", "business_unit")
        extra_kwargs = {
            "organization": {"required": False},
            "business_unit": {"required": False},
        }


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
        fields = "__all__"
        read_only_fields = ("organization", "business_unit")
        extra_kwargs = {
            "organization": {"required": False, "allow_null": True},
            "business_unit": {"required": False, "allow_null": True},
        }

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
        fields = "__all__"
        read_only_fields = ("organization", "business_unit")
        extra_kwargs = {
            "organization": {"required": False},
            "business_unit": {"required": False},
        }

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
    """A serializer for the `Shipment` model.

    A serializer class for the Shipment Model. This serializer is used
    to convert the shipment model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    movements = MovementSerializer(many=True, required=False)

    class Meta:
        """Metaclass for ShipmentSerializer

        Attributes:
            model (models.Shipment): The model that the serializer is for.
        """

        model = models.Shipment
        fields = "__all__"
        read_only_fields = (
            "organization",
            "business_unit",
            "temperature_min",
            "temperature_max",
        )
        extra_kwargs = {
            "organization": {"required": False},
            "business_unit": {"required": False},
        }


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
        fields = "__all__"
        read_only_fields = ("organization", "business_unit")
        extra_kwargs = {
            "organization": {"required": False},
            "business_unit": {"required": False},
        }


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
        fields = "__all__"
        read_only_fields = ("organization", "business_unit")
        extra_kwargs = {
            "organization": {"required": False},
            "business_unit": {"required": False},
        }


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
        fields = "__all__"
        read_only_fields = ("organization", "business_unit")
        extra_kwargs = {
            "organization": {"required": False},
            "business_unit": {"required": False},
        }


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
        fields = "__all__"
        read_only_fields = ("organization", "business_unit")
        extra_kwargs = {
            "organization": {"required": False},
            "business_unit": {"required": False},
        }


class FormulaTemplateSerializer(GenericSerializer):
    """A serializer for the `FormulaTemplate` model.

    A serializer class for the FormulaTemplate Model. This serializer is used
    to convert the FormulaTemplate model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for Formula Template Serializer

        Attributes:
            model (models.FormulaTemplate): The model that the serializer is for.
        """

        model = models.FormulaTemplate
        fields = "__all__"
        read_only_fields = ("organization", "business_unit")
        extra_kwargs = {
            "organization": {"required": False},
            "business_unit": {"required": False},
        }
