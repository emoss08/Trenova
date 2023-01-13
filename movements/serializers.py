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

from equipment.models import Equipment
from movements import models
from order.models import Order
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
        equipment (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the equipment of the movement.
        primary_worker (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the primary worker of the movement.
        secondary_worker (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the secondary worker of the movement.
    """

    order = serializers.PrimaryKeyRelatedField(
        queryset=Order.objects.all(),
    )
    equipment = serializers.PrimaryKeyRelatedField(
        queryset=Equipment.objects.all(),
        allow_null=True,
    )
    primary_worker = serializers.PrimaryKeyRelatedField(
        queryset=Worker.objects.all(),
        allow_null=True,
    )
    secondary_worker = serializers.PrimaryKeyRelatedField(
        queryset=Worker.objects.all(),
        allow_null=True,
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
            "order",
            "equipment",
            "primary_worker",
            "secondary_worker",
        )
