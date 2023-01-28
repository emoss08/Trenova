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
from accounts.models import User
from billing import models
from commodities.models import Commodity
from customer.models import Customer
from order.models import OrderType, Order
from utils.serializers import GenericSerializer
from worker.models import Worker


class BillingControlSerializer(GenericSerializer):
    """A serializer for the `BillingControl` model.

    A serializer class for the BillingControl model. This serializer is used
    to convert BillingControl model instances into a Python dictionary format
    that can be rendered into a JSON response. It also defined the fields that
    should be included in the serialized representation of the model
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

    A serializer class for the BillingQueue Model. This serializer is used
    to convert the BillingQueue model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    Attributes:
        order_type (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the type of the BillingQueue.
        revenue_code (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the revenue code of the BillingQueue.
        customer (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the customer of the BillingQueue.
        worker (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the worker of BillingQueue.
        commodity (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the commodity of the BillingQueue.
        user (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the user that entered the BillingQueue.
    """

    order_type = serializers.PrimaryKeyRelatedField(
        queryset=OrderType.objects.all(),
    )
    order = serializers.PrimaryKeyRelatedField(
        queryset=Order.objects.all(),
    )
    revenue_code = serializers.PrimaryKeyRelatedField(
        queryset=RevenueCode.objects.all(),
        required=False,
        allow_null=True,
    )
    customer = serializers.PrimaryKeyRelatedField(
        queryset=Customer.objects.all(),
    )
    worker = serializers.PrimaryKeyRelatedField(
        queryset=Worker.objects.all(),
    )
    commodity = serializers.PrimaryKeyRelatedField(
        queryset=Commodity.objects.all(),
        required=False,
        allow_null=True
    )
    user = serializers.PrimaryKeyRelatedField(
        queryset=User.objects.all(),
        required=False,
        allow_null=True
    )

    class Meta:
        """Metaclass for BillingQueueSerializer

        Attributes:
            model (models.BillingQueue): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.BillingQueue
        extra_fields = (
            "order_type",
            "order",
            "revenue_code",
            "customer",
            "worker",
            "commodity",
            "user",
        )



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

    method = serializers.ChoiceField(choices=models.FuelMethodChoices.choices)

    class Meta:
        """
        A class representing the metadata for the `AccessorialChargeSerializer` class.
        """

        model = models.AccessorialCharge
        extra_fields = ("method",)


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
