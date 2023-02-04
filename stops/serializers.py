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

from accounts.models import User
from dispatch.models import CommentType, DelayCode
from movements.models import Movement
from stops import models
from utils.serializers import GenericSerializer


class QualifierCodeSerializer(GenericSerializer):
    """A serializer for the `QualifierCode` model.

    A serializer class for the QualifierCode Model. This serializer is used
    to convert the QualifierCode model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for the `QualifierCodeSerializer` class

        Attributes:
            model (models.QualifierCode): The model that the serializer is for.
        """

        model = models.QualifierCode


class StopCommentSerializer(GenericSerializer):
    """A serializer for the `StopComment` model.

    A serializer class for the StopComment Model. This serializer is used
    to convert the StopComment model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.

    Attributes:
        comment_type (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the comment type of the stop comment.
        qualifier_code (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the qualifier code of the stop comment.
        entered_by (serializers.PrimaryKeyRelatedField): A primary key related field that
        determines the user who entered the stop comment.
    """

    comment_type = serializers.PrimaryKeyRelatedField(
        queryset=CommentType.objects.all(),
    )
    qualifier_code = serializers.PrimaryKeyRelatedField(
        queryset=models.QualifierCode.objects.all(),
    )
    entered_by = serializers.PrimaryKeyRelatedField(
        queryset=User.objects.all(),
    )

    class Meta:
        """Metaclass for the `StopCommentSerializer` class

        Attributes:
            model (models.Order): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.StopComment
        extra_fields = (
            "comment_type",
            "qualifier_code",
            "entered_by",
        )


class StopSerializer(GenericSerializer):
    """A serializer for the `Stop` model.

    A serializer class for the Stop Model. This serializer is used
    to convert the Stop model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    stop_comments = serializers.PrimaryKeyRelatedField(
        queryset=models.StopComment.objects.all(),
        allow_null=True,
        required=False,
    )

    class Meta:
        """Metaclass for the `StopSerializer` class

        Attributes:
            model (models.Stop): The model that the serializer is for.
        """

        model = models.Stop
        extra_fields = ("stop_comments",)


class ServiceIncidentSerializer(GenericSerializer):
    """A serializer for the `ServiceIncident` model.

    A serializer class for the ServiceIncident Model. This serializer is used
    to convert the ServiceIncident model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    movement = serializers.PrimaryKeyRelatedField(
        queryset=Movement.objects.all(),
    )
    stop = serializers.PrimaryKeyRelatedField(
        queryset=models.Stop.objects.all(),
    )
    delay_code = serializers.PrimaryKeyRelatedField(
        queryset=DelayCode.objects.all(),
        allow_null=True,
        required=False,
    )

    class Meta:
        """Metaclass for the `ServiceIncidentSerializer` class

        Attributes:
            model (models.ServiceIncident): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.ServiceIncident
        extra_fields = (
            "movement",
            "stop",
            "delay_code",
        )
