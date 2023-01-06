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

from billing import models
from utils.serializers import GenericSerializer


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
