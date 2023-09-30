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
from django.utils.functional import cached_property
from rest_framework import serializers

from accounts.models import Token
from organization import models
from utils.serializers import GenericSerializer


class DepotDetailSerializer(serializers.ModelSerializer):
    """
    Serializer for the Depot model
    """

    class Meta:
        """
        Metaclass for the DepotDetailSerializer
        """

        model = models.DepotDetail
        fields = (
            "id",
            "organization",
            "depot",
            "address_line_1",
            "address_line_2",
            "city",
            "state",
            "zip_code",
            "phone_number",
            "alternate_phone_number",
            "fax_number",
            "created",
            "modified",
        )


class DepotSerializer(serializers.ModelSerializer):
    """Serializer for the Depot model"""

    details = DepotDetailSerializer()

    class Meta:
        """
        Metaclass for the DepotSerializer
        """

        model = models.Depot
        fields = (
            "id",
            "organization",
            "name",
            "description",
            "details",
        )

    @cached_property
    def get_organization(self) -> models.Organization:
        if self.context["request"].user.is_authenticated:
            _organization: models.Organization = self.context[
                "request"
            ].user.organization
            return _organization
        token = self.context["request"].META.get("HTTP_AUTHORIZATION", "").split(" ")[1]
        return Token.objects.get(key=token).user.organization

    def validate_name(self, value: str) -> str:
        """Validate the `name` field of the Depot model.

        This method validates the `name` field of the Depot model.
        It checks if the depot with the given name already exists in the organization.
        If the depot exists, it raises a validation error.

        Args:
            value (str): The value of the `name` field.

        Returns:
            str: The value of the `name` field.

        Raises:
            serializers.ValidationError: If the depot with the given name already exists in the
             organization.
        """
        organization = super().get_organization

        queryset = models.Depot.objects.filter(
            organization=organization,
            name__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance:
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Depot with this `name` already exists. Please try again."
            )

        return value


class OrganizationSerializer(serializers.ModelSerializer):
    """
    Organization Serializer
    """

    depots = serializers.PrimaryKeyRelatedField(  # type: ignore
        many=True,
        queryset=models.Depot.objects.all(),
        required=False,
        allow_null=True,
    )

    class Meta:
        """
        Metaclass for OrganizationSerializer
        """

        model = models.Organization
        fields = (
            "id",
            "name",
            "scac_code",
            "org_type",
            "timezone",
            "language",
            "currency",
            "date_format",
            "time_format",
            "logo",
            "depots",
        )


class DepartmentSerializer(GenericSerializer):
    """
    Department Serializer
    """

    class Meta:
        """
        Metaclass for Department
        """

        model = models.Department
        fields = (
            "id",
            "organization",
            "depot",
            "name",
            "description",
        )


class EmailControlSerializer(GenericSerializer):
    """
    Email Control Serializer
    """

    class Meta:
        """
        Metaclass for Email Control
        """

        model = models.EmailControl


class EmailProfileSerializer(GenericSerializer):
    """
    Email Profile Serializer
    """

    class Meta:
        """
        Metaclass for Email Profile
        """

        model = models.EmailProfile


class EmailLogSerializer(GenericSerializer):
    """
    Email Log Serializer
    """

    class Meta:
        """
        Metaclass for Email Log
        """

        model = models.EmailLog


class TaxRateSerializer(GenericSerializer):
    """
    Tax Rate Serializer
    """

    class Meta:
        """
        Metaclass for Tax Rate
        """

        model = models.TaxRate


class TableChangeAlertSerializer(GenericSerializer):
    """
    Table Change Alert Serializer
    """

    class Meta:
        """
        Metaclass for Table Change Alert
        """

        model = models.TableChangeAlert
        extra_read_only_fields = ("function_name", "trigger_name", "listener_name")

    def validate_name(self, value: str) -> str:
        """Validate the `name` field of the TableChangeAlert model.

        This method validates the `name` field of the TableChangeAlert model.
        It checks if the table change alert with the given name already exists in the organization.
        If the table change alert exists, it raises a validation error.

        Args:
            value (str): The value of the `name` field.

        Returns:
            str: The value of the `name` field.

        Raises:
            serializers.ValidationError: If the table change alert with the given name already exists in the
             organization.
        """
        organization = super().get_organization

        queryset = models.TableChangeAlert.objects.filter(
            organization=organization,
            name__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance:
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Table Change Alert with this `name` already exists. Please try again."
            )

        return value


class NotificationTypeSerializer(GenericSerializer):
    """
    Notification Types Serializer
    """

    class Meta:
        """
        Metaclass for Notification Types
        """

        model = models.NotificationType

    def validate_name(self, value: str) -> str:
        """Validate the `name` field of the NotificationType model.

        This method validates the `name` field of the NotificationType model.
        It checks if the notification type with the given name already exists in the organization.
        If the notification type exists, it raises a validation error.

        Args:
            value (str): The value of the `name` field.

        Returns:
            str: The value of the `name` field.

        Raises:
            serializers.ValidationError: If the notification type with the given name already exists in the
             organization.
        """
        organization = super().get_organization

        queryset = models.NotificationType.objects.filter(
            organization=organization,
            name__iexact=value,  # iexact performs a case-insensitive search
        )

        # Exclude the current instance if updating
        if self.instance:
            queryset = queryset.exclude(pk=self.instance.pk)

        if queryset.exists():
            raise serializers.ValidationError(
                "Notification Type with this `name` already exists. Please try again."
            )

        return value


class NotificationSettingSerializer(GenericSerializer):
    """
    Notification Settings Serializer
    """

    class Meta:
        """
        Metaclass for Notification Settings
        """

        model = models.NotificationSetting
