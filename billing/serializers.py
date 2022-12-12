

from billing import models
from rest_framework import serializers

class ChargeTypeSerializer(serializers.ModelSerializer):
    """
    A serializer for the `ChargeType` model.

    This serializer converts instances of the `ChargeType` model into JSON or other data formats,
    and vice versa. It uses the specified fields (id, name, and description) to create the serialized
    representation of the `ChargeType` model.
    """

    class Meta:
        """
        A class representing the metadata for the `ChargeTypeSerializer` class.
        """

        model = models.ChargeType
        fields = ("id", "name", "description")

class AccessorialChargeSerializer(serializers.ModelSerializer):
    """
    A serializer for the `AccessorialCharge` model.

    This serializer converts instances of the `AccessorialCharge` model into JSON or other data formats,
    and vice versa. It uses the specified fields (code, is_detention, charge_amount, and method) to
    create the serialized representation of the `AccessorialCharge` model.
    """

    class Meta:
        """
        A class representing the metadata for the `AccessorialChargeSerializer` class.
        """
        model = models.AccessorialCharge
        fields = ("code", "is_detention", "charge_amount", "method")


class DocumentClassificationSerializer(serializers.ModelSerializer):
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
        fields = ("id", "name", "description")