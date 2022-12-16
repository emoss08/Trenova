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

    is_active = serializers.BooleanField(default=True)
    is_payable_contact = serializers.BooleanField(default=True)

    class Meta:
        """
        A class representing the metadata for the `CustomerContactSerializer` class.
        """

        model = models.CustomerContact
        fields = (
            "id",
            "organization",
            "is_active",
            "name",
            "email",
            "title",
            "phone",
            "is_payable_contact",
            "created",
            "modified",
        )
        read_only_fields = (
            "organization",
            "id",
            "created",
            "modified",
        )


class CustomerEmailProfileSerializer(serializers.ModelSerializer):
    """Serializer for the CustomerEmailProfile model.

    This serializer converts the CustomerEmailProfile model into a format that
    can be easily converted to and from JSON, and allows for easy validation
    of the data.
    """

    read_receipt_to = serializers.EmailField(required=False)
    read_receipt = serializers.BooleanField(default=False)

    class Meta:
        """
        A class representing the metadata for the `CustomerEmailProfileSerializer` class.
        """

        model = models.CustomerEmailProfile
        fields = (
            "id",
            "name",
            "subject",
            "comment",
            "from_address",
            "blind_copy",
            "read_receipt",
            "read_receipt_to",
            "attachment_name",
            "created",
            "modified",
        )
        read_only_fields = (
            "organization",
            "id",
            "created",
            "modified",
        )


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

    document_class = serializers.StringRelatedField(many=True, read_only=True)

    class Meta:
        """
        A class representing the metadata for the `CustomerRuleProfileSerializer` class.
        """

        model = models.CustomerRuleProfile
        fields = (
            "id",
            "organization",
            "name",
            "created",
            "modified",
            "document_class",
        )
        read_only_fields = (
            "organization",
            "id",
            "created",
            "modified",
        )

    def create(self, validated_data: Any) -> models.CustomerRuleProfile:
        """Create a new CustomerRuleProfile instance.

        Args:
            validated_data (dict): A dictionary of validated data for the new
                CustomerRuleProfile instance. This data should include the
                'document_class' field, which is a list of IDs for the
                DocumentClassification objects associated with the new
                CustomerRuleProfile.

        Returns:
            CustomerRuleProfile: The newly created CustomerRuleProfile instance.
        """

        document_class_ids = validated_data.pop("document_class")

        customer_rule_profile = models.CustomerRuleProfile.objects.create(
            **validated_data
        )
        customer_rule_profile.document_class.set(document_class_ids)

        return customer_rule_profile

    def update(
            self, instance: models.CustomerRuleProfile, validated_data: Any
    ) -> models.CustomerRuleProfile:
        """Update an existing CustomerRuleProfile instance.

        Args:
            instance (CustomerRuleProfile): The CustomerRuleProfile instance to update.
            validated_data (dict): A dictionary of validated data for the updated
                CustomerRuleProfile instance. This data should include the
                'name' and 'document_class' fields, which are the updated values
                for the name and document classifications of the CustomerRuleProfile.

        Returns:
            CustomerRuleProfile: The updated CustomerRuleProfile instance.
        """

        document_class = validated_data.pop("document_class", None)

        instance.name = validated_data.get("name", instance.name)
        instance.save()

        if document_class:
            instance.document_class.set(document_class)
            instance.save()

        return instance


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

    is_active = serializers.BooleanField(default=True)
    email_profile = CustomerEmailProfileSerializer(required=False)
    rule_profile = CustomerRuleProfileSerializer(required=False)

    class Meta:
        """
        A class representing the metadata for the `CustomerBillingProfileSerializer` class.
        """

        model = models.CustomerBillingProfile
        fields = (
            "id",
            "organization",
            "is_active",
            "email_profile",
            "rule_profile",
            "created",
            "modified",
        )
        read_only_fields = (
            "organization",
            "id",
            "created",
            "modified",
        )

    def create(self, validated_data: Any):
        """Create a new CustomerBillingProfile instance.

        Args:
            validated_data (dict): A dictionary of validated data for the new
                CustomerBillingProfile instance. This data should include the
                'email_profile' and 'rule_profile' fields, which are the
                CustomerEmailProfile and CustomerRuleProfile instances
                associated with the new CustomerBillingProfile.

        Returns:
            CustomerBillingProfile: The newly created CustomerBillingProfile instance.
        """

        email_profile = validated_data.pop("email_profile", None)
        rule_profile = validated_data.pop("rule_profile", None)

        customer_billing_profile = models.CustomerBillingProfile.objects.create(
            **validated_data
        )

        if email_profile:
            customer_billing_profile.email_profile = (
                CustomerEmailProfileSerializer(data=email_profile)
                .is_valid(raise_exception=True)
                .save()
            )

        if rule_profile:
            customer_billing_profile.rule_profile = (
                CustomerRuleProfileSerializer(data=rule_profile)
                .is_valid(raise_exception=True)
                .save()
            )

        customer_billing_profile.save()

        return customer_billing_profile

    def update(self, validated_data: Any):
        """Update an existing CustomerBillingProfile instance.

        Args:
            validated_data (dict): A dictionary of validated data for the updated
                CustomerBillingProfile instance. This data should include the
                'email_profile' and 'rule_profile' fields, which are the
                updated values for the CustomerEmailProfile and CustomerRuleProfile
                instances associated with the CustomerBillingProfile.

        Returns:
            CustomerBillingProfile: The updated CustomerBillingProfile instance.
        """

        email_profile = validated_data.pop("email_profile", None)
        rule_profile = validated_data.pop("rule_profile", None)

        instance = models.CustomerBillingProfile.objects.get(
            id=validated_data["id"], organization=validated_data["organization"]
        )

        instance.is_active = validated_data.get("is_active", instance.is_active)
        instance.save()

        if email_profile:
            instance.email_profile = (
                CustomerEmailProfileSerializer(data=email_profile)
                .is_valid(raise_exception=True)
                .save()
            )

        if rule_profile:
            instance.rule_profile = (
                CustomerRuleProfileSerializer(data=rule_profile)
                .is_valid(raise_exception=True)
                .save()
            )

        instance.save()

        return instance


class CustomerSerializer(GenericSerializer):
    """A serializer for the `Customer` model.

    This serializer converts instances of the `Customer` model into JSON or other data formats,
    and vice versa. It uses the specified fields (name, description, and code) to
    create the serialized representation of the `Customer` model.
    """

    billing_profile = CustomerBillingProfileSerializer(required=False)
    contacts = CustomerContactSerializer(many=True, required=False)

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
            "billing_profile",
            "contacts",
        )
        read_only_fields = (
            "id",
            "organization",
            "created",
            "modified",
        )

    def create(self, validated_data: Any) -> models.Customer:
        """Create a new Customer instance.

        Args:
            validated_data (Any): A dictionary of validated data for the new
                Customer instance. This data should include the 'billing_profile'
                and 'contacts' fields, which are the CustomerBillingProfile and
                CustomerContact instances associated with the new Customer.

        Returns:
            Customer: The newly created Customer instance.
        """

        request = self.context["request"]
        if request.user.is_authenticated:
            organization = request.user.organization
        else:
            token = request.META.get("HTTP_AUTHORIZATION", "").split(" ")[1]
            organization = Token.objects.get(key=token).user.organization

        # Pop the billing profile and contacts from the validated data
        billing_profile_data = validated_data.pop("billing_profile", {})
        contacts_data = validated_data.pop("contacts", [])

        # Create the customer
        validated_data["organization"] = organization
        customer = models.Customer.objects.create(**validated_data)

        # Create the billing profile
        if billing_profile_data:
            email_profile_data = billing_profile_data.pop("email_profile", {})
            rule_profile_data = billing_profile_data.pop("rule_profile", {})

            # Create the customer billing profile
            if email_profile_data:
                email_profile_data["organization"] = organization
                email_profile_data = models.CustomerEmailProfile.objects.create(
                    **email_profile_data
                )

            # Create the billing profile
            if rule_profile_data:
                rule_profile_data["organization"] = organization
                rule_profile_data = models.CustomerRuleProfile.objects.create(
                    **rule_profile_data
                )

            # Billing profiles are automatically created from signals. However,
            # If passed, we have to delete the one that was created.
            customer_billing_profile = models.CustomerBillingProfile.objects.get(
                customer=customer
            )
            customer_billing_profile.delete()

            billing_profile_data["organization"] = organization
            models.CustomerBillingProfile.objects.create(
                customer=customer,
                email_profile=email_profile_data,
                rule_profile=rule_profile_data,
                **billing_profile_data,
            )

        # Create the contacts
        if contacts_data:
            contacts_data = [
                {**contact, "organization": organization} for contact in contacts_data
            ]
            contacts = [
                models.CustomerContact(customer=customer, **contact)
                for contact in contacts_data
            ]
            models.CustomerContact.objects.bulk_create(contacts)

        return customer

    def update(self, instance: models.Customer, validated_data: Any):
        """Update an existing Customer instance.

        Args:
            instance (Customer): The existing Customer instance to update.
            validated_data (dict): A dictionary of validated data for the updated
                Customer instance. This data should include the 'billing_profile'
                and 'contacts' fields, which are the updated values for the
                CustomerBillingProfile and CustomerContact instances associated
                with the Customer.

        Returns:
            Customer: The updated Customer instance.
        """

        billing_profile_data = validated_data.pop("billing_profile", None)
        contacts_data = validated_data.pop("contacts", None)

        # Use the update() method to update the instance with the validated data
        instance.update_customer(**validated_data)

        if billing_profile_data:
            rule_profile_data = billing_profile_data.pop("rule_profile", {})
            email_profile_data = billing_profile_data.pop("email_profile", {})

            if email_profile_data:
                # Use the update() method to update the email_profile instance with the data
                instance.billing_profile.email_profile.update_customer_email_profile(**email_profile_data)

            if rule_profile_data:
                # Use the update() method to update the rule_profile instance with the data
                instance.billing_profile.rule_profile.update_customer_rule_profile(**rule_profile_data)

                document_class = rule_profile_data.pop("document_class", [])

                if document_class:
                    instance.billing_profile.rule_profile.document_class = document_class
                    instance.billing_profile.rule_profile.save()

        if contacts_data:
            [contact.update_customer_contact(**contact_data) for contact, contact_data in zip(instance.contacts.all(), contacts_data)]

        return instance