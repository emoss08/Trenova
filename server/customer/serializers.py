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

from rest_framework import serializers

from customer import helpers, models
from utils.serializers import GenericSerializer


class DeliverySlotSerializer(GenericSerializer):
    """A serializer for the DeliverySlot model.

    The serializer provides default operations for creating, updating, and deleting
    delivery slots, as well as listing and retrieving delivery slot.
    It uses the `DeliverySlot` model to convert the delivery slot instances to
    and from JSON-formatted data.
    """

    id = serializers.UUIDField(required=False, allow_null=True)
    location_name = serializers.CharField(required=False, allow_null=True)

    class Meta:
        """
        A class representing the metadata for the `DeliverySlotSerializer` class.
        """

        model = models.DeliverySlot
        extra_fields = ("location_name",)
        extra_read_only_fields = ("customer",)


class CustomerContactSerializer(GenericSerializer):
    """A serializer for the CustomerContact model.

    The serializer provides default operations for creating, updating, and deleting
    customer contacts, as well as listing and retrieving customer contacts.
    It uses the `CustomerContact` model to convert the customer contact instances to
    and from JSON-formatted data.
    """

    id = serializers.UUIDField(required=False, allow_null=True)

    class Meta:
        """
        A class representing the metadata for the `CustomerContactSerializer` class.
        """

        model = models.CustomerContact
        extra_read_only_fields = ("customer",)


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

    def validate_name(self, value: str) -> str:
        """Validate the `name` field of the Customer Fuel Table model.

        This method validates the `name` field of the Customer Fuel Table model.
        It checks if the customer fuel table with the given name already exists in the organization.
        If the customer fuel table exists, it raises a validation error.

        Args:
            value (str): The value of the `name` field.

        Returns:
            str: The value of the `name` field.

        Raises:
            serializers.ValidationError: If the customer fuel table with the given name already exists
            in the organization.
        """
        organization = super().get_organization

        queryset = models.CustomerFuelTable.objects.filter(
            organization=organization,
            name__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.CustomerFuelTable):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Customer Fuel Table with this `name` already exists. Please try again."
            )

        return value

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


class CustomerSerializer(GenericSerializer):
    """
    A serializer for the Customer model.
    """

    email_profile = CustomerEmailProfileSerializer(required=False)
    rule_profile = CustomerRuleProfileSerializer(required=False)
    delivery_slots = DeliverySlotSerializer(many=True, required=False)
    contacts = CustomerContactSerializer(many=True, required=False)
    last_ship_date = serializers.DateField(required=False, allow_null=True)
    last_bill_date = serializers.DateField(required=False, allow_null=True)
    total_shipments = serializers.IntegerField(required=False, allow_null=True)

    class Meta:
        """
        A class representing the metadata for the `CustomerSerializer` class.
        """

        model = models.Customer
        extra_fields = (
            "email_profile",
            "rule_profile",
            "delivery_slots",
            "contacts",
            "last_ship_date",
            "last_bill_date",
            "total_shipments",
        )

    def validate_code(self, value: str) -> str:
        """Validate the `code` field of the Customer model.

        This method validates the `code` field of the Customer model.
        It checks if the customer with the given code already exists in the organization.
        If the customer exists, it raises a validation error.

        Args:
            value (str): The value of the `code` field.

        Returns:
            str: The value of the `code` field.

        Raises:
            serializers.ValidationError: If the customer with the given code already exists in the organization.
        """
        organization = super().get_organization

        queryset = models.Customer.objects.filter(
            organization=organization,
            code__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance and isinstance(self.instance, models.Customer):
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Customer with this `code` already exists. Please try again."
            )

        return value

    def create(self, validated_data: Any) -> models.Customer:
        """Create a new instance of the Customer model with given validated data.

        This executes the creation of new customer, attaches the customer to the business unit and organization
        associated with the request. It updates the email profile, rule profile, delivery slots, and customer contacts
        associated with the Customer.

        Args:
            validated_data (Any): Data validated through serializer for creating a customer.

        Returns:
           models.Customer: Newly created Customer instance.
        """

        # Get the organization of the user from the request.
        organization = super().get_organization

        # Get the business unit of the user from the request.
        business_unit = super().get_business_unit

        # Popped data (email_profile, rule_profile, delivery_slots, customer_contacts)
        email_profile_data = validated_data.pop("email_profile", None)
        rule_profile_data = validated_data.pop("rule_profile", None)
        delivery_slots_data = validated_data.pop("delivery_slots", [])
        customer_contacts_data = validated_data.pop("contacts", [])

        # Create the customer
        customer = models.Customer.objects.create(
            organization=organization,
            business_unit=business_unit,
            **validated_data,
        )

        # Create or update the email profile
        if email_profile_data:
            helpers.create_or_update_email_profile(
                customer=customer,
                email_profile_data=email_profile_data,
                organization=organization,
                business_unit=business_unit,
            )

        # Create or update the rule profile
        if rule_profile_data:
            helpers.create_or_update_rule_profile(
                customer=customer,
                rule_profile_data=rule_profile_data,
                organization=organization,
                business_unit=business_unit,
            )

        # Create or update the delivery slots
        if delivery_slots_data:
            helpers.create_or_update_delivery_slots(
                customer=customer,
                delivery_slots_data=delivery_slots_data,
                organization=organization,
                business_unit=business_unit,
            )

        # Create or update the customer contacts
        if customer_contacts_data:
            helpers.create_or_update_customer_contacts(
                customer=customer,
                customer_contacts_data=customer_contacts_data,
                organization=organization,
                business_unit=business_unit,
            )

        return customer

    def update(self, instance: models.Customer, validated_data: Any) -> models.Customer:  # type: ignore
        """Update an existing instance of the Customer model with given validated data.

        This method updates an existing customer, based on the data provided in the request.
        It updates the email profile, rule profile, delivery slots, and customer contacts associated with the Customer.

        Args:
            instance (models.Customer): Existing instance of Customer model to update.
            validated_data (Any): Data validated through serializer for updating a customer profile.

        Returns:
            models.Customer: Updated Customer instance.
        """

        # Get the organization of the user from the request.
        organization = super().get_organization

        # Get the business unit of the user from the request.
        business_unit = super().get_business_unit

        # Popped data (email_profile, rule_profile, delivery_slots, customer_contacts)
        email_profile_data = validated_data.pop("email_profile", None)
        rule_profile_data = validated_data.pop("rule_profile", None)
        delivery_slots_data = validated_data.pop("delivery_slots", [])
        customer_contacts_data = validated_data.pop("contacts", [])

        # Create or update the email profile
        if email_profile_data:
            helpers.create_or_update_email_profile(
                customer=instance,
                email_profile_data=email_profile_data,
                organization=organization,
                business_unit=business_unit,
            )

        # Create or update the rule profile
        if rule_profile_data:
            helpers.create_or_update_rule_profile(
                customer=instance,
                rule_profile_data=rule_profile_data,
                organization=organization,
                business_unit=business_unit,
            )

        # Create or update the delivery slots
        if delivery_slots_data:
            helpers.create_or_update_delivery_slots(
                customer=instance,
                delivery_slots_data=delivery_slots_data,
                organization=organization,
                business_unit=business_unit,
            )

        # Create or update the customer contacts
        if customer_contacts_data:
            helpers.create_or_update_customer_contacts(
                customer=instance,
                customer_contacts_data=customer_contacts_data,
                organization=organization,
                business_unit=business_unit,
            )

        # Update the customer
        for attr, value in validated_data.items():
            setattr(instance, attr, value)
        instance.save()

        return instance
