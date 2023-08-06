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


from dispatch import models
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

    class Meta:
        """
        A class representing the metadata for the `FleetCodeSerializer` class.
        """

        model = models.FleetCode


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
    """

    class Meta:
        """
        A class representing the metadata for the `RateSerializer` class.
        """

        model = models.Rate


class RateBillingTableSerializer(GenericSerializer):
    """Serializer class for the RateBillingTable model.

    This class extends the `GenericSerializer` class and serializes the `RateBillingTable` model,
    including fields for the related `Rate` and `AccessorialCharge` models.
    """

    class Meta:
        """
        A class representing the metadata for the `RateBillingTableSerializer` class.
        """

        model = models.RateBillingTable
