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

from typing import Any

from customer import models
from customer.selectors import (calculate_customer_total_miles,
                                get_customer_credit_balance,
                                get_customer_on_time_performance_diff,
                                get_customer_order_diff,
                                get_customer_revenue_diff,
                                get_customer_shipment_metrics)
from utils.serializers import GenericSerializer


class CustomerContactSerializer(GenericSerializer):
    """A serializer for the CustomerContact model.

    The serializer provides default operations for creating, updating, and deleting
    customer contacts, as well as listing and retrieving customer contacts.
    It uses the `CustomerContact` model to convert the customer contact instances to
    and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this serializer.
    Filtering is also available, with the ability to filter by customer ID, name, and
    code.
    """

    class Meta:
        """
        A class representing the metadata for the `CustomerContactSerializer` class.
        """

        model = models.CustomerContact


class CustomerEmailProfileSerializer(GenericSerializer):
    """Serializer for the CustomerEmailProfile model.

    This serializer converts the CustomerEmailProfile model into a format that
    can be easily converted to and from JSON, and allows for easy validation
    of the data.
    """

    class Meta:
        """
        A class representing the metadata for the `CustomerEmailProfileSerializer` class.
        """

        model = models.CustomerEmailProfile


class CustomerFuelTableDetailSerializer(GenericSerializer):
    """A serializer for the CustomerFuelTableDetail model.

    The serializer provides default operations for creating, updating, and deleting
    customer fuel table details, as well as listing and retrieving customer fuel table
    details. It uses the `CustomerFuelTableDetail` model.
    """

    class Meta:
        """
        A class representing the metadata for the `CustomerFuelTableDetailSerializer` class.
        """

        model = models.CustomerFuelTableDetail


class CustomerFuelTableSerializer(GenericSerializer):
    """A serializer for the CustomerFuelTable model.

    The serializer provides default operations for creating, updating, and deleting
    customer fuel tables, as well as listing and retrieving customer fuel tables.
    It uses the `CustomerFuelTable` model to convert the customer fuel table instances
    to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this serializer.
    Filtering is also available, with the ability to filter by customer ID, name, and
    code.
    """

    customer_fuel_table_details = CustomerFuelTableDetailSerializer(
        many=True,
        required=False,
    )

    class Meta:
        """
        A class representing the metadata for the `CustomerFuelTableSerializer` class.
        """

        model = models.CustomerFuelTable
        extra_fields = ("customer_fuel_table_details",)

    def update(  # type: ignore
        self, instance: models.CustomerFuelTable, validated_data: Any
    ) -> models.CustomerFuelTable:
        """Update a customer fuel table.

        Args:
            instance (models.CustomerFuelTable): The customer fuel table to update.
            validated_data (Any): The validated data to update the customer fuel table with.

        Returns:
            models.CustomerFuelTable: The updated customer fuel table.
        """

        organization = super().get_organization

        customer_fuel_table_details = validated_data.pop(
            "customer_fuel_table_details",
            {},
        )

        models.CustomerFuelTable.objects.filter(
            id=instance.id, organization=organization
        ).update(**validated_data)

        if customer_fuel_table_details:
            for customer_fuel_table_detail in customer_fuel_table_details:
                customer_fuel_table_detail["customer_fuel_table"] = instance
                customer_fuel_table_detail["organization"] = organization
                CustomerFuelTableDetailSerializer(
                    data=customer_fuel_table_detail
                ).is_valid(raise_exception=True)

            models.CustomerFuelTableDetail.objects.filter(
                customer_fuel_table=instance
            ).delete()
            models.CustomerFuelTableDetail.objects.bulk_create(
                [
                    models.CustomerFuelTableDetail(**customer_fuel_table_detail)
                    for customer_fuel_table_detail in customer_fuel_table_details
                ]
            )

        return instance


class CustomerRuleProfileSerializer(GenericSerializer):
    """A serializer for the CustomerRuleProfile model.

    The serializer provides default operations for creating, updating, and deleting
    customer rule profiles, as well as listing and retrieving customer rule profiles.
    It uses the `CustomerRuleProfile` model to convert the customer rule profile
    instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this serializer.
    Filtering is also available, with the ability to filter by customer ID, name, and
    code.
    """

    class Meta:
        """
        A class representing the metadata for the `CustomerRuleProfileSerializer` class.
        """

        model = models.CustomerRuleProfile


class CustomerBillingProfileSerializer(GenericSerializer):
    """A serializer for the CustomerBillingProfile model.

    The serializer provides default operations for creating, updating, and deleting
    customer billing profiles, as well as listing and retrieving customer billing
    profiles.
    It uses the `CustomerBillingProfile` model to convert the customer billing profile
    instances to and from JSON-formatted data.

    Only authenticated users are allowed to access the views provided by this serializer.
    Filtering is also available, with the ability to filter by customer ID, name, and
    code.
    """

    class Meta:
        """
        A class representing the metadata for the `CustomerBillingProfileSerializer` class.
        """

        model = models.CustomerBillingProfile


class CustomerSerializer(GenericSerializer):
    """A serializer for the `Customer` model.

    This serializer converts instances of the `Customer` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, description, and code) to
    create the serialized representation of the `Customer` model.
    """

    class Meta:
        """
        A class representing the metadata for the `CustomerSerializer` class.
        """

        model = models.Customer

    def to_representation(self, instance: models.Customer) -> dict[str, Any]:
        """Transforms the instance's data into a dictionary.

        This method retrieve the data from an instance of `Customer`, and then
        transforms it into a dictionary format suitable for serialization.
        It also adds a new field `full_address` which extracts the address from
        the `get_address_combination` property.

        Args:
            instance (models.Customer): The `Customer` model instance that will be serialized.

        Returns:
            dict: A dictionary containing the serialized data from th `Customer` model instance,
                including the `full_address` field.
        """

        data = super().to_representation(instance)
        data["full_address"] = instance.get_address_combination

        if self.context["request"].query_params.get("expand_metrics", False):
            data["total_order_metrics"] = get_customer_order_diff(
                customer_id=instance.id
            )
            data["total_revenue_metrics"] = get_customer_revenue_diff(
                customer_id=instance.id
            )
            data["on_time_performance"] = get_customer_on_time_performance_diff(
                customer_id=instance.id
            )
            data["total_mileage_metrics"] = calculate_customer_total_miles(
                customer_id=instance.id
            )
            data["customer_shipment_metrics"] = get_customer_shipment_metrics(
                customer_id=instance.id
            )
            data["credit_balance"] = get_customer_credit_balance(
                customer_id=instance.id
            )
        return data
