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
import typing

from rest_framework import serializers

from dispatch import models, helpers
from utils.helpers import convert_to_date
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


class RateBillingTableSerializer(GenericSerializer):
    """Serializer class for the RateBillingTable model.

    This class extends the `GenericSerializer` class and serializes the `RateBillingTable` model,
    including fields for the related `Rate` and `AccessorialCharge` models.
    """

    id = serializers.UUIDField(required=False, allow_null=True)

    class Meta:
        """
        A class representing the metadata for the `RateBillingTableSerializer` class.
        """

        model = models.RateBillingTable
        extra_read_only_fields = ("rate",)


class RateSerializer(GenericSerializer):
    """Serializer class for the Rate model.

    This class extends the `GenericSerializer` class and serializes the `Rate` model,
    including fields for the related `Customer`, `Commodity`, `OrderType`, and `EquipmentType` models.
    """

    rate_billing_tables = RateBillingTableSerializer(many=True, required=False)

    class Meta:
        """
        A class representing the metadata for the `RateSerializer` class.
        """

        model = models.Rate
        extra_fields = ("rate_billing_tables",)

    def to_internal_value(self, data: typing.Any) -> typing.Any:
        """Convert the input data into the internal (deserialized) data format.

        This function runs over the input `data` dict, checks for the presence of the
        "expiration_date" and "effective_date" fields, and if present, uses the `convert_to_date`
        function to convert these fields into date format. The converted data is then passed to
        the `to_internal_value` function of the superclass for further processing.

        Args:
            data (typing.Any): The input data to be converted.

        Returns:
            typing.Any: The converted data.

        Raises:
            ValidationError: If one of the date strings cannot be converted to a date.
        """
        for field in ["expiration_date", "effective_date"]:
            if date_str := data.get(field):
                data[field] = convert_to_date(date_str)
        return super().to_internal_value(data)

    def create(self, validated_data: typing.Any) -> models.Rate:
        """Creates a new `Rate` instance using the provided validated data.

        This function retrieves the organization and business unit from the user's request
        data and creates a new `Rate` instance using these along with the other validated
        data provided. If rate billing table data is included in the input data, the
        function creates respective `RateBillingTable` instances for the newly created `Rate`.

        Args:
            validated_data (typing.Any): A data structure containing sanitized user input data.

        Returns:
            models.Rate: The newly created `Rate` instance.
        """

        # Get the organization of the user from the request.
        organization = super().get_organization

        # Get the business unit of the user from the request.
        business_unit = super().get_business_unit

        # Pop rate billing table data.
        rate_billing_table_data = validated_data.pop("rate_billing_tables", [])

        # Create the rate
        rate = models.Rate.objects.create(
            organization=organization,
            business_unit=business_unit,
            **validated_data,
        )

        # Create the rate billing tables
        if rate_billing_table_data:
            helpers.create_or_update_rate_billing_table(
                organization=organization,
                business_unit=business_unit,
                rate=rate,
                rate_billing_tables_data=rate_billing_table_data,
            )

        return rate

    def update(self, instance: models.Rate, validated_data: typing.Any) -> models.Rate:  # type: ignore
        """Updates an existing `Rate` instance using the provided validated data.

        This function retrieves the organization and business unit from the user's request.
        It then updates the given `Rate` instance with the new data provided. If rate billing table
        data is provided, this function also updates the corresponding `RateBillingTable` instances
        to match the new data.

        Args:
            instance (models.Rate): The `Rate` instance that is to be updated.
            validated_data (typing.Any): A data structure containing sanitized user input data.

        Returns:
            models.Rate: The updated `Rate` instance.
        """

        # Get the organization of the user from the request.
        organization = super().get_organization

        # Get the business unit of the user from the request.
        business_unit = super().get_business_unit

        if rate_billing_table_data := validated_data.pop("rate_billing_tables", []):
            helpers.create_or_update_rate_billing_table(
                organization=organization,
                business_unit=business_unit,
                rate=instance,
                rate_billing_tables_data=rate_billing_table_data,
            )

        for attr, value in validated_data.items():
            setattr(instance, attr, value)
        instance.save()

        return instance


class FeasibilityToolControlSerializer(GenericSerializer):
    """A serializer for the `FeasibilityToolControl` model.

    A serializer class for the FeasibilityToolControl model. This serializer is used
    to convert the FeasibilityToolControl model instances into a Python dictionary
    format that can be rendered into a JSON response. It also defines the fields
    that should be included in the serialized representation of the model
    """

    class Meta:
        """
        A class representing the metadata for the `FeasibilityToolControlSerializer` class.
        """

        model = models.FeasibilityToolControl
