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

from rest_framework import serializers

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

    email_profile = serializers.PrimaryKeyRelatedField(
        queryset=models.EmailProfile.objects.all(),
        required=False,
        allow_null=True,
    )

    class Meta:
        """
        Metaclass for Table Change Alert
        """

        model = models.TableChangeAlert
        extra_fields = ("email_profile",)
        extra_read_only_fields = ("function_name", "trigger_name", "listener_name")
