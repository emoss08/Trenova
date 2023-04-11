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

from equipment.models import Tractor
from movements import models
from order.models import Order
from organization.models import Organization
from utils.serializers import GenericSerializer
from worker.models import Worker


class MovementSerializer(GenericSerializer):
    """A serializer for the `Movement` model.

    A serializer class for the Movement Model. This serializer is used
    to convert the Movement model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    Attributes:
        order (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the order of the movement.
        tractor (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the tractor of the movement.
        primary_worker (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the primary worker of the movement.
        secondary_worker (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the secondary worker of the movement.
    """

    organization = serializers.PrimaryKeyRelatedField(
        queryset=Organization.objects.all()
    )
    order = serializers.PrimaryKeyRelatedField(
        queryset=Order.objects.all(),
    )
    tractor = serializers.PrimaryKeyRelatedField(
        queryset=Tractor.objects.all(),
        allow_null=True,
        required=False,
    )
    primary_worker = serializers.PrimaryKeyRelatedField(
        queryset=Worker.objects.all(),
        allow_null=True,
        required=False,
    )
    secondary_worker = serializers.PrimaryKeyRelatedField(
        queryset=Worker.objects.all(),
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

        model = models.Movement
        extra_fields = (
            "organization",
            "order",
            "tractor",
            "primary_worker",
            "secondary_worker",
        )
        extra_read_only_fields = ("id", "ref_num")
