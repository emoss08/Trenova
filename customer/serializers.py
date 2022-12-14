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

from typing import Any

from django.db import transaction
from rest_framework import serializers

from accounts.models import Token
from customer import models
from utils.serializers import GenericSerializer


class CustomerFuelTableDetailSerializer(GenericSerializer):
    """A serializer for the CustomerFuelTableDetail model.

    The serializer provides default operations for creating, updating, and deleting
    customer fuel table details, as well as listing and retrieving customer fuel table
    details. It uses the `CustomerFuelTableDetail` model.
    """

    method = serializers.ChoiceField(
        choices=models.FuelMethodChoices.choices,
    )

    class Meta:
        """
        A class representing the metadata for the `CustomerFuelTableDetailSerializer` class.
        """

        model = models.CustomerFuelTableDetail
        fields = (
            "id",
            "amount",
            "method",
            "start_price",
            "percentage",
            "created",
            "modified",
        )
        read_only_fields = ("id", "created", "modified")
        extra_kwargs = {
            "customer_fuel_table": {"required": True},
            "method": {"required": True},
        }


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
        fields = (
            "id",
            "organization",
            "name",
            "description",
            "created",
            "modified",
            "customer_fuel_table_details",
        )
        read_only_fields = (
            "organization",
            "id",
            "created",
            "modified",
        )

    @transaction.atomic
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

        if self.context["request"].user.is_authenticated:
            organization = self.context["request"].user.organization
        else:
            token = (
                self.context["request"].META.get("HTTP_AUTHORIZATION", "").split(" ")[1]
            )
            organization = Token.objects.get(key=token).user.organization

        customer_fuel_table_details = validated_data.pop(
            "customer_fuel_table_details",
            None,
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


class CustomerSerializer(GenericSerializer):
    """A serializer for the `Customer` model.

    This serializer converts instances of the `Customer` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, description, and code) to create the serialized
    representation of the `Customer` model.
    """

    class Meta:
        """
        A class representing the metadata for the `CustomerSerializer` class.
        """

        model = models.Customer
        fields = (
            "id",
            "organization",
            "is_active",
            "code",
            "name",
            "address_line_1",
            "address_line_2",
            "city",
            "state",
            "zip_code",
            "created",
            "modified",
        )
        read_only_fields = (
            "id",
            "organization",
            "created",
            "modified",
        )
