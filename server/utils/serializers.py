# --------------------------------------------------------------------------------------------------
#  COPYRIGHT(c) 2024 Trenova                                                                       -
#                                                                                                  -
#  This file is part of Trenova.                                                                   -
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
from organization.models import BusinessUnit, Organization


class GenericSerializer(serializers.ModelSerializer):
    """
    Generic Serializer for handling common functionalities across models.
    """

    @cached_property
    def get_organization(self) -> Organization:
        request = self.context.get("request")
        if request and request.user.is_authenticated:
            return request.user.organization

        token = request.META.get("HTTP_AUTHORIZATION", "").split(" ")[1]
        return Token.objects.get(key=token).user.organization

    @cached_property
    def get_business_unit(self) -> BusinessUnit:
        request = self.context.get("request")
        if request and request.user.is_authenticated:
            return request.user.organization.business_unit

        token = request.META.get("HTTP_AUTHORIZATION", "").split(" ")[1]
        return Token.objects.get(key=token).user.organization.business_unit

    def create(self, validated_data):
        validated_data["organization"] = self.get_organization
        validated_data["business_unit"] = self.get_business_unit
        return super().create(validated_data)

    def update(self, instance, validated_data):
        validated_data["organization"] = self.get_organization
        validated_data["business_unit"] = self.get_business_unit
        return super().update(instance, validated_data)
