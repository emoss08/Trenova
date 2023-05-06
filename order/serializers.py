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

from accounting.models import RevenueCode
from accounts.models import User
from billing.models import AccessorialCharge, DocumentClassification
from commodities.models import Commodity, HazardousMaterial
from customer.models import Customer
from dispatch.models import CommentType
from equipment.models import EquipmentType
from location.models import Location
from movements.models import Movement
from order import models
from organization.models import Organization
from utils.serializers import GenericSerializer


class OrderControlSerializer(GenericSerializer):
    """A serializer for the `OrderControl` model.

    A serializer class for the OrderControl model. This serializer is used
    to convert the OrderControl model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )

    class Meta:
        """
        Metaclass for OrderControlSerializer

        Attributes:
            model (OrderControl): The model that the serializer is for.
        """

        model = models.OrderControl
        extra_fields = ("organization",)


class OrderTypeSerializer(GenericSerializer):
    """A serializer for the `OrderType` model.

    A serializer class for the OrderType Model. This serializer is used
    to convert the OrderType model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    Attributes:
        is_active (BooleanField): A boolean field that determines if the order type is active.
    """

    is_active = serializers.BooleanField(default=True)
    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )

    class Meta:
        """Metaclass for OrderTypeSerializer

        Attributes:
            model (models.OrderType): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.OrderType
        extra_fields = (
            "is_active",
            "organization",
        )


class ReasonCodeSerializer(GenericSerializer):
    """A serializer for the `ReasonCode` model.

    A serializer class for the ReasonCode Model. This serializer is used
    to convert the ReasonCode model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    Attributes:
        code_type (ChoiceField): A choice field that determines the type of the reason code.
    """

    code_type = serializers.ChoiceField(
        choices=models.ReasonCode.CodeTypeChoices.choices
    )
    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )

    class Meta:
        """Metaclass for ReasonCodeSerializer

        Attributes:
            model (models.ReasonCode): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.ReasonCode
        extra_fields = (
            "code_type",
            "organization",
        )


class OrderSerializer(GenericSerializer):
    order_type = serializers.PrimaryKeyRelatedField(
        queryset=models.OrderType.objects.all(),
    )
    revenue_code = serializers.PrimaryKeyRelatedField(
        queryset=RevenueCode.objects.all(),
        allow_null=True,
        required=False,
    )
    origin_location = serializers.PrimaryKeyRelatedField(
        queryset=Location.objects.all(),
        allow_null=True,
    )
    destination_location = serializers.PrimaryKeyRelatedField(
        queryset=Location.objects.all(),
        allow_null=True,
    )
    customer = serializers.PrimaryKeyRelatedField(
        queryset=Customer.objects.all(),
    )
    equipment_type = serializers.PrimaryKeyRelatedField(
        queryset=EquipmentType.objects.all()
    )
    commodity = serializers.PrimaryKeyRelatedField(
        queryset=Commodity.objects.all(),
        allow_null=True,
        required=False,
    )
    entered_by = serializers.PrimaryKeyRelatedField(
        queryset=User.objects.all(),
    )
    hazmat = serializers.PrimaryKeyRelatedField(
        queryset=HazardousMaterial.objects.all(),
        allow_null=True,
        required=False,
    )
    movements = serializers.PrimaryKeyRelatedField(
        queryset=Movement.objects.all(),
        many=True,
        allow_null=True,
        required=False,
    )
    order_documentation = serializers.PrimaryKeyRelatedField(
        queryset=models.OrderDocumentation.objects.all(),
        many=True,
        allow_null=True,
        required=False,
    )
    order_comments = serializers.PrimaryKeyRelatedField(
        queryset=models.OrderComment.objects.all(),
        many=True,
        allow_null=True,
        required=False,
    )
    additional_charges = serializers.PrimaryKeyRelatedField(
        queryset=models.AdditionalCharge.objects.all(),
        many=True,
        allow_null=True,
        required=False,
    )

    class Meta:
        """Metaclass for OrderSerializer

        Attributes:
            model (models.Order): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.Order
        extra_fields = (
            "order_type",
            "revenue_code",
            "origin_location",
            "destination_location",
            "customer",
            "equipment_type",
            "commodity",
            "entered_by",
            "hazmat",
            "movements",
            "order_documentation",
            "order_comments",
            "additional_charges",
        )


class OrderDocumentationSerializer(GenericSerializer):
    """A serializer for the `OrderDocumentation` model.

    A serializer class for the OrderDocumentation Model. This serializer is used
    to convert the OrderDocumentation model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    Attributes:
        order (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the order of the order documentation.
        document_class (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the document classification of the order documentation.
    """

    order = serializers.PrimaryKeyRelatedField(
        queryset=models.Order.objects.all(),
    )
    document_class = serializers.PrimaryKeyRelatedField(
        queryset=DocumentClassification.objects.all()
    )

    class Meta:
        """Metaclass for OrderDocumentationSerializer

        Attributes:
            model (models.OrderDocumentation): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.OrderDocumentation
        extra_fields = ("order", "document_class")


class OrderCommentSerializer(GenericSerializer):
    """A serializer for the `OrderComment` model.

    A serializer class for the OrderComment Model. This serializer is used
    to convert the OrderComment model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    Attributes:
        order (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the order of the order comment.
    """

    order = serializers.PrimaryKeyRelatedField(
        queryset=models.Order.objects.all(),
    )
    comment_type = serializers.PrimaryKeyRelatedField(
        queryset=CommentType.objects.all(),
    )
    entered_by = serializers.PrimaryKeyRelatedField(
        queryset=User.objects.all(),
    )

    class Meta:
        """Metaclass for OrderCommentSerializer

        Attributes:
            model (models.OrderComment): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.OrderComment
        extra_fields = ("order", "comment_type", "entered_by")


class AdditionalChargeSerializer(GenericSerializer):
    """A serializer for the `AdditionalCharge` model.

    A serializer class for the AdditionalCharge Model. This serializer is used
    to convert the AdditionalCharge model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    Attributes:
        order (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the order of the additional charge.
        accessorial_charge (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the charge of the additional charge.
        entered_by (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the user that entered the additional charge.
    """

    order = serializers.PrimaryKeyRelatedField(
        queryset=models.Order.objects.all(),
    )
    accessorial_charge = serializers.PrimaryKeyRelatedField(
        queryset=AccessorialCharge.objects.all(),
    )
    entered_by = serializers.PrimaryKeyRelatedField(
        queryset=User.objects.all(),
    )

    class Meta:
        """Metaclass for AdditionalChargeSerializer

        Attributes:
            model (models.AdditionalCharge): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.AdditionalCharge
        extra_fields = ("order", "accessorial_charge", "entered_by")
