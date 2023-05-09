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


from billing import models
from utils.serializers import GenericSerializer


class BillingControlSerializer(GenericSerializer):
    """A serializer for the `BillingControl` model.

    A serializer class for the BillingControl model. This serializer is used to convert BillingControl model
    instances into a Python dictionary format that can be rendered into a JSON response. It also defined the
    fields that should be included in the serialized representation of the model
    """

    class Meta:
        """
        Metaclass for the BillingControlSerializer

        Attributes:
            model (BillingControl): The model that the serializer is for.
        """

        model = models.BillingControl


class BillingTransferLogSerializer(GenericSerializer):
    """A serializer for the `BillingTransferLog` model.

    A serializer class for the BillingTransferLog Model. This serializer is used to convert the BillingTransferLog
    model instances into a Python dictionary format that can be rendered into a JSON response. It also defines
    the fields that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for BillingTransferLogSerializer

        Attributes:
            model (models.BillingTransferLog): The model that the serializer is for.
        """

        model = models.BillingTransferLog


class BillingQueueSerializer(GenericSerializer):
    """A serializer for the `BillingQueue` model.

    A serializer class for the BillingQueue Model. This serializer is used to convert the BillingQueue
    model instances into a Python dictionary format that can be rendered into a JSON response. It
    also defines the fields that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for BillingQueueSerializer

        Attributes:
            model (models.BillingQueue): The model that the serializer is for.
            extra_fields (tuple): A tuple of extra fields that should be included
            in the serialized representation of the model.
        """

        model = models.BillingQueue


class BillingHistorySerializer(GenericSerializer):
    """A serializer for the `BillingHistory` model.

    A serializer class for the BillingHistory Model. This serializer is used
    to convert the BillingHistory model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model.
    """

    class Meta:
        """Metaclass for BillingHistorySerializer

        Attributes:
            model (models.BillingHistory): The model that the serializer is for.
        """

        model = models.BillingHistory


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

    class Meta:
        """k
        A class representing the metadata for the `AccessorialChargeSerializer` class.
        """

        model = models.AccessorialCharge


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
