"""
COPYRIGHT 2022 MONTA

This file is part of Monta.

Monta is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

Monta is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with Monta.  If not, see <https://www.gnu.org/licenses/>.
"""

from rest_framework import serializers

from accounting.models import RevenueCode
from accounting.serializers import RevenueCodeSerializer
from accounts.models import User
from accounts.serializers import UserSerializer
from commodities.models import Commodity, HazardousMaterial
from commodities.serializers import CommoditySerializer, HazardousMaterialSerializer
from customer.models import Customer
from customer.serializers import CustomerSerializer
from equipment.models import EquipmentType
from equipment.serializers import EquipmentTypeSerializer
from location.models import Location
from location.serializers import LocationSerializer
from movements.models import Movement
from utils.models import StatusChoices
from utils.serializers import GenericSerializer
from order import models


class OrderControlSerializer(GenericSerializer):
    """A serializer for the `OrderControl` model.

    A serializer class for the OrderControl model. This serializer is used
    to convert the OrderControl model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model
    """

    class Meta:
        """
        Metaclass for OrderControlSerializer

        Attributes:
            model (OrderControl): The model that the serializer is for.
        """

        model = models.OrderControl


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

    class Meta:
        """Metaclass for OrderTypeSerializer

        Attributes:
            model (models.OrderType): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.OrderType
        extra_fields = ("is_active",)


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

    class Meta:
        """Metaclass for ReasonCodeSerializer

        Attributes:
            model (models.ReasonCode): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.ReasonCode
        extra_fields = ("code_type",)


class OrderSerializer(GenericSerializer):
    """A serializer for the `Order` model.

    A serializer class for the Order Model. This serializer is used
    to convert the Order model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    Attributes:
    """

    order_type = serializers.PrimaryKeyRelatedField(
        queryset=models.OrderType.objects.all(),
        allow_null=True,
    )
    revenue_code = serializers.PrimaryKeyRelatedField(
        queryset=RevenueCode.objects.all(),
        allow_null=True,
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
        queryset=Commodity.objects.all(), allow_null=True
    )
    entered_by = serializers.PrimaryKeyRelatedField(
        queryset=User.objects.all(),
    )
    hazmat = serializers.PrimaryKeyRelatedField(
        queryset=HazardousMaterial.objects.all(),
        allow_null=True,
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
        )
