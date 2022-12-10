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


class DepotDetailSerializer(serializers.ModelSerializer):
    """
    Serializer for the Depot model
    """

    class Meta:
        """
        Metaclass for the DepotDetailSerializer
        """

        model = models.DepotDetail
        fields = "__all__"


class DepotSerializer(serializers.ModelSerializer):
    """Serializer for the Depot model"""

    depot_details = DepotDetailSerializer()

    class Meta:
        """
        Metaclass for the DepotSerializer
        """

        model = models.Depot
        fields = [
            "id",
            "organization",
            "name",
            "description",
            "depot_details",
        ]


class OrganizationSerializer(serializers.ModelSerializer):
    """
    Organization Serializer
    """

    depots = DepotSerializer(many=True, read_only=True)

    class Meta:
        """
        Metaclass for OrganizationSerializer
        """

        model: type[models.Organization] = models.Organization
        fields: list[str] = [
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
            "authentication_bg",
            "authentication_template",
            "depots",
        ]


class DepartmentSerializer(serializers.ModelSerializer):
    """
    Department Serializer
    """

    class Meta:
        """
        Metaclass for Department
        """

        model: type[models.Department] = models.Department
        fields: list[str] = [
            "id",
            "organization",
            "depot",
            "name",
            "description",
        ]
