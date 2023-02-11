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

from billing.models import AccessorialCharge
from commodities.models import Commodity
from customer.models import Customer
from dispatch import models
from equipment.models import EquipmentType
from location.models import Location
from order.models import OrderType
from utils.serializers import GenericSerializer


class CommentTypeSerializer(GenericSerializer):
    """A serializer for the CommentType model.

    The serializer provides default operations for creating, updating, and deleting
    comment types, as well as listing and retrieving comment types.It uses the
    `CommentType` model to convert the comment type instances to and from
    JSON-formatted data.

    Only authenticated users are allowed to access the view provided by this serializer.
    Filtering is also available, with the ability to filter by ID, and name.
    """

    class Meta:
        """
        A class representing the metadata for the `CommentTypeSerializer` class.
        """

        model = models.CommentType


class DelayCodeSerializer(GenericSerializer):
    """A serializer for the DelayCode model.

    The serializer provides default operations for creating, updating, and deleting
    delay codes, as well as listing and retrieving delay codes.It uses the
    `DelayCode` model to convert the comment type instances to and from
    JSON-formatted data.

    Only authenticated users are allowed to access the view provided by this serializer.
    Filtering is also available, with the ability to filter by ID, and name.
    """

    class Meta:
        """
        A class representing the metadata for the `DelayCodeSerializer` class.
        """

        model = models.DelayCode


class FleetCodeSerializer(GenericSerializer):
    """A serializer for the FleetCode model.

    The serializer provides default operations for creating, updating, and deleting
    Fleet Codes, as well as listing and retrieving fleet codes.It uses the
    `FleetCode` model to convert the comment type instances to and from
    JSON-formatted data.

    Only authenticated users are allowed to access the view provided by this serializer.
    Filtering is also available, with the ability to filter by ID, and name.
    """

    is_active = serializers.BooleanField(default=True)

    class Meta:
        """
        A class representing the metadata for the `FleetCodeSerializer` class.
        """

        model = models.FleetCode
        extra_fields = ("is_active",)


class DispatchControlSerializer(GenericSerializer):
    """A serializer for the DispatchControl model.

    The serializer provides default operations for creating, updating, and deleting
    Dispatch Control, as well as listing and retrieving Dispatch Control. It uses the
    `DispatchControl` model to convert the dispatch control instances to and from
    JSON-formatted data.

    Only authenticated users are allowed to access the view provided by this serializer.
    Filtering is also available, with the ability to filter by ID, and name.
    """

    class Meta:
        """
        A class representing the metadata for the `DispatchControlSerializer` class.
        """

        model = models.DispatchControl


class RateSerializer(GenericSerializer):
    """Serializer class for the Rate model.

    This class extends the `GenericSerializer` class and serializes the `Rate` model,
    including fields for the related `Customer`, `Commodity`, `OrderType`, and `EquipmentType` models.

    Attributes:
        customer (serializers.PrimaryKeyRelatedField): The related `Customer` model, with a queryset of all `Customer` objects and
        the option to allow `None` values.
        commodity (serializers.PrimaryKeyRelatedField): The related `Commodity` model, with a queryset of all `Commodity` objects and
        the option to allow `None` values.
        order_type (serializers.PrimaryKeyRelatedField): The related `OrderType` model, with a queryset of all `OrderType` objects and
        the option to allow `None` values.
        equipment_type (serializers.PrimaryKeyRelatedField): The related `EquipmentType` model, with a queryset of all `EquipmentType`
        objects and the option to allow `None` values.
    """
    customer = serializers.PrimaryKeyRelatedField(
        queryset=Customer.objects.all(), required=False, allow_null=True
    )
    commodity = serializers.PrimaryKeyRelatedField(
        queryset=Commodity.objects.all(), required=False, allow_null=True
    )
    order_type = serializers.PrimaryKeyRelatedField(
        queryset=OrderType.objects.all(), required=False, allow_null=True
    )
    equipment_type = serializers.PrimaryKeyRelatedField(
        queryset=EquipmentType.objects.all(), required=False, allow_null=True
    )

    class Meta:
        """
        A class representing the metadata for the `RateSerializer` class.
        """
        model = models.Rate
        extra_fields = (
            "customer",
            "commodity",
            "order_type",
            "equipment_type",
        )


class RateTableSerializer(GenericSerializer):
    """Serializer class for the RateTable model.

    This class extends the `GenericSerializer` class and serializes the `RateTable` model,
    including fields for the related `Rate` and `Location` models.

    Attributes:
        rate (serializers.PrimaryKeyRelatedField): The related `Rate` model, with a queryset of all `Rate` objects.
        origin_location (serializers.PrimaryKeyRelatedField): The related `Location` model for the origin, with a
        queryset of all `Location` objects and the option to allow `None` values.
        destination_location (serializers.PrimaryKeyRelatedField): The related `Location` model for the destination,
        with a queryset of all `Location` objects and the option to allow `None` values.
    """
    rate = serializers.PrimaryKeyRelatedField(queryset=models.Rate.objects.all())
    origin_location = serializers.PrimaryKeyRelatedField(
        queryset=Location.objects.all(), required=False, allow_null=True
    )
    destination_location = serializers.PrimaryKeyRelatedField(
        queryset=Location.objects.all(), required=False, allow_null=True
    )

    class Meta:
        """
        A class representing the metadata for the `RateTableSerializer` class.
        """
        model = models.RateTable
        extra_fields = (
            "rate",
            "origin_location",
            "destination_location",
        )


class RateBillingTableSerializer(GenericSerializer):
    """Serializer class for the RateBillingTable model.

    This class extends the `GenericSerializer` class and serializes the `RateBillingTable` model,
    including fields for the related `Rate` and `AccessorialCharge` models.

    Attributes:
        rate (serializers.PrimaryKeyRelatedField): The related `Rate` model, with a queryset of all `Rate` objects.
        charge_code (serializers.PrimaryKeyRelatedField): The related `AccessorialCharge` model, with a queryset of
        all `AccessorialCharge` objects.
    """
    rate = serializers.PrimaryKeyRelatedField(queryset=models.Rate.objects.all())
    charge_code = serializers.PrimaryKeyRelatedField(
        queryset=AccessorialCharge.objects.all()
    )

    class Meta:
        """
        A class representing the metadata for the `RateBillingTableSerializer` class.
        """
        model = models.RateBillingTable
