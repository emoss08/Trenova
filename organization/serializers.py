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
