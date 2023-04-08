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
