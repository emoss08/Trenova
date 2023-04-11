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
from billing import models
from commodities.models import Commodity
from customer.models import Customer
from order.models import Order, OrderType
from organization.models import Organization
from utils.serializers import GenericSerializer
from worker.models import Worker


class BillingControlSerializer(GenericSerializer):
    """A serializer for the `BillingControl` model.

    A serializer class for the BillingControl model. This serializer is used
    to convert BillingControl model instances into a Python dictionary format
    that can be rendered into a JSON response. It also defined the fields that
    should be included in the serialized representation of the model
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )

    class Meta:
        """
        Metaclass for the BillingControlSerializer

        Attributes:
            model (BillingControl): The model that the serializer is for.
        """

        model = models.BillingControl
        extra_fields = ("organization",)


class BillingTransferLogSerializer(GenericSerializer):
    """A serializer for the `BillingTransferLog` model.

    A serializer class for the BillingTransferLog Model. This serializer is used
    to convert the BillingTransferLog model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    Attributes:
        order (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the order of the BillingTransferLog.
        transferred_by (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the transferred_by of the BillingTransferLog.
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )
    order = serializers.PrimaryKeyRelatedField(
        queryset=Order.objects.all(),
    )
    transferred_by = serializers.PrimaryKeyRelatedField(
        queryset=User.objects.all(),
    )

    class Meta:
        """Metaclass for BillingTransferLogSerializer

        Attributes:
            model (models.BillingTransferLog): The model that the serializer is for.
        """

        model = models.BillingTransferLog
        extra_fields = (
            "organization",
            "order",
            "transferred_by",
        )


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

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )
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
        queryset=Commodity.objects.all(), required=False, allow_null=True
    )
    user = serializers.PrimaryKeyRelatedField(
        queryset=User.objects.all(), required=False, allow_null=True
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
            "organization",
            "order_type",
            "order",
            "revenue_code",
            "customer",
            "worker",
            "commodity",
            "user",
        )


class BillingHistorySerializer(GenericSerializer):
    """A serializer for the `BillingHistory` model.

    A serializer class for the BillingHistory Model. This serializer is used
    to convert the BillingHistory model instances into a Python dictionary
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

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )
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
        queryset=Commodity.objects.all(), required=False, allow_null=True
    )
    user = serializers.PrimaryKeyRelatedField(
        queryset=User.objects.all(), required=False, allow_null=True
    )

    class Meta:
        """Metaclass for BillingHistorySerializer

        Attributes:
            model (models.BillingHistory): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.BillingHistory
        extra_fields = (
            "organization",
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

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )

    class Meta:
        """
        A class representing the metadata for the `ChargeTypeSerializer` class.
        """

        model = models.ChargeType
        extra_fields = ("organization",)


class AccessorialChargeSerializer(GenericSerializer):
    """
    A serializer for the `AccessorialCharge` model.

    This serializer converts instances of the `AccessorialCharge` model into JSON
    or other data formats, and vice versa. It uses the specified fields
    (code, is_detention, charge_amount, and method) to create the serialized
    representation of the `AccessorialCharge` model.
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )

    class Meta:
        """k
        A class representing the metadata for the `AccessorialChargeSerializer` class.
        """

        model = models.AccessorialCharge
        extra_fields = ("organization",)


class DocumentClassificationSerializer(GenericSerializer):
    """
    A serializer for the `DocumentClassification` model.

    This serializer converts instances of the `DocumentClassification` model into JSON or other data
    formats, and vice versa. It uses the specified fields (id, name, and description) to create the
    serialized representation of the `DocumentClassification` model.
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )

    class Meta:
        """
        A class representing the metadata for the `DocumentClassificationSerializer` class.
        """

        model = models.DocumentClassification
        extra_fields = ("organization",)
