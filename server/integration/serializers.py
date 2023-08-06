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

from integration import models
from utils.serializers import GenericSerializer


class IntegrationVendorSerializer(GenericSerializer):
    """A serializer class for the `IntegrationVendor` model

    The `IntegrationVendorSerializer` class provides default operations for
    creating, updating and deleting `IntegrationVendors`, as well as listing
    and retrieving them.
    """

    class Meta:
        """
        Metaclass for IntegrationVendorSerializer
        """

        model = models.IntegrationVendor


class IntegrationSerializer(GenericSerializer):
    """A serializer class for the Integration model

    The `IntegrationSerializer` class provides default operations
    for creating, update and deleting Integration, as well as
    listing and retrieving them.
    """

    class Meta:
        """
        A class representing the metadata for the `IntegrationSerializer`
        class.
        """

        model = models.Integration


class GoogleAPISerializer(GenericSerializer):
    """
    A serializer for the `GoogleAPI` model.

    This serializer converts instances of the `GoogleAPI` model into JSON or other data
    formats, and vice versa. It uses the specified fields (name, description, and code)
    to create the serialized representation of the `GoogleAPI` model.
    """

    class Meta:
        """
        A class representing the metadata for the `GoogleAPISerializer` class.
        """

        model = models.GoogleAPI
